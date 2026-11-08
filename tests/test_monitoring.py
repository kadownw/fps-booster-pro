from fps_booster.core.models import GpuSample
from fps_booster.monitoring.gpu import (
    GpuMetricsProvider,
    NvidiaSmiProvider,
    aggregate_engine_utilization,
    parse_nvidia_smi,
)
from fps_booster.monitoring.sampler import ResourceSampler


def test_parse_nvidia_smi():
    s = parse_nvidia_smi("57, 2048, 8192, 63\n")
    assert s.utilization == 57 and s.temperature_c == 63
    assert s.memory_used_bytes == 2048 * 1024 * 1024
    assert parse_nvidia_smi("") is None


def test_parse_nvidia_smi_not_supported_fields():
    s = parse_nvidia_smi("[N/A], 100, 4096, [Not Supported]")
    assert s.utilization is None and s.temperature_c is None


def test_nvidia_provider_handles_failure():
    def boom(*a, **k):
        raise OSError("no exe")

    assert NvidiaSmiProvider("nvidia-smi", runner=boom).sample() is None


def test_aggregate_engine_like_task_manager():
    items = [
        ("pid_100_luid_0x0_0xABC_phys_0_eng_0_engtype_3D", 30.0),
        ("pid_200_luid_0x0_0xABC_phys_0_eng_0_engtype_3D", 25.0),
        ("pid_300_luid_0x0_0xABC_phys_0_eng_1_engtype_3D", 70.0),
        ("garbage", 99.0),
    ]
    assert aggregate_engine_utilization(items) == 70.0
    assert aggregate_engine_utilization([]) is None
    assert aggregate_engine_utilization([("pid_1_luid_0x0_0x1_phys_0_eng_0_engtype_3D", 250.0)]) == 100.0


class StaticGpu(GpuMetricsProvider):
    name = "fake"

    def sample(self):
        return GpuSample(42.0, 1, 2, None, "fake")


class BrokenGpu(GpuMetricsProvider):
    name = "broken"

    def sample(self):
        raise RuntimeError("driver crash")


def test_sampler_rates_and_gpu(fake_psutil):
    t = iter([0.0, 1.0, 2.0])
    sampler = ResourceSampler(psutil_mod=fake_psutil(), gpu_provider=StaticGpu(), clock=lambda: next(t))
    first = sampler.sample()
    second = sampler.sample()
    assert first.disk_read_bps == 0.0
    assert second.disk_read_bps == 10 * 1024 * 1024
    assert second.cpu_percent == 25.0 and len(second.per_cpu) == 8
    assert second.ram_used_bytes == 10 * 1024**3
    assert second.gpu.utilization == 42.0
    assert sampler.gpu_source == "fake"


def test_sampler_survives_gpu_errors(fake_psutil):
    sampler = ResourceSampler(psutil_mod=fake_psutil(), gpu_provider=BrokenGpu())
    assert sampler.sample().gpu is None
