"""Ventana principal: navegación + orquestación de páginas, workers y motor."""

from __future__ import annotations

from collections.abc import Callable

from PySide6.QtCore import QThread
from PySide6.QtWidgets import (
    QButtonGroup,
    QFrame,
    QHBoxLayout,
    QLabel,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QStackedWidget,
    QVBoxLayout,
    QWidget,
)

from fps_booster import APP_NAME, __version__
from fps_booster.context import AppContext
from fps_booster.core.logging_setup import get_logger
from fps_booster.core.models import HardwareInfo, ProcessInfo, ResourceSample
from fps_booster.optimizations.engine import ApplyReport, Consent, ConsentRequiredError, PlannedChange
from fps_booster.processes.analyzer import heavy_candidates
from fps_booster.profiles.profiles import PROFILES, ProfileSetting
from fps_booster.recommendations.recommender import classify_tier, optional_preselection, recommend
from fps_booster.ui.pages.dashboard import DashboardPage
from fps_booster.ui.pages.history_page import HistoryPage
from fps_booster.ui.pages.processes import ProcessesPage
from fps_booster.ui.pages.profiles_page import ProfilesPage
from fps_booster.ui.pages.recommendations_page import RecommendationsPage
from fps_booster.ui.pages.system_page import SystemPage
from fps_booster.ui.widgets.consent_dialog import ConsentDialog
from fps_booster.ui.workers import HardwareThread, MonitorWorker

log = get_logger("ui")

ConsentProvider = Callable[[str, list[PlannedChange]], Consent | None]


class MainWindow(QMainWindow):
    def __init__(self, ctx: AppContext, start_workers: bool = True):
        super().__init__()
        self.ctx = ctx
        self.hardware: HardwareInfo | None = None
        self.last_sample: ResourceSample | None = None
        self.last_processes: list[ProcessInfo] = []
        self.consent_provider: ConsentProvider = self._ask_consent
        self.notify: Callable[[str, str], None] = self._message
        self._monitor_thread: QThread | None = None
        self._hw_thread: HardwareThread | None = None

        self.setWindowTitle(f"{APP_NAME} {__version__}" + (" — SIMULACIÓN" if ctx.is_simulation else ""))
        self.resize(1360, 860)
        self.setMinimumSize(1120, 720)

        root = QWidget()
        root.setObjectName("Root")
        h = QHBoxLayout(root)
        h.setContentsMargins(0, 0, 0, 0)
        h.setSpacing(0)
        h.addWidget(self._build_sidebar())

        right = QVBoxLayout()
        right.setContentsMargins(0, 0, 0, 0)
        right.setSpacing(0)
        right.addWidget(self._build_header())
        self.stack = QStackedWidget()
        right.addWidget(self.stack, 1)
        h.addLayout(right, 1)
        self.setCentralWidget(root)

        self.dashboard = DashboardPage()
        self.processes_page = ProcessesPage()
        self.profiles_page = ProfilesPage()
        self.recs_page = RecommendationsPage()
        self.system_page = SystemPage()
        self.history_page = HistoryPage()
        self.pages = [
            ("Panel", self.dashboard), ("Procesos", self.processes_page),
            ("Perfiles", self.profiles_page), ("Recomendaciones", self.recs_page),
            ("Sistema", self.system_page), ("Historial", self.history_page),
        ]
        for i, (name, page) in enumerate(self.pages):
            self.stack.addWidget(page)
            btn = QPushButton(name)
            btn.setObjectName("NavButton")
            btn.setCheckable(True)
            btn.clicked.connect(lambda _=False, idx=i: self.go_to(idx))
            self._nav_group.addButton(btn, i)
            self._nav_layout.insertWidget(i, btn)
        self._nav_group.button(0).setChecked(True)

        self.dashboard.optimize_requested.connect(self.review_profile)
        self.profiles_page.review_requested.connect(self.review_profile)
        self.recs_page.review_requested.connect(self.review_recommendations)
        self.history_page.revert_requested.connect(self.revert_all)
        self.refresh_history()

        if start_workers:
            self.start_workers()

    # ------------------------------------------------------------- layout
    def _build_sidebar(self) -> QFrame:
        side = QFrame()
        side.setObjectName("Sidebar")
        side.setFixedWidth(220)
        v = QVBoxLayout(side)
        v.setContentsMargins(0, 26, 0, 18)
        v.setSpacing(2)
        logo = QLabel("FPS BOOSTER")
        logo.setObjectName("Logo")
        sub = QLabel("PERFORMANCE SUITE")
        sub.setObjectName("LogoAccent")
        for w in (logo, sub):
            w.setContentsMargins(20, 0, 0, 0)
            v.addWidget(w)
        v.addSpacing(26)
        self._nav_group = QButtonGroup(self)
        self._nav_group.setExclusive(True)
        nav = QWidget()
        self._nav_layout = QVBoxLayout(nav)
        self._nav_layout.setContentsMargins(0, 0, 0, 0)
        self._nav_layout.setSpacing(2)
        self._nav_layout.addStretch(1)
        v.addWidget(nav, 1)
        ver = QLabel(f"v{__version__} · Fase 1")
        ver.setObjectName("Muted")
        ver.setContentsMargins(20, 0, 0, 0)
        v.addWidget(ver)
        return side

    def _build_header(self) -> QFrame:
        bar = QFrame()
        bar.setFixedHeight(56)
        h = QHBoxLayout(bar)
        h.setContentsMargins(28, 12, 28, 0)
        self.header_title = QLabel("Panel")
        self.header_title.setObjectName("PageSubtitle")
        h.addWidget(self.header_title)
        h.addStretch(1)
        self.gpu_badge = QLabel("GPU: detectando…")
        self.gpu_badge.setObjectName("Badge")
        h.addWidget(self.gpu_badge)
        mode = QLabel("MODO SIMULACIÓN · tu PC no se modifica" if self.ctx.is_simulation
                      else "MODO REAL · los cambios afectan a Windows")
        mode.setObjectName("ModeSim" if self.ctx.is_simulation else "ModeReal")
        h.addWidget(mode)
        return bar

    def go_to(self, index: int) -> None:
        self.stack.setCurrentIndex(index)
        self.header_title.setText(self.pages[index][0])
        btn = self._nav_group.button(index)
        if btn is not None:
            btn.setChecked(True)
        if self.pages[index][1] is self.history_page:
            self.refresh_history()

    # ------------------------------------------------------------- workers
    def start_workers(self) -> None:
        self._hw_thread = HardwareThread(self.ctx.detector_factory, self)
        self._hw_thread.detected.connect(self.on_hardware)
        self._hw_thread.failed.connect(lambda msg: self.dashboard.set_status(f"Error detectando hardware: {msg}"))
        self._hw_thread.start()

        self._monitor_thread = QThread(self)
        self._monitor = MonitorWorker(self.ctx.sampler_factory, self.ctx.analyzer_factory)
        self._monitor.moveToThread(self._monitor_thread)
        self._monitor_thread.started.connect(self._monitor.start)
        self._monitor_thread.finished.connect(self._monitor.deleteLater)
        self._monitor.sample_ready.connect(self.on_sample)
        self._monitor.processes_ready.connect(self.on_processes)
        self._monitor.gpu_source_ready.connect(self.on_gpu_source)
        self._monitor_thread.start()

    def stop_workers(self) -> None:
        if self._monitor_thread is not None and self._monitor_thread.isRunning():
            self._monitor_thread.quit()
            self._monitor_thread.wait(3000)
        if self._hw_thread is not None and self._hw_thread.isRunning():
            self._hw_thread.wait(5000)

    def closeEvent(self, event) -> None:  # noqa: N802
        self.stop_workers()
        super().closeEvent(event)

    # ------------------------------------------------------------- data slots
    def on_gpu_source(self, source: str) -> None:
        self.gpu_badge.setText(f"GPU: {source}")
        self.system_page.gpu_source = source
        if self.hardware is not None:
            self.system_page.set_hardware(self.hardware, classify_tier(self.hardware))

    def on_hardware(self, hw: HardwareInfo) -> None:
        self.hardware = hw
        tier = classify_tier(hw)
        self.dashboard.set_hardware(hw, tier)
        self.system_page.set_hardware(hw, tier)
        self.refresh_recommendations()

    def on_sample(self, sample: ResourceSample) -> None:
        self.last_sample = sample
        self.dashboard.update_sample(sample)

    def on_processes(self, processes: list[ProcessInfo]) -> None:
        first = not self.last_processes
        self.last_processes = processes
        self.dashboard.update_processes(processes)
        self.processes_page.update_processes(processes)
        if first and self.hardware is not None:
            self.refresh_recommendations()

    def refresh_recommendations(self) -> None:
        if self.hardware is None:
            return
        recs = recommend(self.hardware, self.last_sample, heavy_candidates(self.last_processes))
        self.recs_page.set_recommendations(recs, classify_tier(self.hardware))

    def refresh_history(self) -> None:
        try:
            self.history_page.set_pending(self.ctx.engine.pending_reverts())
        except Exception:  # noqa: BLE001
            log.exception("No se pudieron leer los cambios pendientes")
        self.history_page.set_entries(self.ctx.change_log.entries(), str(self.ctx.change_log.path))

    # ------------------------------------------------------------- acciones
    def _ask_consent(self, title: str, plan: list[PlannedChange]) -> Consent | None:
        dialog = ConsentDialog(title, plan, self.ctx.is_simulation, self)
        dialog.exec()
        return dialog.result_consent()

    def _message(self, title: str, text: str) -> None:
        QMessageBox.information(self, title, text)

    def review_profile(self, profile_id: str) -> ApplyReport | None:
        profile = PROFILES[profile_id]
        plan = self.ctx.engine.plan_profile(profile, optional_preselection(self.hardware))
        return self._review(f"Perfil {profile.name}", plan)

    def review_recommendations(self) -> ApplyReport | None:
        settings = [ProfileSetting(r.optimization_id, r.target)
                    for r in self.recs_page.recommendations if r.automatic and r.target]
        return self._review("Ajustes recomendados", self.ctx.engine.plan(settings))

    def _review(self, title: str, plan: list[PlannedChange]) -> ApplyReport | None:
        if not any(c.actionable for c in plan):
            self.notify(APP_NAME, "No hay cambios para aplicar: todo ya está configurado o no disponible.")
            return None
        consent = self.consent_provider(title, plan)
        if consent is None:
            self.dashboard.set_status("Cancelado. No se modificó nada.")
            return None
        try:
            report = self.ctx.engine.apply(plan, consent)
        except ConsentRequiredError as exc:
            self.notify(APP_NAME, str(exc))
            return None
        self._show_report("Resultado", report)
        return report

    def revert_all(self) -> ApplyReport | None:
        pending = self.ctx.engine.pending_reverts()
        if not pending:
            self.notify(APP_NAME, "No hay cambios para revertir.")
            return None
        consent = self.consent_for_revert(pending)
        if consent is None:
            return None
        report = self.ctx.engine.revert_all(consent)
        self._show_report("Reversión", report)
        return report

    def consent_for_revert(self, pending: list[dict]) -> Consent | None:
        names = "\n".join(f"• {p['title']} → {p['original_label']}" for p in pending)
        answer = QMessageBox.question(
            self, "Revertir cambios",
            f"Se restaurarán los valores originales:\n\n{names}\n\n¿Continuar?",
        )
        if answer != QMessageBox.StandardButton.Yes:
            return None
        return Consent.grant([p["id"] for p in pending])

    def _show_report(self, title: str, report: ApplyReport) -> None:
        prefix = "[Simulación] " if report.simulated else ""
        lines = [("✔ " if r.success else "✖ ") + r.message for r in report.results]
        if report.rolled_back:
            lines.append("\nHubo un error: se deshicieron los cambios de este lote.")
        status = (f"{prefix}{report.applied_count} cambio(s) aplicados." if report.ok
                  else f"{prefix}Hubo errores. Revisá Historial.")
        self.dashboard.set_status(status + " Medí tus FPS para comprobar el efecto real.")
        self.refresh_history()
        self.notify(f"{prefix}{title}", "\n".join(lines) or "Sin cambios.")
