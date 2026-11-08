"""Perfiles predefinidos. Un perfil es solo una lista de objetivos; no aplica nada por sí mismo."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class ProfileSetting:
    optimization_id: str
    target: str
    optional: bool = False  # opcional → desmarcado salvo que el hardware lo justifique
    reason: str = ""


@dataclass(frozen=True)
class Profile:
    id: str
    name: str
    tagline: str
    description: str
    settings: tuple[ProfileSetting, ...]


GAMING = Profile(
    id="gaming",
    name="Gaming",
    tagline="Máximo rendimiento",
    description=(
        "Plan de alto rendimiento, Modo Juego activado y grabación en segundo plano desactivada. "
        "Ideal enchufado a corriente."
    ),
    settings=(
        ProfileSetting("power_plan", "high_performance"),
        ProfileSetting("game_mode", "on"),
        ProfileSetting("game_dvr", "off"),
        ProfileSetting("transparency", "off", optional=True,
                       reason="Recomendado en equipos de gama baja o con gráficos integrados"),
    ),
)

BALANCED = Profile(
    id="balanced",
    name="Equilibrado",
    tagline="Uso diario",
    description="Plan equilibrado de Windows con el Modo Juego activo. Buen punto medio.",
    settings=(
        ProfileSetting("power_plan", "balanced"),
        ProfileSetting("game_mode", "on"),
    ),
)

SAVER = Profile(
    id="saver",
    name="Ahorro",
    tagline="Batería y temperatura",
    description="Plan economizador para alargar la batería y reducir temperatura y ruido.",
    settings=(
        ProfileSetting("power_plan", "power_saver"),
        ProfileSetting("transparency", "off", optional=True, reason="Ahorra un poco de energía"),
    ),
)

PROFILES: dict[str, Profile] = {p.id: p for p in (GAMING, BALANCED, SAVER)}
