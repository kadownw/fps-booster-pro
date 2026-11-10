// Sistema de movimiento de South Tweaks.
//
// Principios: cada movimiento comunica algo (llegada, jerarquía, estado o respuesta a una acción), dura poco y
// usa solo transform/opacity (GPU, sin relayout). Hay un único vocabulario de curvas y tiempos, definido acá y
// en las variables CSS --ease-* / --t-*, para que todo se sienta parte de la misma pieza. Con «reducir
// movimiento» activado en Windows, todo se vuelve instantáneo.
//
//   Curvas    out    cubic-bezier(.16,1,.3,1)    entradas y revelados: arranca rápido y se posa suave
//             inOut  cubic-bezier(.65,0,.35,1)   cambios de estado y salidas
//             spring cubic-bezier(.34,1.56,.64,1) micro-interacciones (interruptores, contadores): leve rebote
//   Tiempos   rápido 160 ms (hover, press) · base 280 ms (estados) · lento 560 ms (entradas, medidores)
//   Escalonado 45 ms entre hermanos, máximo 12 pasos (nunca más de ~0,6 s de espera total)

const media = window.matchMedia('(prefers-reduced-motion: reduce)');
export const reduced = () => media.matches;

export const EASE = {
  out: 'cubic-bezier(.16,1,.3,1)',
  inOut: 'cubic-bezier(.65,0,.35,1)',
  spring: 'cubic-bezier(.34,1.56,.64,1)',
};

// ---------- revelados (carga de página y scroll) ----------
let io = null;
export function initReveals(scrollRoot) {
  if (io) io.disconnect();
  io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      e.target.classList.add('in');
      io.unobserve(e.target);
    }
  }, { root: scrollRoot, rootMargin: '0px 0px -4% 0px', threshold: 0.06 });
}

// reveal marca nodos para que entren al aparecer en pantalla, escalonados según su orden.
export function reveal(nodes, { base = 0, step = 45, max = 12, kind = 'up' } = {}) {
  let i = 0;
  for (const n of nodes) {
    if (!(n instanceof Element) || n.classList.contains('in')) continue;
    n.setAttribute('data-reveal', kind);
    n.style.setProperty('--d', `${base + Math.min(i, max) * step}ms`);
    i++;
    if (reduced() || !io) n.classList.add('in');
    else io.observe(n);
  }
  // Salvaguarda: si el observador no llega a disparar (ventana minimizada o sin repintar), lo que ya está
  // dentro de la pantalla se muestra igual. Nunca puede quedar contenido invisible.
  setTimeout(() => {
    const vh = window.innerHeight;
    for (const n of nodes) {
      if (!(n instanceof Element) || n.classList.contains('in')) continue;
      const r = n.getBoundingClientRect();
      if (r.top < vh && r.bottom > 0) { n.classList.add('in'); if (io) io.unobserve(n); }
    }
  }, 700 + base + Math.min(i, max) * step);
}

// Revela una página recién montada: encabezado primero, después los bloques y, dentro de las grillas, cada
// tarjeta en orden. Las que están fuera de la pantalla esperan a que el scroll las traiga.
export function revealPage(page) {
  const blocks = [...page.children];
  reveal(blocks, { base: 60, step: 60, max: 6 });
  page.querySelectorAll('.hwgrid, .tiles, .tgrid, .fixgrid, .cover-grid, .detail-grid, .tlist').forEach((g) => {
    reveal([...g.children], { base: 140, step: 45 });
  });
}

// ---------- transiciones entre páginas ----------
// La nueva página se monta AL INSTANTE (sus datos pueden llegar enseguida y tienen que encontrarla en el
// documento) y entra con su coreografía; la anterior queda superpuesta y se desvanece rápido, sin competir.
export function swapPage(container, next, onMounted) {
  const olds = [...container.children];
  container.append(next);
  for (const old of olds) {
    if (reduced()) { old.remove(); continue; }
    old.classList.add('page-leave');
    old.setAttribute('aria-hidden', 'true');
    setTimeout(() => old.remove(), 160);
  }
  onMounted && onMounted(next);
}

// ---------- títulos: entrada palabra por palabra ----------
export function splitTitle(tag, text, cls = '') {
  const el = document.createElement(tag);
  el.className = 'split ' + cls;
  el.setAttribute('aria-label', text);
  text.split(' ').forEach((w, i) => {
    const outer = document.createElement('span');
    outer.className = 'w';
    outer.setAttribute('aria-hidden', 'true');
    outer.style.setProperty('--d', `${60 + i * 55}ms`);
    const inner = document.createElement('span');
    inner.textContent = w;
    outer.append(inner);
    el.append(outer, ' ');
  });
  return el;
}

// ---------- números que cuentan hasta su valor ----------
export function tweenNumber(el, to, { dur = 650, decimals = 0 } = {}) {
  const from = parseFloat(el.dataset.v ?? 'NaN');
  el.dataset.v = String(to);
  if (!Number.isFinite(to)) { el.textContent = '–'; return; }
  if (!Number.isFinite(from) || reduced() || Math.abs(to - from) < 0.5) { el.textContent = to.toFixed(decimals); return; }
  const t0 = performance.now();
  const step = (now) => {
    const k = Math.min(1, (now - t0) / dur);
    const e = 1 - Math.pow(1 - k, 4); // ease-out cuártico, igual sensación que --ease-out
    el.textContent = (from + (to - from) * e).toFixed(decimals);
    if (k < 1 && el.dataset.v === String(to)) requestAnimationFrame(step);
  };
  requestAnimationFrame(step);
}

// Pequeño rebote para confirmar un cambio (contador de la cola, etc.).
export function bump(el) {
  if (!el || reduced()) return;
  el.classList.remove('bump');
  void el.offsetWidth; // reinicia la animación
  el.classList.add('bump');
}

// ---------- cursor: foco de luz en tarjetas y fondo que acompaña ----------
// Una sola escucha para toda la app; el trabajo se agrupa en un cuadro de animación.
export function initPointerFX() {
  let raf = 0, lastX = 0, lastY = 0, target = null;
  const root = document.documentElement;
  let px = 0, py = 0, tx = 0, ty = 0;
  window.addEventListener('pointermove', (e) => {
    lastX = e.clientX; lastY = e.clientY;
    target = e.target instanceof Element ? e.target.closest('.spot') : null;
    tx = (e.clientX / window.innerWidth) * 2 - 1;
    ty = (e.clientY / window.innerHeight) * 2 - 1;
    if (!raf) raf = requestAnimationFrame(frame);
  }, { passive: true });
  function frame() {
    raf = 0;
    if (target) {
      const r = target.getBoundingClientRect();
      target.style.setProperty('--mx', `${lastX - r.left}px`);
      target.style.setProperty('--my', `${lastY - r.top}px`);
    }
    if (reduced()) return;
    // el fondo sigue al cursor con inercia (interpolación), nunca de golpe
    px += (tx - px) * 0.08; py += (ty - py) * 0.08;
    root.style.setProperty('--px', px.toFixed(3));
    root.style.setProperty('--py', py.toFixed(3));
    if (Math.abs(tx - px) > 0.002 || Math.abs(ty - py) > 0.002) raf = requestAnimationFrame(frame);
  }
}

// Parallax del fondo con el scroll del contenido (muy sutil: el fondo se mueve al 6 % de la velocidad).
export function initScrollParallax(scroller) {
  let raf = 0;
  scroller.addEventListener('scroll', () => {
    if (raf || reduced()) return;
    raf = requestAnimationFrame(() => {
      raf = 0;
      document.documentElement.style.setProperty('--sy', String(scroller.scrollTop));
      scroller.classList.toggle('scrolled', scroller.scrollTop > 8);
    });
  }, { passive: true });
}

// ---------- tooltips explicativos ----------
// Cualquier elemento con data-tip (y opcional data-tip-title) muestra un globo al pasar el mouse o al recibir
// foco con el teclado. El texto se inserta como texto (nunca HTML).
export function initTooltips() {
  const tip = document.createElement('div');
  tip.className = 'tooltip';
  tip.setAttribute('role', 'tooltip');
  tip.id = 'st-tooltip';
  const title = document.createElement('b');
  const body = document.createElement('p');
  tip.append(title, body);
  document.body.append(tip);
  let cur = null, hideT = 0;
  const show = (el) => {
    clearTimeout(hideT);
    cur = el;
    title.textContent = el.getAttribute('data-tip-title') || '';
    title.hidden = !title.textContent;
    body.textContent = el.getAttribute('data-tip') || '';
    tip.dataset.tone = el.getAttribute('data-tip-tone') || '';
    tip.classList.add('on');
    el.setAttribute('aria-describedby', tip.id);
    const r = el.getBoundingClientRect();
    const tw = tip.offsetWidth, th = tip.offsetHeight;
    let x = r.left + r.width / 2 - tw / 2;
    x = Math.max(10, Math.min(x, window.innerWidth - tw - 10));
    let y = r.top - th - 10, below = false;
    if (y < 10) { y = r.bottom + 10; below = true; }
    tip.style.transform = `translate3d(${Math.round(x)}px, ${Math.round(y)}px, 0)`;
    tip.classList.toggle('below', below);
  };
  const hide = () => {
    if (cur) cur.removeAttribute('aria-describedby');
    cur = null;
    hideT = setTimeout(() => tip.classList.remove('on'), 60);
  };
  document.addEventListener('pointerover', (e) => {
    const el = e.target instanceof Element ? e.target.closest('[data-tip]') : null;
    if (el && el !== cur) show(el);
    else if (!el && cur) hide();
  });
  document.addEventListener('focusin', (e) => {
    const el = e.target instanceof Element ? e.target.closest('[data-tip]') : null;
    if (el) show(el);
  });
  document.addEventListener('focusout', hide);
  document.addEventListener('scroll', hide, true);
}
