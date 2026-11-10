import { h, svg } from './lib.js';
import { tweenNumber } from './motion.js';

// Medidor de anillo: arco de 280° abierto abajo, trazo fino con extremo redondeado y degradado según el
// estado (normal / atención / crítico). El arco y el número se animan juntos con la misma curva.
const ARC = 77.8;            // 280° de 360° expresado sobre pathLength = 100
let uid = 0;

const TONES = {
  cool: ['#38bdf8', '#6366f1'],
  warm: ['#fbbf24', '#f97316'],
  hot: ['#fb7185', '#e11d48'],
};

export function ringGauge({ size = 132 } = {}) {
  const id = 'rg' + uid++;
  const stop1 = svg('stop', { offset: '0', 'stop-color': TONES.cool[0] });
  const stop2 = svg('stop', { offset: '1', 'stop-color': TONES.cool[1] });
  const track = svg('circle', { cx: 60, cy: 60, r: 50, pathLength: 100, class: 'rg-track', 'stroke-dasharray': `${ARC} 100` });
  const bar = svg('circle', { cx: 60, cy: 60, r: 50, pathLength: 100, class: 'rg-bar', stroke: `url(#${id})`, 'stroke-dasharray': `0 100` });
  const value = h('span', { class: 'rg-val' }, '–');
  const unit = h('span', { class: 'rg-unit' });
  const label = h('div', { class: 'rg-label' });
  const root = h('div', { class: 'ring', style: `width:${size}px;height:${size}px` },
    svg('svg', { viewBox: '0 0 120 120' },
      svg('defs', {}, svg('linearGradient', { id, x1: 0, y1: 0, x2: 1, y2: 1 }, stop1, stop2)),
      svg('g', { transform: 'rotate(130 60 60)' }, track, bar)),
    h('div', { class: 'rg-center' }, h('div', { class: 'rg-row' }, value, unit), label));

  let lastState = '';
  return {
    el: root,
    // frac 0..1 | null (sin datos). num: número a mostrar; unitText / labelText: lo que lo acompaña.
    set({ frac, state = 'cool', num, unitText = '', labelText = '', decimals = 0 }) {
      unit.textContent = unitText;
      label.textContent = labelText;
      if (num == null || frac == null) {
        value.textContent = '–';
        bar.style.strokeDasharray = '0 100';
        root.classList.add('nodata');
        return;
      }
      root.classList.remove('nodata');
      tweenNumber(value, num, { decimals });
      const f = Math.min(1, Math.max(0, frac));
      bar.style.strokeDasharray = `${(f * ARC).toFixed(2)} 100`;
      if (state !== lastState) {
        lastState = state;
        stop1.setAttribute('stop-color', TONES[state][0]);
        stop2.setAttribute('stop-color', TONES[state][1]);
        root.dataset.state = state;
      }
    },
  };
}
