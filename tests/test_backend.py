import pytest

from fps_booster.system.backend import (
    POWER_SCHEMES,
    SimulatedSystemBackend,
    SystemReader,
    WindowsSystemBackend,
    WindowsSystemReader,
    parse_active_scheme,
    parse_powercfg_list,
)

POWERCFG_ES = """
Planes de energía existentes (* Activo)
-----------------------------------
GUID del plan de energía: 381b4222-f694-41f0-9685-ff5bb260df2e  (Equilibrado) *
GUID del plan de energía: 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c  (Alto rendimiento)
GUID del plan de energía: d607337b-9037-4c24-9839-6ba76cf2e343  (Ultimate Performance (ExitLag))
"""


def test_parse_powercfg_list_locale_independent():
    schemes = parse_powercfg_list(POWERCFG_ES)
    assert schemes[POWER_SCHEMES["balanced"]] == "Equilibrado"
    assert schemes[POWER_SCHEMES["high_performance"]] == "Alto rendimiento"
    assert schemes["d607337b-9037-4c24-9839-6ba76cf2e343"] == "Ultimate Performance (ExitLag)"


def test_parse_active_scheme():
    out = "Power Scheme GUID: 381B4222-F694-41F0-9685-FF5BB260DF2E  (Balanced)"
    assert parse_active_scheme(out) == POWER_SCHEMES["balanced"]
    assert parse_active_scheme("sin datos") is None


class SpyReader(SystemReader):
    """Reader que registra llamadas; no tiene escritura."""

    def __init__(self):
        self.calls = []

    def get_active_power_scheme(self):
        self.calls.append("active")
        return POWER_SCHEMES["balanced"]

    def list_power_schemes(self):
        return {POWER_SCHEMES["balanced"]: "Balanced", POWER_SCHEMES["high_performance"]: "HP"}

    def read_registry_value(self, hive, key, name):
        self.calls.append(("read", name))
        return 1


def test_simulated_writes_only_to_overlay():
    reader = SpyReader()
    sim = SimulatedSystemBackend(reader)
    assert sim.read_registry_value("HKCU", r"Software\X", "Val") == 1
    sim.write_registry_dword("HKCU", r"Software\X", "Val", 0)
    assert sim.read_registry_value("HKCU", r"software\x", "val") == 0  # sin distinguir mayúsculas
    sim.delete_registry_value("HKCU", r"Software\X", "Val")
    assert sim.read_registry_value("HKCU", r"Software\X", "Val") is None
    sim.set_active_power_scheme(POWER_SCHEMES["high_performance"])
    assert sim.get_active_power_scheme() == POWER_SCHEMES["high_performance"]
    assert len(sim.operations) == 3


def test_simulated_refuses_to_wrap_real_backend():
    with pytest.raises(TypeError):
        SimulatedSystemBackend(WindowsSystemBackend())


def test_simulated_rejects_hklm_and_unknown_scheme():
    sim = SimulatedSystemBackend()
    with pytest.raises(PermissionError):
        sim.write_registry_dword("HKLM", r"SOFTWARE\X", "v", 1)
    with pytest.raises(LookupError):
        sim.set_active_power_scheme("11111111-2222-3333-4444-555555555555")
    with pytest.raises(ValueError):
        sim.set_active_power_scheme("not-a-guid; rm -rf")


def test_reader_has_no_write_methods():
    for method in ("set_active_power_scheme", "write_registry_dword", "delete_registry_value"):
        assert not hasattr(WindowsSystemReader, method)


def test_real_backend_validates_guid_before_running(monkeypatch):
    calls = []
    backend = WindowsSystemBackend(runner=lambda *a, **k: calls.append(a))
    with pytest.raises(ValueError):
        backend.set_active_power_scheme("x & shutdown")
    assert calls == []
