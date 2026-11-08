import sys

import pytest

from fps_booster.hardware.detector import (
    HardwareDetector,
    build_windows_info,
    is_integrated_gpu,
    parse_disks,
    parse_gpus,
    vendor_from_name,
)

GB = 1024**3


@pytest.mark.parametrize("name,vendor,integrated", [
    ("NVIDIA GeForce RTX 3060", "nvidia", False),
    ("AMD Radeon RX 9060 XT", "amd", False),
    ("AMD Radeon(TM) Graphics", "amd", True),
    ("AMD Radeon 780M", "amd", True),
    ("Intel(R) UHD Graphics 620", "intel", True),
    ("Intel(R) Arc(TM) Graphics", "intel", True),
    ("Intel(R) Arc(TM) A770 Graphics", "intel", False),
    ("Microsoft Basic Display Adapter", "microsoft", True),
])
def test_gpu_vendor_and_integration(name, vendor, integrated):
    assert vendor_from_name(name) == vendor
    assert is_integrated_gpu(name, vendor) is integrated


def test_parse_gpus_prefers_registry_vram_over_wmi_cap():
    raw = [{"Name": "NVIDIA GeForce RTX 4070", "AdapterRAM": 4293918720, "DriverVersion": "32.0"}]
    reg = [{"DriverDesc": "NVIDIA GeForce RTX 4070", "Memory": 12 * GB}]
    (gpu,) = parse_gpus(raw, reg)
    assert gpu.vram_bytes == 12 * GB
    assert gpu.integrated is False


def test_parse_gpus_handles_single_object_and_binary_memory():
    raw = {"Name": "AMD Radeon RX 6600", "AdapterRAM": None}
    reg = {"DriverDesc": "AMD Radeon RX 6600", "Memory": list((8 * GB).to_bytes(8, "little"))}
    (gpu,) = parse_gpus(raw, reg)
    assert gpu.vram_bytes == 8 * GB


def test_parse_disks_media_and_system_flag():
    raw = [
        {"FriendlyName": "Samsung 980", "MediaType": "Unspecified", "BusType": "NVMe", "Size": 10},
        {"FriendlyName": "WD Blue", "MediaType": "HDD", "BusType": "SATA", "Size": 20},
    ]
    disks = parse_disks(raw, "Samsung 980")
    assert disks[0].media_type == "SSD" and disks[0].is_system
    assert disks[1].media_type == "HDD" and not disks[1].is_system


@pytest.mark.parametrize("build,name", [
    (22631, "Windows 11"), (19045, "Windows 10"), (9600, "Windows (no compatible)"),
])
def test_windows_version_from_build(build, name):
    info = build_windows_info(str(build), "22H2", "Professional", 100)
    assert info.name == name
    assert info.build == build


def test_windows_fallback_platform_version():
    info = build_windows_info(None, None, None, None, fallback_version="10.0.26100")
    assert info.name == "Windows 11"


def test_detector_with_fakes(fake_psutil):
    def ps(_script):
        return {
            "gpus": [{"Name": "Intel(R) UHD Graphics", "AdapterRAM": 1 * GB}],
            "gpu_registry": [],
            "disks": [{"FriendlyName": "HDD1", "MediaType": "HDD", "Size": 1000}],
            "system_disk": "HDD1",
        }

    reg = {
        "ProcessorNameString": "Intel(R) Core(TM) i5-8250U  CPU @ 1.60GHz",
        "CurrentBuildNumber": "19045", "DisplayVersion": "22H2", "EditionID": "Core", "UBR": 4291,
    }
    det = HardwareDetector(ps_runner=ps, reg_reader=lambda h, k, n: reg.get(n), psutil_mod=fake_psutil())
    hw = det.detect()
    assert hw.cpu.name == "Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz"
    assert hw.cpu.logical_cores == 8
    assert hw.windows.name == "Windows 10" and hw.windows.full_build == "19045.4291"
    assert hw.gpus[0].integrated
    assert hw.system_disk().media_type == "HDD"
    assert len(hw.volumes) == 1  # se ignora la unidad de CD
    assert hw.is_laptop is False


def test_detector_survives_powershell_failure(fake_psutil):
    det = HardwareDetector(ps_runner=lambda s: None, reg_reader=lambda h, k, n: None,
                           psutil_mod=fake_psutil())
    hw = det.detect()
    assert hw.gpus == () and hw.disks == ()
    assert hw.primary_gpu() is None


@pytest.mark.windows
@pytest.mark.skipif(sys.platform != "win32", reason="requiere Windows")
def test_real_detection_is_read_only_and_works():
    hw = HardwareDetector().detect()
    assert hw.cpu.logical_cores and hw.memory.total_bytes > 0
    assert hw.windows.is_supported
