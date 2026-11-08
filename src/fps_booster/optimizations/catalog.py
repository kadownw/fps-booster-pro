"""Catálogo de optimizaciones de la fase 1: todas reversibles, sin admin, solo HKCU/powercfg."""

from __future__ import annotations

from fps_booster.optimizations.base import (
    Optimization,
    PowerPlanOptimization,
    RegistryOptimization,
    RegistryValue,
)

GAME_MODE = RegistryOptimization(
    id="game_mode",
    title="Modo Juego de Windows",
    description=(
        "Windows prioriza el juego en primer plano y evita instalar actualizaciones o mostrar "
        "notificaciones mientras jugás."
    ),
    values=(RegistryValue("HKCU", r"Software\Microsoft\GameBar", "AutoGameModeEnabled"),),
    states={"on": {"AutoGameModeEnabled": 1}, "off": {"AutoGameModeEnabled": 0}},
    labels={"on": "Activado", "off": "Desactivado"},
)

GAME_DVR = RegistryOptimization(
    id="game_dvr",
    title="Grabación en segundo plano (Game DVR)",
    description=(
        "Desactiva la captura continua de la Xbox Game Bar, que graba el juego en segundo plano y "
        "consume GPU y disco. Podrás seguir usando la Game Bar manualmente."
    ),
    values=(
        RegistryValue("HKCU", r"System\GameConfigStore", "GameDVR_Enabled"),
        RegistryValue("HKCU", r"Software\Microsoft\Windows\CurrentVersion\GameDVR", "AppCaptureEnabled"),
    ),
    states={
        "on": {"GameDVR_Enabled": 1, "AppCaptureEnabled": 1},
        "off": {"GameDVR_Enabled": 0, "AppCaptureEnabled": 0},
    },
    labels={"on": "Activada", "off": "Desactivada"},
)

TRANSPARENCY = RegistryOptimization(
    id="transparency",
    title="Efectos de transparencia",
    description=(
        "Desactiva la transparencia de la barra de tareas y menús. Libera algo de carga de GPU en "
        "equipos con gráficos integrados. Solo cambia la apariencia."
    ),
    values=(
        RegistryValue(
            "HKCU", r"Software\Microsoft\Windows\CurrentVersion\Themes\Personalize", "EnableTransparency"
        ),
    ),
    states={"on": {"EnableTransparency": 1}, "off": {"EnableTransparency": 0}},
    labels={"on": "Activados", "off": "Desactivados"},
)

POWER_PLAN = PowerPlanOptimization()


def default_catalog() -> dict[str, Optimization]:
    return {opt.id: opt for opt in (POWER_PLAN, GAME_MODE, GAME_DVR, TRANSPARENCY)}
