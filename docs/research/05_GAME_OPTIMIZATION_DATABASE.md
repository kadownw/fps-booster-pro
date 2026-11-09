# 05 — Base de optimización por videojuego

Estado: **borrador de investigación.** Salvo lo marcado [SEC]/[OF], los datos por juego son **[NV]**
(conocimiento previo no verificado en esta sesión). Los juegos cambian de motor, API, anticheat y
opciones con actualizaciones: cada ficha debe verificarse contra el juego instalado y fecharse antes de
usarse para recomendar algo.

## 1. Reglas de seguridad por juego (obligatorias)

1. **Nunca** inyectar código, leer/escribir memoria del proceso, adjuntar depuradores, superponer
   overlays propios en juegos con anticheat, ni modificar binarios o archivos protegidos.
2. La medición se hace con **PresentMon (ETW)**, que no interactúa con el proceso del juego
   (`feat/benchmark/docs/BENCHMARK.md`).
3. Cambios de configuración: solo en archivos de configuración de usuario documentados, con el juego
   **cerrado**, backup completo del archivo y confirmación de que el juego arranca. En juegos donde la
   configuración se sincroniza con la cuenta (p. ej., Valorant), preferir guiar al usuario dentro del juego.
4. No forzar tecnologías por driver que el juego no integra (caso **Anti-Lag+**: baneos de VAC en CS2 y
   problemas en CoD y Apex [SEC]).
5. Opciones de lanzamiento de Steam/Epic: solo **sugerir**, no editar.

## 2. Esquema de ficha (`kb/games/<id>.json`)

| Campo | Descripción |
|---|---|
| `exe` / alias | Proceso a medir (p. ej., `VALORANT-Win64-Shipping.exe`, verificado en `feat/benchmark`) |
| `engine`, `apis` | Motor y APIs gráficas disponibles |
| `typical_limit` | Dónde suele estar el cuello (CPU/GPU) por segmento de hardware |
| `settings_cpu_heavy` / `settings_gpu_heavy` | Opciones que más cargan cada lado |
| `upscalers`, `framegen`, `latency_tech` | DLSS/FSR/XeSS, FG/MFG/AFMF, Reflex/Anti-Lag 2/XeLL |
| `vram_guidance`, `ram_guidance` | Umbrales orientativos |
| `anticheat` | Vanguard, VAC, EAC, BattlEye, Ricochet, etc. y restricciones conocidas |
| `config_files` | Rutas de usuario y si se sincronizan con la nube |
| `benchmark_method` | Escena reproducible (benchmark integrado, mapa de práctica) |
| `verified_on`, `game_version`, `sources` | Trazabilidad |

## 3. Fichas resumidas

| Juego | Motor / API | Límite típico | Opciones de mayor impacto | Upscaling / FG / latencia | Anticheat | Método de benchmark seguro |
|---|---|---|---|---|---|---|
| **Valorant** | Unreal Engine (migración a UE5 anunciada [NV]) / DX11 | CPU (FPS muy altos) | Multihilo, sombras, calidad de materiales; límite de FPS | Reflex | **Vanguard (kernel)** | Campo de práctica, recorrido fijo, 3×60 s (script en `feat/benchmark`) |
| **Counter-Strike 2** | Source 2 / DX11 (Vulkan opcional) | CPU | Sombras, MSAA, detalle de shaders; FSR para GPU débil | FSR 1; Reflex; **Anti-Lag 2 integrado** [SEC] | VAC | Mapa de benchmark de la comunidad o demo grabada; misma escena |
| **Fortnite** | Unreal Engine 5 / DX11, DX12, "Performance mode" | CPU en modo Performance; GPU con Lumen/Nanite | Modo de renderizado, Lumen, sombras, distancia de visión | TSR, DLSS, FSR; Reflex | EAC + BattlEye | Modo Creativo/replay con cámara fija |
| **Apex Legends** | Motor derivado de Source / DX11 (DX12 en pruebas [NV]) | CPU/GPU | Sombras, efectos volumétricos; límite de FPS | Reflex | EAC | Campo de tiro |
| **Call of Duty / Warzone** | IW Engine / DX12 | GPU (y CPU en mapas grandes) | Texturas (VRAM), sombras, oclusión | DLSS/FSR/XeSS, FG, Reflex | Ricochet (kernel) | Benchmark integrado (cuando exista) |
| **Overwatch 2** | Motor propio / DX11 | CPU/GPU | Calidad de efectos, sombras; límite de FPS | Reflex | Propio de Blizzard | Campo de práctica |
| **Rainbow Six Siege (X)** | AnvilNext / DX11, Vulkan | CPU | Sombras, reflejos, escala de render | Reflex [NV] | BattlEye | Benchmark integrado |
| **Rocket League** | Unreal Engine 3 / DX11 | CPU | Pocas; límite de FPS | — | EAC [NV] | Partida libre |
| **GTA V (Enhanced/Legacy)** | RAGE / DX11, DX12 (Enhanced) | CPU (población, distancia) | Población, distancia, hierba | DLSS/FSR (Enhanced) [NV] | BattlEye (online) | Benchmark integrado |
| **Red Dead Redemption 2** | RAGE / Vulkan, DX12 | GPU | Calidad de agua, volumétricos, MSAA | DLSS/FSR | — (Rockstar online) | Benchmark integrado |
| **Cyberpunk 2077** | REDengine 4 / DX12 | GPU (RT) y CPU (multitudes) | RT/Path tracing, densidad de multitud | DLSS (MFG), FSR, XeSS, Reflex | — | Benchmark integrado |
| **Elden Ring** | Motor propio / DX12 | Variable; **tope de 60 FPS** del juego | RT, sombras | — | EAC | Zona fija; recordar el tope (detección de "tope de FPS" ya existe en `feat/benchmark`) |
| **Minecraft Java** | Java + OpenGL | CPU (y RAM asignada a la JVM) | Distancia de renderizado, simulación | — (mods de terceros) | — (servidores) | Mundo de prueba con semilla fija |
| **Minecraft Bedrock** | RenderDragon / DX12 | GPU con RTX, CPU sin RT | Distancia, RT | DLSS (RTX) | — | Mundo fijo |
| **Hogwarts Legacy** | Unreal Engine 4 / DX12 | VRAM y CPU (tirones de streaming/shaders) | Texturas, RT | DLSS/FG, FSR, XeSS | Denuvo (DRM) | Recorrido fijo |
| **The Last of Us Part I** | Motor de Naughty Dog / DX12 | VRAM, CPU (compilación de shaders al iniciar) | Texturas | DLSS/FSR | — | Escena fija tras completar compilación |
| **God of War (2018)** | Motor propio / DX11 | GPU | Sombras, reflejos | DLSS/FSR | — | Escena fija |
| **Black Myth: Wukong** | Unreal Engine 5 / DX12 | GPU | Lumen, RT | DLSS/FG, FSR, XeSS | — | Herramienta de benchmark oficial publicada por el estudio [NV] |

## 4. Principios de recomendación por juego

- **Juegos competitivos (CPU-bound):** priorizar 1 % low y latencia: Reflex/Anti-Lag 2, límite de FPS
  estable, bajar opciones que cargan CPU. **No** usar generación de cuadros (agrega latencia y no sube los
  FPS del motor).
- **Juegos cinematográficos (GPU-bound):** upscaling en Calidad/Equilibrado; generación de cuadros si los
  FPS base son suficientes (orientativo ≥ 50–60 [NV]) y con Reflex; vigilar VRAM.
- **Tirones por compilación de shaders:** dejar que el juego precompile, **no borrar** la caché de shaders,
  evitar medir la primera corrida tras actualizar driver o juego.
- **Topes del juego:** detectar FPS planos en un valor típico (60/144/240) y advertir que la corrida mide
  estabilidad, no el máximo (ya implementado en `feat/benchmark`).

## 5. Próximos pasos de investigación por juego

1. Verificar cada ficha en el juego instalado (versión, API, opciones y rutas de configuración reales).
2. Consultar la documentación de soporte de cada estudio sobre archivos de configuración y anticheat.
3. Empezar por **Valorant, CS2 y Fortnite** (alineado con `PERFORMANCE_PLAN.md`), con baseline real en la
   PC de desarrollo.
