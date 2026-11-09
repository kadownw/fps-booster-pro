# 08 — Modelo de seguridad y restauración

Estado actual (`go-rewrite`) marcado como ✅ implementado, ◐ parcial, ✗ falta.

| Requisito (Parte 16) | Estado | Detalle / propuesta |
|---|---|---|
| Modo diagnóstico de solo lectura | ◐ | Hay "modo simulación" (no escribe). Propuesta: modo **Diagnóstico** por defecto en la primera ejecución: detección, telemetría y benchmark sin ningún cambio. En `main`/`feat/benchmark` la simulación es el **modo por defecto** (regla del `CLAUDE.md` del colaborador); en `go-rewrite` el modo real es el defecto → **alinear (decisión D2)**. |
| Vista previa de cambios | ◐ | El diálogo muestra título y descripción. Falta mostrar **qué claves/servicios/tareas** se tocan y sus valores actuales → nuevos. |
| Consentimiento explícito | ✅ | Diálogo por ajuste y por perfil. Para perfiles: consentimiento **por ítem** con casillas, no global. |
| Permisos mínimos | ◐ | Hoy la app entera se reinicia como administrador. Propuesta: **UI sin elevación + helper elevado** con lista blanca de operaciones (registro en rutas permitidas, servicios permitidos, tareas permitidas) y protocolo de mensajes validado (ya propuesto en `PERFORMANCE_PLAN.md §3`). |
| Registro de acciones | ✅ | `changes.log` JSONL. Agregar: hash del estado anterior/posterior y versión de la app. |
| Copias de configuraciones originales | ✅ | `backup.json` (escritura atómica), por ajuste, incremental. Agregar: firma HMAC local para detectar manipulación. |
| Restauración por cambio | ✅ | Revertir ajuste. |
| Restauración por sesión | ✗ | Agrupar cambios en una "sesión" (p. ej., un experimento o perfil) y revertirlos juntos. |
| Detección de errores | ✅ | Verificación tras escribir; rollback del lote si falla. |
| Recuperación tras cierre inesperado | ◐ | El backup persiste y se puede revertir al reabrir. Falta: **diario de transacciones** (intención → aplicado → verificado) para detectar cambios a medio aplicar al arrancar y ofrecer completarlos o revertirlos; y cambios de **sesión** (prioridad de proceso) que deben revertirse si la app se cerró. |
| Validación de compatibilidad | ◐ | `MustExist`, `MinBuild/MaxBuild`, fabricante de GPU. Falta: filtro por KB (arquitectura CPU, X3D, portátil/batería). |
| Verificación de firmas de ejecutables | ✗ | Para PresentMon u otras herramientas externas: verificar **SHA-256 publicado + firma Authenticode del editor esperado** antes de ejecutar (como hace `feat/benchmark`). Verificar también la propia firma del exe distribuido (hoy **sin firmar**). |
| Control de dependencias | ◐ | `go.sum` y `package-lock.json` versionados. Agregar `govulncheck` y `npm audit` en CI. |
| Pruebas de regresión | ✅ | 27 pruebas Go, incluyendo round-trip real y pruebas admin opcionales. Agregar: invariantes de seguridad como en `docs/AUDIT.md` (p. ej., "ningún código desactiva Defender/VBS", "ningún perfil ejecuta listas fijas sin evaluación"). |
| Separación de componentes privilegiados | ✗ | Ver "Permisos mínimos". |
| Prevención de ejecución arbitraria de comandos | ✅ | `powercfg` con GUID validado; PowerShell solo con rutas de tareas validadas por regex; sin `cmd /c` con entrada del usuario. Agregar prueba que recorra el código buscando `exec.Command` fuera de los módulos permitidos (equivalente al invariante de `AUDIT.md`). |

## Reglas no negociables (consolidadas de `main`, `feat/benchmark` y este encargo)

1. No desactivar Defender, firewall, UAC, VBS/HVCI, Secure Boot, mitigaciones ni actualizaciones por defecto.
2. No cerrar procesos sin autorización explícita por proceso.
3. No tocar procesos ni archivos protegidos de juegos; no inyectar ni leer memoria; nada que interfiera con anticheats.
4. No aplicar overclock, undervolt ni cambios de BIOS.
5. No mostrar mejoras de FPS sin medición A/B.
6. Todo cambio persistente: backup + verificación + reversión; cambios de sesión: reversión automática.
7. No descargar ni ejecutar software externo sin autorización; si se autoriza, verificar hash y firma.
8. Drivers de kernel de terceros (p. ej., PawnIO para temperaturas de CPU): **opcional, desactivado por
   defecto**, con explicación del riesgo (antecedente: WinRing0, CVE-2020-14979, marcado por Defender).
9. Borrados irreversibles (limpieza) siempre con vista previa y confirmación; nunca cachés de shaders por defecto.

## Restauración en capas

1. **Por ajuste** (backup del valor original).
2. **Por sesión/experimento** (grupo de cambios).
3. **Revertir todo** (todos los backups).
4. **Punto de restauración de Windows** (opcional; Windows limita la frecuencia de creación).
5. **Guía manual** para cambios de BIOS (Nivel 4), que la app no puede revertir.
