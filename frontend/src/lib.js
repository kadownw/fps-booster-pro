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
  lock: 'M6 11h12v9H6zM8 11V8a4 4 0 0 1 8 0v3',
  wrench: 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94z',
  wifi: 'M5 12.55a11 11 0 0 1 14.08 0M1.42 9a16 16 0 0 1 21.16 0M8.53 16.11a6 6 0 0 1 6.95 0M12 20h.01',
  bluetooth: 'm7 7 10 10-5 5V2l5 5L7 17',
  store: 'M6 2 3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4zM3 6h18M16 10a4 4 0 0 1-8 0',
  moon: 'M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z',
  installer: 'M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16zM3.27 6.96 12 12.01l8.73-5.05M12 22.08V12',
  valorant: 'M3 5h4l5 8 5-8h4L12 20z',
  user: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
  alert: 'M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0zM12 9v4M12 17h.01',
  info: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20zM12 16v-4M12 8h.01',
  video: 'M23 7l-7 5 7 5zM1 5h15v14H1z',
  eyeOff: 'M9.9 4.2A10 10 0 0 1 12 4c6.4 0 10 8 10 8a17 17 0 0 1-2.2 3.2M6.6 6.6C3.6 8.4 2 12 2 12s3.6 8 10 8a9.7 9.7 0 0 0 5.4-1.6M14.1 14.1a3 3 0 0 1-4.2-4.2M2 2l20 20',
  cloudOff: 'M2 2l20 20M5.8 5.8A7 7 0 0 0 7 20h10a5 5 0 0 0 1.9-.4M22 15.5A5 5 0 0 0 17 10h-1.3A7 7 0 0 0 9.5 5.2',
  power: 'M12 2v10M18.4 6.6a9 9 0 1 1-12.8 0',
  search: 'M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16zM21 21l-4.35-4.35',
  refresh: 'M21 12a9 9 0 1 1-3-6.7L21 8M21 3v5h-5',
  bell: 'M6 8a6 6 0 1 1 12 0c0 7 3 9 3 9H3s3-2 3-9M10.3 21a1.94 1.94 0 0 0 3.4 0',
  mapPin: 'M12 21s-7-6.2-7-12a7 7 0 1 1 14 0c0 5.8-7 12-7 12zM12 11a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z',
  timer: 'M10 2h4M12 14l3-3M12 22a8 8 0 1 0 0-16 8 8 0 0 0 0 16z',
  battery: 'M2 7h16v10H2zM22 11v2M6 10v4M10 10v4',
  plug: 'M12 22v-5M9 8V2M15 8V2M6 8h12v4a6 6 0 0 1-12 0z',
  mic: 'M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3zM19 10v2a7 7 0 0 1-14 0v-2M12 19v3',
  sparkles: 'M12 3l1.9 5.6 5.6 1.9-5.6 1.9L12 18l-1.9-5.6-5.6-1.9 5.6-1.9zM19 3v4M21 5h-4',
  keyboard: 'M2 6h20v12H2zM6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10',
  volume: 'M11 5 6 9H2v6h4l5 4zM19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07',
  volumeX: 'M11 5 6 9H2v6h4l5 4zM22 9l-6 6M16 9l6 6',
  folder: 'M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z',
  download: 'M12 3v12M7 10l5 5 5-5M5 21h14',
  activity: 'M22 12h-4l-3 9L9 3l-3 9H2',
  layers: 'M12 2l10 5-10 5L2 7zM2 17l10 5 10-5M2 12l10 5 10-5',
  trend: 'M3 17l6-6 4 4 8-8M14 7h7v7',
  temp: 'M14 14.76V3.5a2.5 2.5 0 0 0-5 0v11.26a4.5 4.5 0 1 0 5 0z',
  close: 'M18 6 6 18M6 6l12 12',
  check: 'M20 6 9 17l-5-5',
};

// Icono propio de cada ajuste (si no tiene, se usa el de su categoría).
export const TWEAK_ICON = {
  game_mode: 'gamepad', game_dvr: 'video', fullscreen_opt: 'monitor', windowed_flip: 'layers', gamebar_controller_off: 'gamepad',
  mouse_raw_curve: 'mouse', raw_mouse_throttle: 'mouse', kbd_latency: 'keyboard', sticky_keys: 'keyboard', audio_ducking_off: 'volume',
  transparency: 'layers', menu_delay: 'timer', visual_fx: 'sparkles', explorer_startup: 'folder', explorer_prefs: 'folder',
  startup_sound_off: 'volumeX', sound_scheme_none: 'volumeX', bg_apps: 'activity', hags: 'gpu', gpu_msi_mode: 'cpu',
  mpo_off: 'layers', flip_immediate: 'bolt', net_throttle: 'globe', delivery_opt: 'download', nic_power_off: 'plug',
  webdav_limit: 'folder', games_priority: 'trend', mmcss_extra: 'activity', foreground_boost: 'trend', power_throttle: 'battery',
  south_power_plan: 'power', hibernate_off: 'moon', fast_startup: 'power', usb_recovery: 'plug', timer_resolution: 'timer',
  memory_compression_off: 'ram', svchost_split: 'cog', trim_on: 'disk', ntfs_lastaccess_off: 'disk', maintenance_off: 'wrench',
  updates_pause: 'refresh', exclude_wu_drivers: 'download', store_autoupdate_off: 'store', edge_background_off: 'globe',
  search_local: 'search', content_delivery: 'bell', ads: 'eyeOff', activity_history: 'activity', news_interests: 'bell',
  telemetry: 'cloudOff', error_reporting: 'info', copilot: 'sparkles', ai_recall: 'sparkles', location_off: 'mapPin',
  settings_ads: 'bell', find_my_device: 'mapPin', start_recommended: 'grid', meet_now: 'video', cortana: 'mic',
  voice_activation_off: 'mic', cloud_sync_off: 'cloudOff', experimentation_lock: 'lock', media_tracking_off: 'eyeOff',
  nudge_blocker: 'bell', proximity_deny: 'wifi', preinstalled_apps: 'store', prevent_device_encryption: 'lock',
  svc_xbox_manual: 'gamepad', svc_max_performance: 'cog', retrim_ssd: 'disk', flush_dns: 'globe', remove_bloat_apps: 'trash',
  remove_copilot_app: 'trash', remove_gamebar_app: 'trash', bitlocker_off: 'lock', nvidia_profile: 'gpu', nv_driver_prefs: 'gpu',
  nv_hdcp_off: 'gpu', amd_ulps: 'gpu', amd_software_prefs: 'gpu', amd_graphics_perf: 'gpu',
};

// Icono de cada categoría de ajustes.
export const CAT_ICON = {
  Todos: 'grid', Energía: 'bolt', Juegos: 'gamepad', Entrada: 'mouse', Visual: 'eye', Sistema: 'cpu',
  GPU: 'gpu', Red: 'globe', Privacidad: 'shield', Fixes: 'wrench', Servicios: 'cog', 'Acciones únicas': 'play',
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
