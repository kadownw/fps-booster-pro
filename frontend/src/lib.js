// Utilidades de interfaz: DOM sin innerHTML (los textos nunca se interpretan como HTML), iconos y formatos.
const NS = 'http://www.w3.org/2000/svg';

export function h(tag, props = {}, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(props || {})) {
    if (v === false || v == null) continue;
    if (k === 'class') el.className = v;
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'style') el.style.cssText = v;
    else if (k.startsWith('data-') || k.startsWith('aria-')) el.setAttribute(k, v);
    else el[k] = v;
  }
  append(el, kids);
  return el;
}

export function append(el, kids) {
  for (const kid of kids.flat(Infinity)) {
    if (kid === null || kid === undefined || kid === false) continue;
    el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  }
}

export function svg(tag, attrs = {}, ...kids) {
  const el = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
  append(el, kids);
  return el;
}

export const ICONS = {
  home: 'M3 11l9-8 9 8M5 10v10h5v-6h4v6h5V10',
  sliders: 'M4 6h10M18 6h2M4 12h4M12 12h8M4 18h12M20 18h0M14 4v4M8 10v4M16 16v4',
  trash: 'M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13M10 11v6M14 11v6',
  rocket: 'M5 15c-1 1-2 4-2 6 2 0 5-1 6-2M14 4c3-1 6-1 7-1 0 1 0 4-1 7l-7 7-6-6zM15 9h0',
  cpu: 'M9 3v2M15 3v2M9 19v2M15 19v2M3 9h2M3 15h2M19 9h2M19 15h2M7 7h10v10H7zM10 10h4v4h-4z',
  gpu: 'M3 7h18v10H3zM7 17v3M12 17v3M17 17v3M8.5 12a3 3 0 106 0 3 3 0 10-6 0',
  ram: 'M3 8h18v8H3zM7 8v8M11 8v8M15 8v8M19 8v8M5 16v2M9 16v2M13 16v2M17 16v2',
  disk: 'M4 6c0-1.7 3.6-3 8-3s8 1.3 8 3-3.6 3-8 3-8-1.3-8-3zM4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3',
  chart: 'M4 20V10M10 20V4M16 20v-8M22 20H2',
  bolt: 'M13 2L4 14h7l-1 8 9-12h-7z',
  heart: 'M12 21s-7-4.4-9.3-9A5.3 5.3 0 0112 6a5.3 5.3 0 019.3 6C19 16.6 12 21 12 21z',
  gamepad: 'M6 9h12a4 4 0 014 4v2a3 3 0 01-5.3 2L15 15H9l-1.7 2A3 3 0 012 15v-2a4 4 0 014-4zM8 11v3M6.5 12.5h3',
  monitor: 'M3 5h18v11H3zM8 20h8M12 16v4',
  chevron: 'M9 6l6 6-6 6',
  shield: 'M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z',
  restore: 'M3 12a9 9 0 109-9M3 4v5h5M12 8v4l3 2',
  mouse: 'M7 4h10a3 3 0 013 3v10a5 5 0 01-5 5H9a5 5 0 01-5-5V7a3 3 0 013-3zM12 4v6',
  eye: 'M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12zM12 15a3 3 0 100-6 3 3 0 000 6z',
  globe: 'M12 3a9 9 0 100 18 9 9 0 000-18zM3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18',
  cog: 'M12 15a3 3 0 100-6 3 3 0 000 6zM19.4 15a1.7 1.7 0 00.3 1.8l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.7 1.7 0 00-1.8-.3 1.7 1.7 0 00-1 1.5V21a2 2 0 11-4 0v-.1a1.7 1.7 0 00-1.1-1.5 1.7 1.7 0 00-1.8.3l-.1.1a2 2 0 11-2.8-2.8l.1-.1a1.7 1.7 0 00.3-1.8 1.7 1.7 0 00-1.5-1H3a2 2 0 110-4h.1a1.7 1.7 0 001.5-1.1 1.7 1.7 0 00-.3-1.8l-.1-.1a2 2 0 112.8-2.8l.1.1a1.7 1.7 0 001.8.3H9a1.7 1.7 0 001-1.5V3a2 2 0 114 0v.1a1.7 1.7 0 001 1.5 1.7 1.7 0 001.8-.3l.1-.1a2 2 0 112.8 2.8l-.1.1a1.7 1.7 0 00-.3 1.8V9a1.7 1.7 0 001.5 1H21a2 2 0 110 4h-.1a1.7 1.7 0 00-1.5 1z',
  play: 'M6 4l14 8-14 8z',
  grid: 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z',
  user: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
};

// Icono de cada categoría de ajustes.
export const CAT_ICON = {
  Todos: 'grid', Energía: 'bolt', Juegos: 'gamepad', Entrada: 'mouse', Visual: 'eye', Sistema: 'cpu',
  GPU: 'gpu', Red: 'globe', Privacidad: 'shield', Servicios: 'cog', 'Acciones únicas': 'play',
};

export const icon = (name, cls = '') => svg('svg', {
  viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 1.8,
  'stroke-linecap': 'round', 'stroke-linejoin': 'round', class: 'ico ' + cls,
}, svg('path', { d: ICONS[name] }));

export const fmtBytes = (n) => {
  if (n < 1024) return `${n} B`;
  const u = ['KB', 'MB', 'GB', 'TB']; let i = -1;
  do { n /= 1024; i++; } while (n >= 1024 && i < u.length - 1);
  return `${n.toFixed(n >= 100 ? 0 : 1)} ${u[i]}`;
};

export const fmtDuration = (sec) => {
  sec = Math.max(0, Math.round(sec));
  const hh = Math.floor(sec / 3600), mm = Math.floor((sec % 3600) / 60), ss = sec % 60;
  if (hh) return `${hh} h ${String(mm).padStart(2, '0')} min`;
  if (mm) return `${mm} min ${String(ss).padStart(2, '0')} s`;
  return `${ss} s`;
};

// ---------- color según el estado (verde -> amarillo -> rojo) ----------
export const PALETTE = {
  cool: ['#22c7d6', '#2fe3a2'],
  warm: ['#2fe3a2', '#f6c945'],
  hot: ['#f6c945', '#ff4d4d'],
};
export const GLOW = { cool: '#1fcf9a', warm: '#f6c945', hot: '#ff5a4d' };

const hex = (c) => [1, 3, 5].map((i) => parseInt(c.slice(i, i + 2), 16));
export function mix(a, b, t) {
  const A = hex(a), B = hex(b);
  return '#' + A.map((v, i) => Math.round(v + (B[i] - v) * t).toString(16).padStart(2, '0')).join('');
}
export const colorAt = (state, t) => mix(PALETTE[state][0], PALETTE[state][1], t);

// zone(valor, [aviso, crítico]) -> 'cool' | 'warm' | 'hot'
export const zone = (v, [warn, hot]) => (v >= hot ? 'hot' : v >= warn ? 'warm' : 'cool');
export const worst = (...zs) => (zs.includes('hot') ? 'hot' : zs.includes('warm') ? 'warm' : 'cool');
