"""Contrato de una optimización reversible."""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any

from fps_booster.system.backend import POWER_SCHEME_NAMES, POWER_SCHEMES, SystemBackend


class OptimizationError(Exception):
    pass


class OptimizationUnavailable(OptimizationError):
    pass


class Optimization(ABC):
    """Cada optimización sabe leer su estado, aplicarlo y restaurar un estado anterior.

    Los estados son JSON-serializables para poder guardarse en el backup.
    """

    id: str
    title: str
    description: str
    risk: str = "bajo"
    requires_admin: bool = False
    requires_restart: bool = False
    reversible: bool = True

    @abstractmethod
    def read(self, backend: SystemBackend) -> Any: ...

    @abstractmethod
    def resolve_target(self, target: str) -> Any:
        """Convierte un objetivo simbólico ("on", "high_performance") al estado concreto."""

    @abstractmethod
    def write_state(self, backend: SystemBackend, state: Any) -> None: ...

    def apply(self, backend: SystemBackend, target: str) -> None:
        self.write_state(backend, self.resolve_target(target))

    def restore(self, backend: SystemBackend, original: Any) -> None:
        self.write_state(backend, original)

    def same_state(self, a: Any, b: Any) -> bool:
        return a == b

    def matches(self, current: Any, target: str) -> bool:
        return self.same_state(current, self.resolve_target(target))

    def availability(self, backend: SystemBackend, target: str) -> tuple[bool, str]:
        return True, ""

    def describe_state(self, state: Any, backend: SystemBackend | None = None) -> str:
        return str(state)

    def describe_target(self, target: str, backend: SystemBackend | None = None) -> str:
        return self.describe_state(self.resolve_target(target), backend)


class PowerPlanOptimization(Optimization):
    id = "power_plan"
    title = "Plan de energía"
    description = (
        "Cambia el plan de energía de Windows. 'Alto rendimiento' evita que la CPU baje su "
        "frecuencia durante el juego; consume más energía y genera más calor."
    )
    risk = "bajo"

    def read(self, backend: SystemBackend) -> str | None:
        guid = backend.get_active_power_scheme()
        return guid.lower() if guid else None

    def resolve_target(self, target: str) -> str:
        if target in POWER_SCHEMES:
            return POWER_SCHEMES[target]
        return str(target).lower()

    def write_state(self, backend: SystemBackend, state: Any) -> None:
        if not state:
            raise OptimizationError("No hay plan de energía para restaurar")
        guid = str(state).lower()
        if guid not in backend.list_power_schemes():
            raise OptimizationUnavailable(f"El plan {self.describe_state(guid)} no existe")
        backend.set_active_power_scheme(guid)

    def same_state(self, a: Any, b: Any) -> bool:
        return str(a or "").lower() == str(b or "").lower()

    def availability(self, backend: SystemBackend, target: str) -> tuple[bool, str]:
        guid = self.resolve_target(target)
        if guid in backend.list_power_schemes():
            return True, ""
        return False, (
            f"El plan '{self.describe_state(guid)}' no está disponible en este equipo "
            "(habitual en portátiles con Modern Standby)."
        )

    def describe_state(self, state: Any, backend: SystemBackend | None = None) -> str:
        guid = str(state or "").lower()
        if not guid:
            return "Desconocido"
        if guid in POWER_SCHEME_NAMES:
            return POWER_SCHEME_NAMES[guid]
        if backend is not None:
            name = backend.list_power_schemes().get(guid)
            if name:
                return name
        return f"Personalizado ({guid[:8]}…)"


@dataclass(frozen=True)
class RegistryValue:
    hive: str
    key: str
    name: str


@dataclass
class RegistryOptimization(Optimization):
    """Uno o varios DWORD de HKCU. Estado = {nombre: valor | None(no existe)}."""

    id: str
    title: str
    description: str
    values: tuple[RegistryValue, ...]
    states: dict[str, dict[str, int]]
    labels: dict[str, str] = field(default_factory=dict)
    risk: str = "bajo"
    requires_restart: bool = False

    def read(self, backend: SystemBackend) -> dict[str, int | None]:
        out: dict[str, int | None] = {}
        for v in self.values:
            raw = backend.read_registry_value(v.hive, v.key, v.name)
            out[v.name] = int(raw) if isinstance(raw, int) else None
        return out

    def resolve_target(self, target: str) -> dict[str, int | None]:
        if target not in self.states:
            raise OptimizationError(f"Objetivo desconocido para {self.id}: {target}")
        return dict(self.states[target])

    def write_state(self, backend: SystemBackend, state: Any) -> None:
        if not isinstance(state, dict):
            raise OptimizationError("Estado de registro inválido")
        for v in self.values:
            if v.name not in state:
                continue
            value = state[v.name]
            if value is None:
                backend.delete_registry_value(v.hive, v.key, v.name)
            else:
                backend.write_registry_dword(v.hive, v.key, v.name, int(value))

    def same_state(self, a: Any, b: Any) -> bool:
        if not isinstance(a, dict) or not isinstance(b, dict):
            return False
        return all(a.get(k) == b.get(k) for k in b)

    def describe_state(self, state: Any, backend: SystemBackend | None = None) -> str:
        if isinstance(state, dict):
            for key, values in self.states.items():
                if self.same_state(state, values):
                    return self.labels.get(key, key)
            if all(v is None for v in state.values()):
                return "Predeterminado de Windows"
            detail = ", ".join(f"{k}={'sin definir' if v is None else v}" for k, v in state.items())
            return f"Mixto ({detail})"
        return str(state)
