from conftest import make_hw
from fps_booster.core.models import ProcessInfo
from fps_booster.recommendations.recommender import (
    NOT_MEASURED,
    classify_tier,
    optional_preselection,
    recommend,
)


def ids(recs):
    return {r.id for r in recs}


def test_tiers():
    assert classify_tier(make_hw()) == "alta"
    assert classify_tier(make_hw(threads=8, ram_gb=16, gpu_name="NVIDIA GTX 1650", vram_gb=4)) == "media"
    low = make_hw(threads=4, ram_gb=8, gpu_name="Intel(R) UHD Graphics", vram_gb=1, integrated=True)
    assert classify_tier(low) == "baja"
    # Una iGPU nunca es "alta", aunque CPU y RAM sean buenas.
    assert classify_tier(make_hw(gpu_name="AMD Radeon 780M", integrated=True)) == "media"


def test_low_end_recommendations():
    hw = make_hw(threads=4, ram_gb=4, gpu_name="Intel(R) UHD Graphics", vram_gb=1, integrated=True,
                 media="HDD", free_ratio=0.05, build=19045, laptop=True)
    recs = recommend(hw)
    assert {"transparency", "igpu_settings", "low_ram", "hdd", "low_disk", "win_update"} <= ids(recs)
    assert "enchufado" in next(r for r in recs if r.id == "power_plan").detail
    assert optional_preselection(hw) == {"transparency"}


def test_high_end_has_no_low_end_noise():
    recs = recommend(make_hw())
    assert not ({"transparency", "igpu_settings", "low_ram", "hdd", "low_disk"} & ids(recs))
    assert optional_preselection(make_hw()) == set()


def test_no_fps_promises():
    hw = make_hw(threads=4, ram_gb=4, gpu_name="Intel(R) UHD Graphics", integrated=True, media="HDD")
    for r in recommend(hw):
        assert r.impact_note == NOT_MEASURED
        text = f"{r.title} {r.detail}".lower()
        assert "fps" not in text or "medí" in text
        assert "%" not in r.detail or "uso" in r.title.lower()


def test_heavy_processes_are_suggested_not_closed():
    procs = [ProcessInfo(1, "chrome.exe", 30.0, 2 * 1024**3, "segundo_plano", True, ("CPU alta",))]
    rec = next(r for r in recommend(make_hw(), heavy_processes=procs) if r.id == "heavy_processes")
    assert "no los cierra" in rec.detail
    assert rec.optimization_id is None


def test_recommendations_sorted_by_priority():
    prios = [r.priority for r in recommend(make_hw(ram_gb=8))]
    assert prios == sorted(prios)
