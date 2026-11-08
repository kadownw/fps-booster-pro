"""Identificación de procesos de alto consumo.

IMPORTANTE: este módulo solo LEE. No cierra, suspende ni cambia la prioridad de ningún proceso.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Any

import psutil
from psutil import AccessDenied, NoSuchProcess, ZombieProcess

from fps_booster.core.models import ProcessInfo

PROTECTED = frozenset(
    n.lower()
    for n in (
        "System", "System Idle Process", "Registry", "smss.exe", "csrss.exe", "wininit.exe",
        "winlogon.exe", "services.exe", "lsass.exe", "lsaiso.exe", "svchost.exe", "dwm.exe",
        "explorer.exe", "fontdrvhost.exe", "Memory Compression", "MsMpEng.exe",
        "NisSrv.exe", "SecurityHealthService.exe", "audiodg.exe", "sihost.exe", "ctfmon.exe",
        "spoolsv.exe", "WmiPrvSE.exe", "dllhost.exe", "conhost.exe", "taskhostw.exe",
        "RuntimeBroker.exe", "SearchIndexer.exe", "StartMenuExperienceHost.exe",
        "ShellExperienceHost.exe", "TextInputHost.exe", "SystemSettings.exe", "MsMpEngCP.exe",
    )
)

BACKGROUND_APPS = {
    "chrome.exe": "Navegador",
    "msedge.exe": "Navegador",
    "firefox.exe": "Navegador",
    "opera.exe": "Navegador",
    "brave.exe": "Navegador",
    "discord.exe": "Chat de voz",
    "teams.exe": "Reuniones",
    "ms-teams.exe": "Reuniones",
    "zoom.exe": "Reuniones",
    "slack.exe": "Mensajería",
    "whatsapp.exe": "Mensajería",
    "telegram.exe": "Mensajería",
    "spotify.exe": "Música",
    "onedrive.exe": "Sincronización en la nube",
    "dropbox.exe": "Sincronización en la nube",
    "googledrivefs.exe": "Sincronización en la nube",
    "steamwebhelper.exe": "Launcher de juegos",
    "epicgameslauncher.exe": "Launcher de juegos",
    "eadesktop.exe": "Launcher de juegos",
    "battle.net.exe": "Launcher de juegos",
    "upc.exe": "Launcher de juegos",
    "obs64.exe": "Grabación",
    "code.exe": "Editor",
    "adobe desktop service.exe": "Adobe",
    "creative cloud.exe": "Adobe",
}


@dataclass(frozen=True)
class Thresholds:
    cpu_percent: float = 15.0  # % del total de CPU
    memory_mb: float = 800.0


DEFAULT_THRESHOLDS = Thresholds()

def classify(name: str, pid: int, own_pid: int) -> str:
    if pid == own_pid:
        return "propio"
    lname = name.lower()
    if lname in PROTECTED or pid in (0, 4):
        return "protegido"
    if lname in BACKGROUND_APPS:
        return "segundo_plano"
    return "usuario"


class ProcessAnalyzer:
    def __init__(
        self,
        psutil_mod: Any = psutil,
        thresholds: Thresholds = DEFAULT_THRESHOLDS,
        own_pid: int | None = None,
    ):
        self._ps = psutil_mod
        self.thresholds = thresholds
        self._own_pid = os.getpid() if own_pid is None else own_pid
        self._cpu_count = max(1, int(self._ps.cpu_count() or 1))

    def snapshot(self) -> list[ProcessInfo]:
        result: list[ProcessInfo] = []
        for proc in self._ps.process_iter(["pid", "name", "memory_info"]):
            try:
                info = proc.info
                pid = int(info["pid"])
                if pid == 0:
                    continue
                name = info.get("name") or f"pid {pid}"
                mem = info.get("memory_info")
                rss = int(mem.rss) if mem is not None else 0
                # psutil cachea las instancias de process_iter: la 1.ª lectura de CPU da 0.
                cpu = float(proc.cpu_percent(interval=None)) / self._cpu_count
            except (NoSuchProcess, AccessDenied, ZombieProcess):
                continue
            result.append(self._build(pid, name, round(cpu, 1), rss))
        result.sort(key=lambda p: (p.cpu_percent, p.memory_bytes), reverse=True)
        return result

    def _build(self, pid: int, name: str, cpu: float, rss: int) -> ProcessInfo:
        reasons: list[str] = []
        if cpu >= self.thresholds.cpu_percent:
            reasons.append(f"CPU alta ({cpu:.0f}%)")
        if rss >= self.thresholds.memory_mb * 1024 * 1024:
            reasons.append(f"RAM alta ({rss / 1024 / 1024:.0f} MB)")
        category = classify(name, pid, self._own_pid)
        if category == "segundo_plano":
            reasons.append(BACKGROUND_APPS[name.lower()])
        return ProcessInfo(
            pid=pid,
            name=name,
            cpu_percent=cpu,
            memory_bytes=rss,
            category=category,
            heavy=any(r.startswith(("CPU", "RAM")) for r in reasons),
            reasons=tuple(reasons),
        )


def heavy_candidates(processes: list[ProcessInfo], limit: int = 10) -> list[ProcessInfo]:
    """Procesos pesados que el usuario podría revisar (excluye sistema y la propia app)."""
    return [p for p in processes if p.heavy and p.category in ("usuario", "segundo_plano")][:limit]
