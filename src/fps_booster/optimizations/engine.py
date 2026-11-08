"""Motor transaccional de optimizaciones.

Garantías:
- Sin ``Consent`` explícito que cubra cada cambio → ``ConsentRequiredError``. Nada se toca.
- Antes de cada cambio se guarda el valor original (solo la primera vez).
- Después de cada cambio se verifica leyendo el estado.
- Si un cambio falla, se revierte todo el lote (rollback).
"""

from __future__ import annotations

from collections.abc import Iterable
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from fps_booster.core.logging_setup import get_logger
from fps_booster.optimizations.base import Optimization, OptimizationError
from fps_booster.profiles.profiles import Profile, ProfileSetting
from fps_booster.state.backup_store import BackupStore
from fps_booster.state.change_log import ChangeLog
from fps_booster.system.backend import SystemBackend

log = get_logger("engine")


class ConsentRequiredError(PermissionError):
    pass


@dataclass(frozen=True)
class Consent:
    """Autorización explícita del usuario. Solo la crea el diálogo de consentimiento (o un test)."""

    approved_ids: frozenset[str]
    granted_at: str
    source: str = "dialogo_usuario"

    @classmethod
    def grant(cls, ids: Iterable[str], source: str = "dialogo_usuario") -> Consent:
        return cls(frozenset(ids), datetime.now().astimezone().isoformat(timespec="seconds"), source)


@dataclass
class PlannedChange:
    optimization: Optimization
    target: str
    current: Any
    available: bool
    already_applied: bool
    selected: bool
    optional: bool = False
    note: str = ""
    current_label: str = ""
    target_label: str = ""

    @property
    def id(self) -> str:
        return self.optimization.id

    @property
    def actionable(self) -> bool:
        return self.available and not self.already_applied


@dataclass
class ChangeResult:
    optimization_id: str
    success: bool
    message: str
    before: Any = None
    after: Any = None


@dataclass
class ApplyReport:
    results: list[ChangeResult] = field(default_factory=list)
    rolled_back: bool = False
    simulated: bool = True

    @property
    def ok(self) -> bool:
        return not self.rolled_back and all(r.success for r in self.results)

    @property
    def applied_count(self) -> int:
        return sum(1 for r in self.results if r.success)


class OptimizationEngine:
    def __init__(
        self,
        backend: SystemBackend,
        backups: BackupStore,
        change_log: ChangeLog,
        catalog: dict[str, Optimization],
    ):
        self.backend = backend
        self.backups = backups
        self.log = change_log
        self.catalog = catalog

    @property
    def simulated(self) -> bool:
        return self.backend.is_simulation

    # ------------------------------------------------------------------ plan
    def plan(self, settings: Iterable[ProfileSetting], preselect: Iterable[str] = ()) -> list[PlannedChange]:
        """Lee el estado actual. No modifica nada."""
        preselect = set(preselect)
        changes: list[PlannedChange] = []
        for s in settings:
            opt = self.catalog.get(s.optimization_id)
            if opt is None:
                log.warning("Optimización desconocida en plan: %s", s.optimization_id)
                continue
            try:
                current = opt.read(self.backend)
                available, note = opt.availability(self.backend, s.target)
                already = opt.matches(current, s.target)
            except Exception as exc:  # noqa: BLE001
                log.exception("Error leyendo %s", opt.id)
                current, available, already, note = None, False, False, f"Error al leer: {exc}"
            if already and not note:
                note = "Ya está aplicado"
            changes.append(
                PlannedChange(
                    optimization=opt,
                    target=s.target,
                    current=current,
                    available=available,
                    already_applied=already,
                    selected=available and not already and (not s.optional or opt.id in preselect),
                    optional=s.optional,
                    note=note or s.reason,
                    current_label=opt.describe_state(current, self.backend),
                    target_label=opt.describe_target(s.target, self.backend),
                )
            )
        return changes

    def plan_profile(self, profile: Profile, preselect: Iterable[str] = ()) -> list[PlannedChange]:
        return self.plan(profile.settings, preselect)

    # ------------------------------------------------------------------ apply
    def apply(self, plan: list[PlannedChange], consent: Consent | None) -> ApplyReport:
        to_apply = [c for c in plan if c.selected and c.actionable]
        self._check_consent(consent, [c.id for c in to_apply])
        report = ApplyReport(simulated=self.simulated)
        done: list[tuple[Optimization, Any]] = []

        for change in to_apply:
            opt = change.optimization
            before = None
            try:
                before = opt.read(self.backend)
                self.backups.remember_original(opt.id, before)
                done.append((opt, before))  # se incluye antes de aplicar: si falla a medias, se restaura
                opt.apply(self.backend, change.target)
                after = opt.read(self.backend)
                if not opt.matches(after, change.target):
                    raise OptimizationError("La verificación posterior no coincide con lo esperado")
            except Exception as exc:  # noqa: BLE001
                msg = f"{opt.title}: {exc}"
                self.log.record("error", opt.id, success=False, before=before, message=msg)
                report.results.append(ChangeResult(opt.id, False, msg, before))
                self._rollback(done, report)
                return report
            self.log.record(
                "apply", opt.id, before=before, after=after,
                message=f"{opt.title}: {change.current_label} → {change.target_label}",
            )
            report.results.append(ChangeResult(opt.id, True, f"{opt.title}: aplicado", before, after))
        return report

    def _rollback(self, done: list[tuple[Optimization, Any]], report: ApplyReport) -> None:
        report.rolled_back = True
        for opt, before in reversed(done):
            try:
                opt.restore(self.backend, before)
                self.log.record("rollback", opt.id, after=before, message=f"{opt.title}: restaurado")
                if self.backups.has(opt.id) and opt.same_state(self.backups.get(opt.id), before):
                    self.backups.forget(opt.id)  # el sistema volvió al original: nada pendiente
            except Exception as exc:  # noqa: BLE001
                self.log.record("error", opt.id, success=False, message=f"Rollback falló: {exc}")

    # ------------------------------------------------------------------ revert
    def pending_reverts(self) -> list[dict[str, Any]]:
        out = []
        for opt_id, original in self.backups.all().items():
            opt = self.catalog.get(opt_id)
            if opt is None:
                continue
            current = opt.read(self.backend)
            out.append(
                {
                    "id": opt_id,
                    "title": opt.title,
                    "original": original,
                    "original_label": opt.describe_state(original, self.backend),
                    "current_label": opt.describe_state(current, self.backend),
                    "differs": not opt.same_state(current, original),
                }
            )
        return out

    def revert_all(self, consent: Consent | None) -> ApplyReport:
        ids = list(self.backups.all())
        self._check_consent(consent, ids)
        report = ApplyReport(simulated=self.simulated)
        for opt_id in ids:
            opt = self.catalog.get(opt_id)
            original = self.backups.get(opt_id)
            if opt is None:
                msg = f"Optimización desconocida en backup: {opt_id}"
                self.log.record("error", opt_id, success=False, message=msg)
                report.results.append(ChangeResult(opt_id, False, msg))
                continue
            try:
                before = opt.read(self.backend)
                opt.restore(self.backend, original)
                after = opt.read(self.backend)
                if not opt.same_state(after, original):
                    raise OptimizationError("El valor restaurado no coincide con el original")
            except Exception as exc:  # noqa: BLE001
                msg = f"{opt.title}: no se pudo revertir ({exc})"
                self.log.record("error", opt_id, success=False, message=msg)
                report.results.append(ChangeResult(opt_id, False, msg))
                continue
            self.backups.forget(opt_id)
            self.log.record("revert", opt_id, before=before, after=after,
                            message=f"{opt.title}: restaurado a {opt.describe_state(original, self.backend)}")
            report.results.append(ChangeResult(opt_id, True, f"{opt.title}: revertido", before, after))
        return report

    # ------------------------------------------------------------------ helpers
    @staticmethod
    def _check_consent(consent: Consent | None, ids: list[str]) -> None:
        if not ids:
            return
        if consent is None:
            raise ConsentRequiredError("Se requiere consentimiento explícito del usuario")
        missing = [i for i in ids if i not in consent.approved_ids]
        if missing:
            raise ConsentRequiredError(f"Sin consentimiento para: {', '.join(missing)}")
