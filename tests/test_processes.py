from conftest import FakeProc
from fps_booster.processes.analyzer import ProcessAnalyzer, Thresholds, classify, heavy_candidates

MB = 1024 * 1024


def make_analyzer(fake_psutil, procs):
    return ProcessAnalyzer(psutil_mod=fake_psutil(procs, logical=4), thresholds=Thresholds(15, 800),
                           own_pid=999)


def test_snapshot_normalizes_cpu_and_sorts(fake_psutil):
    procs = [
        FakeProc(10, "chrome.exe", 80.0, 1200 * MB),   # 80/4 = 20%
        FakeProc(11, "notepad.exe", 4.0, 20 * MB),
        FakeProc(4, "System", 100.0, 1 * MB),
        FakeProc(0, "System Idle Process", 300.0, 0),
    ]
    snap = make_analyzer(fake_psutil, procs).snapshot()
    assert [p.pid for p in snap] == [4, 10, 11]  # pid 0 excluido
    chrome = next(p for p in snap if p.pid == 10)
    assert chrome.cpu_percent == 20.0
    assert chrome.heavy and chrome.category == "segundo_plano"
    assert any("RAM alta" in r for r in chrome.reasons)


def test_heavy_candidates_exclude_system_and_self(fake_psutil):
    procs = [
        FakeProc(4, "System", 400.0, 0),
        FakeProc(20, "dwm.exe", 100.0, 900 * MB),
        FakeProc(999, "python.exe", 100.0, 900 * MB),
        FakeProc(30, "game.exe", 100.0, 4000 * MB),
    ]
    snap = make_analyzer(fake_psutil, procs).snapshot()
    assert [p.name for p in heavy_candidates(snap)] == ["game.exe"]


def test_classify():
    assert classify("svchost.exe", 1234, 1) == "protegido"
    assert classify("Discord.exe", 1234, 1) == "segundo_plano"
    assert classify("x.exe", 1, 1) == "propio"
    assert classify("x.exe", 2, 1) == "usuario"


def test_analyzer_never_kills(fake_psutil):
    procs = [FakeProc(30, "game.exe", 100.0, 4000 * MB)]
    make_analyzer(fake_psutil, procs).snapshot()
    assert procs[0].killed is False
