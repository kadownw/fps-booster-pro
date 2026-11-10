# Sistema visual y de movimiento

La interfaz se inspira en paneles de monitoreo modernos (fondo azul noche, superficies de vidrio sutiles,
medidores de anillo finos) y en apps de optimización con menú lateral y tarjetas por sección. Todo vive en
`frontend/src/style.css` (tokens y componentes) y `frontend/src/motion.js` (comportamiento del movimiento).

## Tokens

| Token | Valor | Uso |
|---|---|---|
| Fondo | `#060912` → `#0a1020` | base con dos luces suaves y una cuadrícula que se desvanece |
| Superficie | `rgba(148,170,255,.035–.095)` | tarjetas, botones, chips |
| Acento | `#38bdf8 → #6366f1` | activo, progreso, medidores normales |
| Estados | ok `#34d399` · atención `#fbbf24` · crítico `#fb7185` | salud, medidores, avisos |
| Fabricantes | NVIDIA `#76b900` · AMD `#ff3d47` · Intel `#3b9dff` | insignia, icono y línea superior de la tarjeta |
| Radios | 9 / 12 / 16 / 22 px | controles / filas / tarjetas / ventanas |

Cada categoría de ajustes tiene su color de icono (Energía ámbar, Juegos violeta, Entrada cian, Visual rosa,
Sistema azul, GPU verde, Red turquesa, Privacidad índigo, Servicios gris, Acciones únicas naranja) y cada ajuste
tiene su propio icono (`TWEAK_ICON` en `lib.js`).

## Movimiento

**Curvas**: `out cubic-bezier(.16,1,.3,1)` para entradas; `inOut cubic-bezier(.65,0,.35,1)` para estados y
salidas; `spring cubic-bezier(.34,1.56,.64,1)` solo en micro-interacciones.
**Tiempos**: 160 ms (hover, press) · 280 ms (estados) · 560 ms (entradas y medidores).
**Escalonado**: 45 ms entre hermanos, máximo 12 pasos.

| Momento | Qué pasa |
|---|---|
| Carga de la app | el menú lateral entra desde la izquierda; el título de la página sube palabra por palabra; los bloques aparecen escalonados |
| Scroll | cada tarjeta aparece (sube 16 px y se vuelve opaca) cuando entra en pantalla, una sola vez |
| Parallax | las luces del fondo se mueven al 6–10 % de la velocidad del scroll |
| Cursor | las luces del fondo siguen al cursor con inercia; las tarjetas muestran un foco de luz donde está el mouse |
| Cambio de página | la página nueva entra con su coreografía y la anterior se desvanece encima en 150 ms; el indicador del menú se desliza al botón nuevo |
| Números | los medidores de anillo y el puntaje de salud cuentan hasta su valor con la misma curva del arco |
| Imágenes | las portadas aparecen con fundido y escala 1,06 → 1; al pasar el mouse hacen zoom suave y muestran los FPS |
| Micro-interacciones | interruptores con rebote leve, botones que se hunden al presionar, contador de la cola que rebota al cambiar, barra de progreso con brillo |
| Avisos | el triángulo de peligro muestra al pasar el mouse (o con el teclado) qué desactiva el ajuste y qué se pierde |

Solo se animan `transform` y `opacity`. Con «Reducir movimiento» de Windows activado todo es instantáneo, y
ningún contenido puede quedar invisible: si el observador de scroll no dispara, lo visible se muestra igual.
