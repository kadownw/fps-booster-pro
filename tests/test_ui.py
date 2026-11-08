"""Pruebas de interfaz en modo offscreen (sin ventana visible, sin hilos reales)."""

import time

import pytest

pytest.importorskip("PySide6")

from PySide6.QtWidgets import QApplication, QDialog  # noqa: E402

from conftest import FakeProc, make_hw  # noqa: E402
from fps_booster.context import build_context  # noqa: E402
from fps_booster.core.models import GpuSample, ResourceSample  # noqa: E402
from fps_booster.optimizations.engine import Consent  # noqa: E402
from fps_booster.processes.analyzer import ProcessAnalyzer  # noqa: E402
from fps_booster.system.backend import POWER_SCHEMES, SimulatedSystemBackend  # noqa: E402
from fps_booster.ui import theme  # noqa: E402
from fps_booster.ui.main_window import MainWindow  # noqa: E402
from fps_booster.ui.widgets.consent_dialog import ConsentDialog  # noqa: E402

pytestmark = pytest.mark.ui


@pytest.fixture(scope="module")
def app():
    instance = QApplication.instance() or QApplication([])
    instance.setStyleSheet(theme.QSS)
    return instance


@pytest.fixture
def window(app, tmp_path):
    ctx = build_context(data_dir=tmp_path)
    ctx.backend = SimulatedSystemBackend()  # sin lecturas reales en las pruebas de UI
    ctx.engine.backend = ctx.backend
    win = MainWindow(ctx, start_workers=False)
    win.notify = lambda title, text: None
    yield win
    win.close()


def sample(cpu=40.0, gpu=55.0):
    return ResourceSample(time.time(), cpu, (cpu,) * 8, 60.0, 8 * 1024**3, 16 * 1024**3,
                          1e6, 2e6, 1e3, 2e3, GpuSample(gpu, 2 * 1024**3, None, 70.0, "fake"))


def test_window_builds_and_updates(window, fake_psutil):
    window.on_hardware(make_hw())
    for _ in range(3):
        window.on_sample(sample())
    procs = ProcessAnalyzer(fake_psutil([FakeProc(5, "chrome.exe", 90.0, 2 * 1024**3)]), own_pid=1).snapshot()
    window.on_processes(procs)
    assert window.dashboard.cpu_card.value_label.text() == "40%"
    assert window.dashboard.gpu_card.value_label.text() == "55%"
    assert window.processes_page.table.rowCount() == 1
    assert window.recs_page.recommendations
    for i in range(len(window.pages)):
        window.go_to(i)
    window.grab()  # fuerza el pintado de todos los widgets


def test_gpu_not_available_shows_nd(window):
    s = ResourceSample(time.time(), 10, (10,), 10, 1, 2, 0, 0, 0, 0, None)
    window.on_sample(s)
    assert window.dashboard.gpu_card.value_label.text() == "N/D"


def test_cancelled_consent_changes_nothing(window):
    window.consent_provider = lambda title, plan: None
    assert window.review_profile("gaming") is None
    assert window.ctx.backend.operations == []


def test_optimize_flow_with_consent_and_revert(window, monkeypatch):
    window.consent_provider = lambda title, plan: Consent.grant(
        [c.id for c in plan if c.selected and c.actionable], source="test")
    report = window.review_profile("gaming")
    assert report.ok and report.simulated
    assert window.ctx.backend.get_active_power_scheme() == POWER_SCHEMES["high_performance"]
    assert window.history_page.revert_btn.isEnabled()

    monkeypatch.setattr(window, "consent_for_revert", lambda pending: Consent.grant([p["id"] for p in pending]))
    assert window.revert_all().ok
    assert window.ctx.backend.get_active_power_scheme() == POWER_SCHEMES["balanced"]
    assert not window.history_page.revert_btn.isEnabled()


def test_consent_dialog_requires_ack(app, engine):
    from fps_booster.profiles.profiles import GAMING

    plan = engine.plan_profile(GAMING)
    dlg = ConsentDialog("Perfil Gaming", plan, simulated=True)
    assert not dlg.apply_btn.isEnabled()
    dlg.ack.setChecked(True)
    assert dlg.apply_btn.isEnabled()
    dlg._boxes["game_dvr"].setChecked(False)
    dlg.done(QDialog.DialogCode.Accepted)
    consent = dlg.result_consent()
    assert consent.approved_ids == {"power_plan", "game_mode"}
    assert next(c for c in plan if c.id == "game_dvr").selected is False


def test_consent_dialog_without_ack_returns_none(app, engine):
    from fps_booster.profiles.profiles import GAMING

    dlg = ConsentDialog("x", engine.plan_profile(GAMING), simulated=True)
    dlg.done(QDialog.DialogCode.Accepted)
    assert dlg.result_consent() is None
