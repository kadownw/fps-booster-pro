import { h, svg, colorAt } from './lib.js';

// Medidor de marcas radiales: un arco abierto abajo con N marcas que se encienden según el valor.
// El color depende del estado (cool/warm/hot) y cambia suavemente.
const START = 150;   // grados (0° = derecha, sentido horario): abajo a la izquierda
const SWEEP = 240;

export function tickGauge({ ticks = 44, size = 170 } = {}) {
  const cx = 100, cy = 100, r1 = 70, r2 = 88;
  const lines = [];
  for (let i = 0; i < ticks; i++) {
    const a = ((START + (SWEEP * i) / (ticks - 1)) * Math.PI) / 180;
    const major = i % 5 === 0;
    const rin = major ? r1 - 4 : r1;
    lines.push(svg('line', {
      x1: cx + rin * Math.cos(a), y1: cy + rin * Math.sin(a), x2: cx + r2 * Math.cos(a), y2: cy + r2 * Math.sin(a),
      class: 'tick', 'stroke-width': major ? 3.2 : 2.4, 'stroke-linecap': 'round',
    }));
  }
  const value = h('div', { class: 'g-value' });
  const unit = h('span', { class: 'g-unit' });
  const label = h('div', { class: 'g-label' });
  const root = h('div', { class: 'gauge2', style: `width:${size}px;height:${size}px` },
    svg('svg', { viewBox: '0 0 200 200' }, lines),
    h('div', { class: 'g-center' }, h('div', { class: 'g-row' }, value, unit), label));

  let lastLit = -1, lastState = '';
  return {
    el: root,
    // frac: 0..1 | null (sin datos). text/unitText/labelText: lo que se muestra en el centro.
    set({ frac, state = 'cool', text, unitText = '', labelText = '' }) {
      value.textContent = text;
      unit.textContent = unitText;
      label.textContent = labelText;
      const lit = frac == null ? 0 : Math.round(Math.min(1, Math.max(0, frac)) * ticks);
      if (lit === lastLit && state === lastState) return;
      lastLit = lit; lastState = state;
      lines.forEach((ln, i) => {
        if (i < lit) {
          ln.style.stroke = colorAt(state, i / (ticks - 1));
          ln.style.opacity = 1;
        } else {
          ln.style.stroke = '';
          ln.style.opacity = '';
        }
        ln.classList.toggle('lit', i < lit);
      });
    },
  };
}
