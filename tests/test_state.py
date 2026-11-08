import pytest

from fps_booster.state.backup_store import BackupStore
from fps_booster.state.change_log import ChangeLog


def test_backup_keeps_first_original_and_persists(tmp_path):
    path = tmp_path / "b.json"
    store = BackupStore(path)
    assert store.remember_original("power_plan", "guid-a") is True
    assert store.remember_original("power_plan", "guid-b") is False
    assert BackupStore(path).get("power_plan") == "guid-a"  # sobrevive al reinicio
    store.forget("power_plan")
    assert not BackupStore(path).has("power_plan")


def test_backup_recovers_from_corruption(tmp_path):
    path = tmp_path / "b.json"
    path.write_text("{ esto no es json", encoding="utf-8")
    store = BackupStore(path)
    assert store.all() == {}
    assert list(tmp_path.glob("b.corrupt-*.json"))  # el archivo dañado se conserva


def test_change_log_order_and_bad_lines(tmp_path):
    log = ChangeLog(tmp_path / "c.jsonl", "simulacion")
    log.record("apply", "a", message="uno")
    with log.path.open("a", encoding="utf-8") as fh:
        fh.write("línea rota\n")
    log.record("error", "b", success=False, message="dos")
    entries = log.entries()
    assert [e["optimization_id"] for e in entries] == ["b", "a"]
    assert entries[0]["success"] is False


def test_change_log_rejects_unknown_action(tmp_path):
    with pytest.raises(ValueError):
        ChangeLog(tmp_path / "c.jsonl", "x").record("kill_process")
