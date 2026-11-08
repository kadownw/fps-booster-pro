"""Recomendaciones según el hardware. Nunca se promete una ganancia de FPS sin medir."""

from __future__ import annotations

from collections.abc import Iterable
from dataclasses import dataclass

from fps_booster.core.models import HardwareInfo, ProcessInfo, ResourceSample

NOT_MEASURED = "Impacto no medido: compará los FPS antes y después en la misma escena del juego."
GB = 1024**3


@dataclass(frozen=True)
class Recommendation:
    id: str
    title: str
    detail: str
    category: str  # energia | windows | hardware | procesos | juego
    priority: int  # 1 alta · 2 media · 3 baja
    optimization_id: str | None = None
    target: str | None = None
    impact_note: str = NOT_MEASURED

    @property
    def automatic(self) -> bool:
        return self.optimization_id is not None


def hardware_score(hw: HardwareInfo) -> int:
    score = 0
    threads = hw.cpu.logical_cores or 0
    score += 2 if threads >= 12 else 1 if threads >= 6 else 0
    ram_gb = hw.memory.total_bytes / GB
    score += 2 if ram_gb >= 15 else 1 if ram_gb >= 7.5 else 0  # 16 GB reporta ~15.9
    gpu = hw.primary_gpu()
    if gpu and not gpu.integrated:
        vram_gb = (gpu.vram_bytes or 0) / GB
        score += 2 if vram_gb >= 7.5 else 1 if vram_gb >= 3.5 else 0
    return score


def classify_tier(hw: HardwareInfo) -> str:
    score = hardware_score(hw)
    gpu = hw.primary_gpu()
    integrated = gpu is None or gpu.integrated
    if score >= 5 and not integrated:
        return "alta"
    if score >= 3:
        return "media"
    return "baja"


def optional_preselection(hw: HardwareInfo | None) -> set[str]:
    """Ajustes opcionales que conviene premarcar según el hardware."""
    if hw is None:
        return set()
    gpu = hw.primary_gpu()
    if classify_tier(hw) == "baja" or gpu is None or gpu.integrated:
        return {"transparency"}
    return set()


def recommend(
    hw: HardwareInfo,
    sample: ResourceSample | None = None,
    heavy_processes: Iterable[ProcessInfo] = (),
) -> list[Recommendation]:
    tier = classify_tier(hw)
    gpu = hw.primary_gpu()
    ram_gb = hw.memory.total_bytes / GB
    recs: list[Recommendation] = []

    plan_detail = "Evita que la CPU reduzca su frecuencia en medio de la partida."
    if hw.is_laptop:
        plan_detail += " En portátil, jugá enchufado: con batería aumenta el consumo y la temperatura."
    recs.append(Recommendation("power_plan", "Usar plan de energía de alto rendimiento", plan_detail,
                               "energia", 1, "power_plan", "high_performance"))
    recs.append(Recommendation("game_mode", "Activar el Modo Juego",
                               "Reduce interrupciones de Windows Update y notificaciones durante el juego.",
                               "windows", 2, "game_mode", "on"))
    recs.append(Recommendation(
        "game_dvr", "Desactivar la grabación en segundo plano",
        "La captura continua usa GPU y disco." + (" Muy relevante en tu equipo." if tier != "alta" else ""),
        "windows", 1 if tier != "alta" else 2, "game_dvr", "off"))

    if tier == "baja" or gpu is None or gpu.integrated:
        recs.append(Recommendation("transparency", "Desactivar efectos de transparencia",
                                   "Con gráficos integrados, cada recurso de GPU cuenta.",
                                   "windows", 2, "transparency", "off"))

    if gpu is not None and gpu.integrated:
        recs.append(Recommendation(
            "igpu_settings", "Gráficos integrados: ajustá la calidad en el juego",
            "Bajá resolución o usá escalado (FSR/XeSS) y sombras en bajo. Si tenés dos módulos de RAM "
            "iguales (dual channel), la GPU integrada rinde notablemente mejor.", "juego", 1))
    if hw.is_laptop and any(not g.integrated for g in hw.gpus) and any(g.integrated for g in hw.gpus):
        recs.append(Recommendation(
            "hybrid_gpu", "Asegurate de que el juego use la GPU dedicada",
            "Configuración > Sistema > Pantalla > Gráficos → elegí 'Alto rendimiento' para el juego.",
            "hardware", 1))

    if ram_gb < 7.5:
        recs.append(Recommendation(
            "low_ram", f"RAM limitada ({ram_gb:.0f} GB)",
            "Cerrá navegadores y apps de chat antes de jugar. Considerá ampliar a 16 GB.", "hardware", 1))
    elif ram_gb < 15:
        recs.append(Recommendation(
            "mid_ram", f"RAM justa para juegos actuales ({ram_gb:.0f} GB)",
            "Evitá tener el navegador abierto mientras jugás.", "hardware", 2))

    sys_disk = hw.system_disk()
    if sys_disk is not None and sys_disk.media_type == "HDD":
        recs.append(Recommendation(
            "hdd", "Disco del sistema mecánico (HDD)",
            "Instalar Windows y los juegos en un SSD reduce tiempos de carga y tirones.", "hardware", 1))

    vol = hw.system_volume()
    if vol is not None and vol.total_bytes and vol.free_bytes / vol.total_bytes < 0.15:
        recs.append(Recommendation(
            "low_disk", f"Poco espacio libre en {vol.mountpoint} ({vol.free_bytes / GB:.0f} GB)",
            "Liberá espacio con 'Liberador de espacio en disco' o Sensor de almacenamiento.",
            "hardware", 2))

    if sample is not None and sample.ram_percent >= 85:
        recs.append(Recommendation(
            "ram_pressure", f"Uso de RAM alto ahora ({sample.ram_percent:.0f}%)",
            "Revisá la pestaña Procesos y cerrá manualmente lo que no necesites.", "procesos", 1))

    heavy = list(heavy_processes)[:5]
    if heavy:
        names = ", ".join(sorted({p.name for p in heavy}))
        recs.append(Recommendation(
            "heavy_processes", "Procesos con consumo alto",
            f"{names}. FPS BOOSTER no los cierra: revisalos y cerralos vos si no los necesitás.",
            "procesos", 2))

    recs.append(Recommendation(
        "drivers", "Mantené actualizados los drivers de la GPU",
        "Descargalos solo desde el sitio oficial de NVIDIA, AMD o Intel.", "hardware", 3))
    if hw.windows.name == "Windows 10":
        recs.append(Recommendation(
            "win_update", "Windows 10 sin soporte",
            "Windows 10 dejó de recibir actualizaciones generales. Si tu equipo es compatible, "
            "Windows 11 incluye mejoras para juegos (DirectStorage, Auto HDR).", "windows", 3))

    recs.sort(key=lambda r: r.priority)
    return recs
