import json

import pytest

from fps_booster.optimizations.base import OptimizationError
from fps_booster.optimizations.catalog import GAME_DVR
from fps_booster.optimizations.engine import Consent, ConsentRequiredError
from fps_booster.profiles.profiles import BALANCED, GAMING, PROFILES, SAVER, ProfileSetting
from fps_booster.state.backup_store import BackupStore
from fps_booster.system.backend import POWER_SCHEMES


def approve(plan):
    return Consent.grant([c.id for c in plan if c.selected and c.actionable], source="test")


def test_profiles_exist_and_reference_catalog(engine):
    assert set(PROFILES) == {"gaming", "balanced", "saver"}
    for profile in PROFILES.values():
        for s in profile.settings:
            assert s.optimization_id in engine.catalog
            engine.catalog[s.optimization_id].resolve_target(s.target)  # objetivo válido


def test_plan_reads_without_modifying(engine, sim_backend):
    plan = engine.plan_profile(GAMING)
    assert sim_backend.operations == []
    by_id = {c.id: c for c in plan}
    assert by_id["power_plan"].current_label == "Equilibrado"
    assert by_id["power_plan"].target_label == "Alto rendimiento"
    assert by_id["transparency"].selected is False  # opcional → desmarcado por defecto
    assert engine.plan_profile(GAMING, preselect={"transparency"})[3].selected is True


def test_apply_requires_consent(engine, sim_backend):
    plan = engine.plan_profile(GAMING)
    with pytest.raises(ConsentRequiredError):
        engine.apply(plan, None)
    with pytest.raises(ConsentRequiredError):
        engine.apply(plan, Consent.grant(["power_plan"]))  # consentimiento parcial
    assert sim_backend.operations == []
    assert engine.backups.all() == {}


def test_apply_and_revert_roundtrip(engine, sim_backend):
    plan = engine.plan_profile(GAMING)
    report = engine.apply(plan, approve(plan))
    assert report.ok and report.simulated and report.applied_count == 3
    assert sim_backend.get_active_power_scheme() == POWER_SCHEMES["high_performance"]
    assert GAME_DVR.read(sim_backend) == {"GameDVR_Enabled": 0, "AppCaptureEnabled": 0}

    originals = engine.backups.all()
    assert originals["power_plan"] == POWER_SCHEMES["balanced"]
    assert originals["game_mode"] == {"AutoGameModeEnabled": 0}
    assert originals["game_dvr"] == {"GameDVR_Enabled": None, "AppCaptureEnabled": None}

    pending = engine.pending_reverts()
    assert all(p["differs"] for p in pending)
    rev = engine.revert_all(Consent.grant(originals))
    assert rev.ok
    assert sim_backend.get_active_power_scheme() == POWER_SCHEMES["balanced"]
    assert GAME_DVR.read(sim_backend) == {"GameDVR_Enabled": None, "AppCaptureEnabled": None}  # borrados
    assert engine.backups.all() == {}


def test_revert_requires_consent(engine):
    plan = engine.plan_profile(GAMING)
    engine.apply(plan, approve(plan))
    with pytest.raises(ConsentRequiredError):
        engine.revert_all(None)
    assert engine.backups.all()


def test_original_is_kept_across_profiles(engine, sim_backend):
    for profile in (GAMING, SAVER, BALANCED):
        plan = engine.plan_profile(profile)
        assert engine.apply(plan, approve(plan)).ok
    assert engine.backups.get("power_plan") == POWER_SCHEMES["balanced"]  # el de antes de FPS BOOSTER
    sim_backend.set_active_power_scheme(POWER_SCHEMES["power_saver"])
    engine.revert_all(Consent.grant(engine.backups.all()))
    assert sim_backend.get_active_power_scheme() == POWER_SCHEMES["balanced"]


def test_already_applied_is_not_reapplied(engine, sim_backend):
    plan = engine.plan_profile(BALANCED)
    power = next(c for c in plan if c.id == "power_plan")
    assert power.already_applied and not power.selected
    engine.apply(plan, approve(plan))
    assert "powercfg" not in " ".join(sim_backend.operations)


def test_unavailable_scheme_is_marked(tmp_path):
    from fps_booster.optimizations.catalog import default_catalog
    from fps_booster.optimizations.engine import OptimizationEngine
    from fps_booster.state.change_log import ChangeLog
    from fps_booster.system.backend import SimulatedSystemBackend

    modern_standby = SimulatedSystemBackend(schemes={POWER_SCHEMES["balanced"]: "Balanced"})
    eng = OptimizationEngine(modern_standby, BackupStore(tmp_path / "b.json"),
                             ChangeLog(tmp_path / "c.jsonl", "simulacion"), default_catalog())
    power = eng.plan_profile(GAMING)[0]
    assert not power.available and not power.selected
    assert "no está disponible" in power.note


def test_failure_rolls_back_whole_batch(engine, sim_backend, monkeypatch):
    plan = engine.plan_profile(GAMING)

    def fail(backend, target):
        backend.write_registry_dword("HKCU", r"System\GameConfigStore", "GameDVR_Enabled", 0)
        raise OptimizationError("fallo simulado a mitad de camino")

    monkeypatch.setattr(GAME_DVR, "apply", fail)
    report = engine.apply(plan, approve(plan))
    assert report.rolled_back and not report.ok
    assert sim_backend.get_active_power_scheme() == POWER_SCHEMES["balanced"]
    assert GAME_DVR.read(sim_backend)["GameDVR_Enabled"] is None  # el cambio parcial también se deshizo
    assert engine.backups.all() == {}
    actions = [e["action"] for e in engine.log.entries()]
    assert "error" in actions and "rollback" in actions


def test_verification_failure_triggers_rollback(engine, sim_backend, monkeypatch):
    plan = engine.plan_profile(GAMING)
    monkeypatch.setattr(GAME_DVR, "apply", lambda backend, target: None)  # "aplica" sin efecto
    report = engine.apply(plan, approve(plan))
    assert report.rolled_back
    assert "verificación" in report.results[-1].message


def test_change_log_records_apply(engine):
    plan = engine.plan_profile(GAMING)
    engine.apply(plan, approve(plan))
    entries = engine.log.entries()
    assert len(entries) == 3
    assert all(e["action"] == "apply" and e["mode"] == "simulacion" for e in entries)
    lines = engine.log.path.read_text(encoding="utf-8").strip().splitlines()
    assert all(json.loads(line)["timestamp"] for line in lines)


def test_plan_ignores_unknown_optimization(engine):
    assert engine.plan([ProfileSetting("does_not_exist", "on")]) == []
