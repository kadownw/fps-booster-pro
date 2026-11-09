import { h, svg } from './lib.js';

// Gráfico de líneas en tiempo real (0-100 %), con cuadrícula, área suave y cursor con valores.
// series: [{ key, label, color }]; data: { [key]: number[] } (el último valor es el más reciente).
// El ancho se adapta al contenedor (el viewBox usa píxeles reales, así el texto no se deforma).
export function lineChart({ series, height = 230 }) {
  const H = height, padL = 34, padR = 10, padT = 10, padB = 18;
  const ih = H - padT - padB;
  let W = 600, iw = W - padL - padR;

  const gridG = svg('g');
  const defs = svg('defs', {}, series.map((s) => svg('linearGradient', { id: 'ar-' + s.key, x1: 0, y1: 0, x2: 0, y2: 1 },
    svg('stop', { offset: '0', 'stop-color': s.color, 'stop-opacity': 0.28 }), svg('stop', { offset: '1', 'stop-color': s.color, 'stop-opacity': 0 }))));
  const areas = {}, lines = {};
  for (const s of series) {
    areas[s.key] = svg('path', { fill: `url(#ar-${s.key})`, stroke: 'none' });
    lines[s.key] = svg('path', { fill: 'none', stroke: s.color, 'stroke-width': 2, 'stroke-linejoin': 'round', 'stroke-linecap': 'round' });
  }
  const cursor = svg('line', { y1: padT, y2: padT + ih, class: 'cursor', style: 'display:none' });
  const dots = Object.fromEntries(series.map((s) => [s.key, svg('circle', { r: 3.5, fill: s.color, style: 'display:none' })]));
  const root = svg('svg', { class: 'chart', style: `height:${H}px` },
    defs, gridG, series.map((s) => areas[s.key]), series.map((s) => lines[s.key]), cursor, Object.values(dots));

  const tip = h('div', { class: 'tip', style: 'display:none' });
  const wrap = h('div', { class: 'chart-wrap' }, root, tip);
  let current = {}, visible = null, n = 0;

  const path = (arr, close) => {
    if (!arr.length) return '';
    const step = n > 1 ? iw / (n - 1) : 0;
    const off = n - arr.length; // alinea a la derecha cuando faltan puntos
    let d = '';
    arr.forEach((v, i) => {
      const x = padL + (off + i) * step, y = padT + ih * (1 - Math.min(100, Math.max(0, v)) / 100);
      d += (i ? 'L' : 'M') + x.toFixed(1) + ' ' + y.toFixed(1);
    });
    if (close) {
      const last = padL + (off + arr.length - 1) * step, first = padL + off * step;
      d += `L${last.toFixed(1)} ${padT + ih}L${first.toFixed(1)} ${padT + ih}Z`;
    }
    return d;
  };

  const draw = () => {
    n = Math.max(0, ...series.map((s) => (current[s.key] || []).length));
    for (const s of series) {
      const arr = current[s.key] || [];
      const on = !visible || visible[s.key] !== false;
      lines[s.key].setAttribute('d', on ? path(arr, false) : '');
      areas[s.key].setAttribute('d', on ? path(arr, true) : '');
    }
  };

  const layout = () => {
    W = Math.max(280, Math.round(wrap.clientWidth || 600));
    iw = W - padL - padR;
    root.setAttribute('viewBox', `0 0 ${W} ${H}`);
    gridG.replaceChildren(...[0, 25, 50, 75, 100].flatMap((v) => {
      const y = padT + ih * (1 - v / 100);
      return [svg('line', { x1: padL, x2: W - padR, y1: y, y2: y, class: 'grid' }),
        svg('text', { x: padL - 6, y: y + 3.5, class: 'axis', 'text-anchor': 'end' }, `${v}`)];
    }));
    draw();
  };
  layout();
  new ResizeObserver(() => { if (Math.abs((wrap.clientWidth || 0) - W) > 2) layout(); }).observe(wrap);

  const hide = () => { cursor.style.display = 'none'; tip.style.display = 'none'; Object.values(dots).forEach((d) => (d.style.display = 'none')); };
  wrap.addEventListener('pointermove', (e) => {
    const rect = root.getBoundingClientRect();
    const px = ((e.clientX - rect.left) / rect.width) * W;
    if (n < 2 || px < padL || px > W - padR) return hide();
    const idx = Math.round(((px - padL) / iw) * (n - 1));
    const x = padL + (idx / (n - 1)) * iw;
    cursor.setAttribute('x1', x); cursor.setAttribute('x2', x); cursor.style.display = '';
    const rows = [];
    for (const s of series) {
      const arr = current[s.key] || [];
      const v = arr[idx - (n - arr.length)];
      if (v == null || (visible && visible[s.key] === false)) { dots[s.key].style.display = 'none'; continue; }
      dots[s.key].setAttribute('cx', x);
      dots[s.key].setAttribute('cy', padT + ih * (1 - v / 100));
      dots[s.key].style.display = '';
      rows.push(h('div', {}, h('i', { style: `background:${s.color}` }), `${s.label} `, h('b', {}, `${v.toFixed(0)}%`)));
    }
    tip.replaceChildren(...rows);
    tip.style.display = '';
    const left = (x / W) * rect.width;
    tip.style.left = Math.min(Math.max(left + 12, 4), rect.width - 130) + 'px';
  });
  wrap.addEventListener('pointerleave', hide);

  return {
    el: wrap,
    update(data, vis) { current = data; visible = vis; draw(); },
  };
}

// Barras comparativas (sesiones de juego): una barra por elemento con su valor encima.
export function barChart({ items, max, unit = '', color = '#2fe3a2' }) {
  const top = max || Math.max(1, ...items.map((i) => i.value));
  return h('div', { class: 'bars' }, items.map((it) => h('div', { class: 'bar-col', title: it.title || '' },
    h('div', { class: 'bar-val' }, `${Math.round(it.value)}${unit}`),
    h('div', { class: 'bar-track' }, h('div', { class: 'bar-fill', style: `height:${Math.max(3, (it.value / top) * 100)}%;background:linear-gradient(180deg,${it.color || color},${it.color || color}22)` })),
    h('div', { class: 'bar-lab' }, it.label))));
}
