import './style.css';
import {
  GetSystemInfo, GetLive, ListTweaks, ApplyTweak, RevertTweak, RevertAll,
  SetSimulation, GetSimulation, RelaunchAsAdmin, InitError,
} from '../wailsjs/go/main/App';

const NS = 'http://www.w3.org/2000/svg';

// Construye DOM sin innerHTML. Aplana arrays anidados e ignora null/false.
function h(tag, props = {}, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(props || {})) {
    if (v === false || v == null) continue;
    if (k === 'class') el.className = v;
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'style') el.style.cssText = v;
    else el[k] = v;
  }
  append(el, kids);
  return el;
}
function append(el, kids) {
  for (const kid of kids.flat(Infinity)) {
    if (kid === null || kid === undefined || kid === false) continue;
    el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  }
}
function svg(tag, attrs = {}, ...kids) {
  const el = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
  append(el, kids);
  return el;
}

// ---------- estado ----------
const S = { info: null, tweaks: [], cat: 'Todos', q: '', sim: false, busy: false, rows: new Map() };
const el = {};

// ---------- toasts y modal ----------
const toasts = h('div', { class: 'toasts' });
function toast(msg, kind = '') {
  const t = h('div', { class: 'toast ' + kind }, msg);
  toasts.append(t);
  setTimeout(() => { t.style.transition = 'opacity .3s'; t.style.opacity = '0'; setTimeout(() => t.remove(), 300); }, kind === 'err' ? 6500 : 2600);
}
function ask(title, body, ok = 'Aplicar') {
  return new Promise((resolve) => {
    const done = (v) => { scrim.remove(); document.removeEventListener('keydown', onKey); resolve(v); };
    const onKey = (e) => { if (e.key === 'Escape') done(false); };
    const scrim = h('div', { class: 'scrim', onclick: (e) => { if (e.target === scrim) done(false); } },
      h('div', { class: 'modal' },
        h('h3', {}, title), h('p', {}, body),
        h('p', {}, 'Se guarda el valor original y podés revertirlo cuando quieras.'),
        h('div', { class: 'actions' },
          h('button', { onclick: () => done(false) }, 'Cancelar'),
          h('button', { class: 'primary', onclick: () => done(true) }, ok))));
    document.addEventListener('keydown', onKey);
    document.body.append(scrim);
  });
}

// ---------- gauge ----------
function gauge(id, label) {
  const R = 30, C = 2 * Math.PI * R;
  const fg = svg('circle', { class: 'fg', cx: 37, cy: 37, r: R, fill: 'none', 'stroke-width': 7, 'stroke-dasharray': C, 'stroke-dashoffset': C });
  const s = svg('svg', { viewBox: '0 0 74 74' },
    svg('defs', {}, svg('linearGradient', { id: 'g', x1: 0, y1: 0, x2: 1, y2: 1 },
      svg('stop', { offset: '0', 'stop-color': '#6c8cff' }), svg('stop', { offset: '1', 'stop-color': '#9b6cff' }))),
    svg('circle', { class: 'bg', cx: 37, cy: 37, r: R, fill: 'none', 'stroke-width': 7 }), fg);
  const num = h('div', { class: 'num' }, '–');
  el[id] = { fg, num, C };
  return h('div', { class: 'glass gauge' }, s, h('div', {}, h('div', { class: 'k' }, label), num));
}
function setGauge(id, v) {
  const g = el[id]; if (!g) return;
  g.fg.setAttribute('stroke-dashoffset', g.C * (1 - Math.min(100, Math.max(0, v)) / 100));
  g.num.textContent = Math.round(v) + '%';
}

// ---------- filas de ajustes (se crean una sola vez y se actualizan) ----------
const RISK = { safe: 'Seguro', moderate: 'Moderado' };

function buildRow(t) {
  const input = h('input', {
    type: 'checkbox',
    onchange: async (e) => {
      const want = e.target.checked;
      e.target.checked = !want;            // el estado real lo decide el motor
      if (want && !(await ask(t.title, t.description))) return;
      run(want ? ApplyTweak : RevertTweak, t.id, want ? 'Aplicado' : 'Revertido');
    },
  });
  const tags = h('span', { class: 'tags', style: 'display:contents' });
  const row = h('div', { class: 'tweak', style: `animation-delay:${Math.min(S.rows.size, 14) * 25}ms` },
    h('div', { class: 'info' }, h('div', { class: 't' }, t.title, tags), h('div', { class: 'd' }, t.description)),
    h('label', { class: 'switch' }, input, h('span')));
  const r = { row, input, tags, t };
  S.rows.set(t.id, r);
  return r;
}

function paintRow(r, t) {
  r.t = t;
  r.row.classList.toggle('on', t.active);
  r.row.classList.toggle('off', t.disabled);
  r.input.checked = t.active;
  r.input.disabled = S.busy || t.disabled || (t.blocked && !t.active);
  r.row.title = t.disabled ? 'No aplica a tu equipo' : (t.blocked ? 'Requiere administrador' : '');
  r.tags.replaceChildren(...[
    h('span', { class: 'tag ' + t.risk }, RISK[t.risk] || t.risk),
    t.admin && h('span', { class: 'tag adm' }, 'Admin'),
    t.reboot && h('span', { class: 'tag' }, 'Reinicio'),
  ].filter(Boolean));
}

function visible(t) {
  const q = S.q.trim().toLowerCase();
  return (S.cat === 'Todos' || t.category === S.cat) &&
    (!q || (t.title + ' ' + t.description + ' ' + t.category).toLowerCase().includes(q));
}

function renderList() {
  const frag = document.createDocumentFragment();
  const cats = [...new Set(S.tweaks.filter(visible).map((t) => t.category))];
  for (const c of cats) {
    frag.append(h('div', { class: 'section' }, h('h2', {}, c)));
    for (const t of S.tweaks.filter((x) => x.category === c && visible(x))) frag.append(S.rows.get(t.id).row);
  }
  if (!cats.length) frag.append(h('p', { class: 'k', style: 'padding:30px 0;text-align:center' }, 'Nada coincide con tu búsqueda.'));
  el.list.replaceChildren(frag);
}

function renderNav() {
  const cats = ['Todos', ...new Set(S.tweaks.map((t) => t.category))];
  el.nav.replaceChildren(...cats.map((c) => {
    const n = c === 'Todos' ? S.tweaks.length : S.tweaks.filter((t) => t.category === c).length;
    return h('button', { class: 'nav' + (c === S.cat ? ' on' : ''), onclick: () => { S.cat = c; renderNav(); renderList(); el.main.scrollTo({ top: 0 }); } }, c, h('i', {}, n));
  }));
}

function paintAll() {
  for (const t of S.tweaks) { const r = S.rows.get(t.id); if (r) paintRow(r, t); }
  const anyUndo = S.tweaks.some((t) => t.canUndo);
  el.revertAll.disabled = S.busy || !anyUndo;
  el.simBox.disabled = S.busy;
  el.simBox.checked = S.sim;
  el.simBanner.hidden = !S.sim;
  el.adminBanner.hidden = !(S.info && !S.info.admin && S.tweaks.some((t) => t.blocked));
  el.busy.hidden = !S.busy;
}

async function refresh() {
  const list = (await ListTweaks()) || [];
  const newIds = list.map((t) => t.id).join();
  const needRebuild = newIds !== S.tweaks.map((t) => t.id).join();
  S.tweaks = list;
  if (needRebuild) { S.rows.clear(); for (const t of list) buildRow(t); renderNav(); renderList(); }
  paintAll();
}

async function run(fn, id, okMsg) {
  if (S.busy) return;
  S.busy = true; paintAll();
  try {
    const err = id === undefined ? await fn() : await fn(id);
    if (err) toast(err, 'err'); else toast(okMsg, 'ok');
  } catch (e) {
    toast(String(e), 'err');
  } finally {
    S.busy = false;
    await refresh();
  }
}

// ---------- layout ----------
function build() {
  el.nav = h('nav');
  el.simBox = h('input', { type: 'checkbox', onchange: async (e) => { await SetSimulation(e.target.checked); S.sim = e.target.checked; await refresh(); } });
  el.revertAll = h('button', { class: 'danger', onclick: async () => {
    if (await ask('Revertir todo', 'Todos los ajustes aplicados vuelven a sus valores originales.', 'Revertir')) run(RevertAll, undefined, 'Todo revertido');
  } }, 'Revertir todo');
  el.busy = h('div', { class: 'busy-bar', hidden: true });
  el.simBanner = h('div', { class: 'banner info', hidden: true }, 'Modo simulación: no se modifica nada en tu Windows.');
  el.adminBanner = h('div', { class: 'banner', hidden: true }, 'Algunos ajustes necesitan permisos de administrador.',
    h('button', { onclick: async () => { const e = await RelaunchAsAdmin(); if (e) toast(e, 'err'); } }, 'Reiniciar como administrador'));
  el.list = h('div');
  el.search = h('input', { class: 'search', placeholder: 'Buscar ajustes…', oninput: (e) => { S.q = e.target.value; renderList(); } });

  const i = S.info;
  const spec = h('div', { class: 'glass spec spec-card' },
    ...[['CPU', `${i.cpu || 'Desconocido'} · ${i.cores} hilos`], ['GPU', (i.gpus || []).join(' / ') || 'No detectada'],
        ['RAM', `${i.ramGB.toFixed(1)} GB`], ['Windows', i.windows]]
      .map(([k, v]) => h('div', {}, h('span', { class: 'k' }, k), h('b', {}, v))));

  el.main = h('main', {},
    h('div', { class: 'top' },
      h('div', {}, h('h1', {}, 'Optimización'), h('p', {}, 'Ajustes reversibles de Windows. No cierra procesos ni promete FPS.')),
      el.search),
    el.simBanner, el.adminBanner,
    h('div', { class: 'stats' }, gauge('cpu', 'CPU'), gauge('ram', 'RAM'), spec),
    el.list);

  document.getElementById('app').replaceChildren(
    h('aside', {},
      h('div', { class: 'brand' }, h('div', { class: 'logo' }, 'FPS'), h('div', {}, h('b', {}, 'FPS Booster Pro'), h('small', {}, 'v2 · Go'))),
      el.nav,
      h('div', { class: 'side-foot' },
        h('label', { class: 'sim' }, 'Modo simulación', h('span', { class: 'switch' }, el.simBox, h('span'))),
        el.revertAll)),
    el.main);
  document.body.append(el.busy, toasts);
}

async function tick() {
  try { const l = await GetLive(); setGauge('cpu', l.cpu); setGauge('ram', l.ram); } catch { /* cerrando */ }
}

(async () => {
  const [initErr, info, sim] = await Promise.all([InitError(), GetSystemInfo(), GetSimulation()]);
  S.info = info; S.sim = sim;
  if (initErr) {
    document.getElementById('app').replaceChildren(h('div', { class: 'banner err', style: 'margin:30px;grid-column:1/-1' }, 'No se pudo iniciar de forma segura: ' + initErr));
    return;
  }
  build();
  await refresh();
  tick(); setInterval(tick, 1500);
})();
