import './style.css';
import {
  GetSystemInfo, GetLive, ListTweaks, ApplyTweak, RevertTweak, RevertAll,
  SetSimulation, GetSimulation, RelaunchAsAdmin, InitError,
  ListProfiles, ApplyProfile, CreateRestorePoint,
  ScanCleanup, RunCleanup, ListStartup, SetStartup,
} from '../wailsjs/go/main/App';

const NS = 'http://www.w3.org/2000/svg';

// ---------- helpers de DOM (sin innerHTML: los textos nunca se interpretan como HTML) ----------
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
const icon = (d) => svg('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 1.8, 'stroke-linecap': 'round', 'stroke-linejoin': 'round', class: 'ico' }, svg('path', { d }));
const ICONS = {
  home: 'M3 11l9-8 9 8M5 10v10h5v-6h4v6h5V10',
  sliders: 'M4 6h10M18 6h2M4 12h4M12 12h8M4 18h12M20 18h0M14 4v4M8 10v4M16 16v4',
  trash: 'M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13M10 11v6M14 11v6',
  rocket: 'M5 15c-1 1-2 4-2 6 2 0 5-1 6-2M14 4c3-1 6-1 7-1 0 1 0 4-1 7l-7 7-6-6zM15 9h0',
};
const fmtBytes = (n) => {
  if (n < 1024) return `${n} B`;
  const u = ['KB', 'MB', 'GB', 'TB']; let i = -1;
  do { n /= 1024; i++; } while (n >= 1024 && i < u.length - 1);
  return `${n.toFixed(n >= 100 ? 0 : 1)} ${u[i]}`;
};

// ---------- estado ----------
const S = {
  info: null, tweaks: [], profiles: [], cat: 'Todos', q: '', sim: false, busy: false, page: 'home',
  rows: new Map(), clean: null, startup: null, live: { cpu: 0, ram: 0 },
};
const el = {};

// ---------- toasts y modal ----------
const toasts = h('div', { class: 'toasts' });
function toast(msg, kind = '') {
  const t = h('div', { class: 'toast ' + kind }, msg);
  toasts.append(t);
  setTimeout(() => { t.style.transition = 'opacity .3s'; t.style.opacity = '0'; setTimeout(() => t.remove(), 300); }, kind === 'err' ? 6500 : 2600);
}
// ask(): confirma. `body` puede ser texto o nodos; `check` agrega una casilla opcional (devuelve {ok, checked}).
function dialog(title, body, { ok = 'Aceptar', cancel = 'Cancelar', check = null, danger = false } = {}) {
  return new Promise((resolve) => {
    let chk = null;
    const done = (v) => { scrim.remove(); document.removeEventListener('keydown', onKey); resolve({ ok: v, checked: chk ? chk.checked : false }); };
    const onKey = (e) => { if (e.key === 'Escape') done(false); };
    if (check) chk = h('input', { type: 'checkbox', checked: check.checked });
    const scrim = h('div', { class: 'scrim', onclick: (e) => { if (e.target === scrim) done(false); } },
      h('div', { class: 'modal' },
        h('h3', {}, title), h('div', { class: 'mbody' }, body),
        check && h('label', { class: 'chk' }, chk, check.label),
        h('div', { class: 'actions' },
          cancel && h('button', { onclick: () => done(false) }, cancel),
          h('button', { class: danger ? 'danger-solid' : 'primary', onclick: () => done(true) }, ok))));
    document.addEventListener('keydown', onKey);
    document.body.append(scrim);
  });
}
const ask = async (title, body, ok = 'Aplicar') => (await dialog(title, [h('p', {}, body), h('p', {}, 'Se guarda el valor original y podés revertirlo cuando quieras.')], { ok })).ok;

// ---------- gauge ----------
function gauge(id, label) {
  const R = 30, C = 2 * Math.PI * R;
  const fg = svg('circle', { class: 'fg', cx: 37, cy: 37, r: R, fill: 'none', 'stroke-width': 7, 'stroke-dasharray': C, 'stroke-dashoffset': C });
  const s = svg('svg', { viewBox: '0 0 74 74' },
    svg('defs', {}, svg('linearGradient', { id: 'g' + id, x1: 0, y1: 0, x2: 1, y2: 1 },
      svg('stop', { offset: '0', 'stop-color': '#6c8cff' }), svg('stop', { offset: '1', 'stop-color': '#9b6cff' }))),
    svg('circle', { class: 'bg', cx: 37, cy: 37, r: R, fill: 'none', 'stroke-width': 7 }), fg);
  fg.style.stroke = `url(#g${id})`;
  const num = h('div', { class: 'num' }, '–');
  el[id] = { fg, num, C };
  return h('div', { class: 'glass gauge' }, s, h('div', {}, h('div', { class: 'k' }, label), num));
}
function setGauge(id, v) {
  const g = el[id]; if (!g || !g.fg.isConnected) return;
  g.fg.setAttribute('stroke-dashoffset', g.C * (1 - Math.min(100, Math.max(0, v)) / 100));
  g.num.textContent = Math.round(v) + '%';
}

// ---------- ajustes ----------
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

function paintSettingsList() {
  if (!el.list || !el.list.isConnected) return;
  const frag = document.createDocumentFragment();
  const cats = [...new Set(S.tweaks.filter(visible).map((t) => t.category))];
  for (const c of cats) {
    frag.append(h('div', { class: 'section' }, h('h2', {}, c)));
    for (const t of S.tweaks.filter((x) => x.category === c && visible(x))) frag.append(S.rows.get(t.id).row);
  }
  if (!cats.length) frag.append(h('p', { class: 'k', style: 'padding:30px 0;text-align:center' }, 'Nada coincide con tu búsqueda.'));
  el.list.replaceChildren(frag);
}

function paintChips() {
  if (!el.chips || !el.chips.isConnected) return;
  const cats = ['Todos', ...new Set(S.tweaks.map((t) => t.category))];
  el.chips.replaceChildren(...cats.map((c) => h('button', {
    class: 'chip' + (c === S.cat ? ' on' : ''),
    onclick: () => { S.cat = c; paintChips(); paintSettingsList(); },
  }, c, h('i', {}, c === 'Todos' ? S.tweaks.length : S.tweaks.filter((t) => t.category === c).length))));
}

function paintState() {
  for (const t of S.tweaks) { const r = S.rows.get(t.id); if (r) paintRow(r, t); }
  if (el.revertAll) {
    el.revertAll.disabled = S.busy || !S.tweaks.some((t) => t.canUndo);
    el.simBox.disabled = S.busy;
    el.simBox.checked = S.sim;
    el.busy.hidden = !S.busy;
  }
  if (el.simBanner) el.simBanner.hidden = !S.sim;
  if (el.adminBanner) el.adminBanner.hidden = !(S.info && !S.info.admin && S.tweaks.some((t) => t.blocked));
  if (el.activeCount) el.activeCount.textContent = `${S.tweaks.filter((t) => t.active).length} de ${S.tweaks.filter((t) => !t.disabled).length}`;
  document.querySelectorAll('[data-needbusy]').forEach((b) => { b.disabled = S.busy; });
}

async function refresh() {
  const list = (await ListTweaks()) || [];
  const newIds = list.map((t) => t.id).join();
  const rebuild = newIds !== S.tweaks.map((t) => t.id).join();
  S.tweaks = list;
  if (rebuild) { S.rows.clear(); for (const t of list) buildRow(t); }
  paintState();
  if (rebuild) { paintChips(); paintSettingsList(); }
}

async function run(fn, id, okMsg) {
  if (S.busy) return;
  S.busy = true; paintState();
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

// ---------- perfiles + punto de restauración ----------
async function applyProfile(p) {
  if (S.busy) return;
  const canRestore = S.info.admin;
  const list = p.tweaks.map((id) => S.tweaks.find((t) => t.id === id)).filter((t) => t && !t.disabled);
  const pending = list.filter((t) => !t.active);
  const body = [
    h('p', {}, p.description),
    h('p', { class: 'k', style: 'margin:10px 0 4px' }, `${pending.length} ajustes por aplicar, ${list.length - pending.length} ya activos`),
    h('ul', { class: 'mini' }, pending.slice(0, 8).map((t) => h('li', {}, t.title)), pending.length > 8 && h('li', { class: 'k' }, `y ${pending.length - 8} más…`)),
    !canRestore && p.needsAdmin && h('p', { class: 'warn' }, 'Sin administrador, los ajustes que lo requieren se omitirán.'),
  ];
  const r = await dialog(`Aplicar perfil «${p.title}»`, body, { ok: 'Aplicar', check: canRestore ? { label: 'Crear un punto de restauración antes', checked: true } : null });
  if (!r.ok) return;
  S.busy = true; paintState();
  try {
    if (r.checked) {
      const err = await CreateRestorePoint();
      if (err) {
        S.busy = false; paintState();
        const go = await dialog('No se pudo crear el punto de restauración', [h('p', {}, err), h('p', {}, 'Podés continuar igual: cada ajuste guarda su valor original y se puede revertir desde la app.')], { ok: 'Continuar', cancel: 'Cancelar' });
        if (!go.ok) return;
        S.busy = true; paintState();
      }
    }
    const res = await ApplyProfile(p.id);
    if (res.error) { toast(res.error, 'err'); return; }
    const sec = (title, arr) => arr && arr.length ? [h('p', { class: 'k', style: 'margin:12px 0 4px' }, `${title} (${arr.length})`), h('ul', { class: 'mini' }, arr.map((x) => h('li', {}, x)))] : null;
    S.busy = false;
    await dialog(`Perfil «${p.title}»`, [sec('Aplicados', res.applied), sec('Omitidos', res.skipped), sec('Con error', res.failed),
      !res.applied?.length && !res.failed?.length && h('p', {}, 'No había nada nuevo para aplicar.')], { ok: 'Listo', cancel: null });
  } catch (e) {
    toast(String(e), 'err');
  } finally {
    S.busy = false;
    await refresh();
  }
}

async function makeRestorePoint() {
  if (!S.info.admin) { toast('Necesitás ejecutar como administrador.', 'err'); return; }
  S.busy = true; paintState();
  try {
    const err = await CreateRestorePoint();
    if (err) toast(err, 'err'); else toast('Punto de restauración creado', 'ok');
  } finally { S.busy = false; paintState(); }
}

// ---------- páginas ----------
function pageHome() {
  const i = S.info;
  const spec = h('div', { class: 'glass spec spec-card' },
    ...[['CPU', `${i.cpu || 'Desconocido'} · ${i.cores} hilos`], ['GPU', (i.gpus || []).join(' / ') || 'No detectada'],
        ['RAM', `${i.ramGB.toFixed(1)} GB`], ['Windows', i.windows]]
      .map(([k, v]) => h('div', {}, h('span', { class: 'k' }, k), h('b', {}, v))));
  el.activeCount = h('b', {}, '–');
  const profileCards = S.profiles.map((p) => h('div', { class: 'glass pcard' },
    h('div', { class: 'ptop' }, h('h3', {}, p.title), p.needsAdmin && h('span', { class: 'tag adm' }, 'Admin')),
    h('p', { class: 'd' }, p.description),
    h('div', { class: 'row-end' }, h('span', { class: 'k' }, `${p.tweaks.length} ajustes`),
      h('button', { class: 'primary', 'data-needbusy': 1, onclick: () => applyProfile(p) }, 'Aplicar'))));
  return [
    pageHead('Inicio', 'Estado de tu equipo y optimización rápida.'),
    h('div', { class: 'stats' }, gauge('cpu', 'CPU'), gauge('ram', 'RAM'), spec),
    h('div', { class: 'section' }, h('h2', {}, 'Optimización rápida')),
    h('div', { class: 'pgrid' }, profileCards),
    h('div', { class: 'section' }, h('h2', {}, 'Seguridad')),
    h('div', { class: 'glass safety' },
      h('div', {}, h('b', {}, 'Todo es reversible'),
        h('p', { class: 'd' }, 'Cada ajuste guarda su valor original antes de aplicarse. Además podés crear un punto de restauración de Windows antes de empezar.'),
        h('p', { class: 'd' }, 'Ajustes activos: ', el.activeCount)),
      h('div', { class: 'row' }, h('button', { 'data-needbusy': 1, onclick: makeRestorePoint }, 'Crear punto de restauración'),
        h('button', { class: 'danger', 'data-needbusy': 1, onclick: async () => {
          if ((await dialog('Revertir todo', 'Todos los ajustes aplicados vuelven a sus valores originales.', { ok: 'Revertir', danger: true })).ok) run(RevertAll, undefined, 'Todo revertido');
        } }, 'Revertir todo'))),
  ];
}

function pageHead(title, sub, right) {
  return h('div', { class: 'top' }, h('div', {}, h('h1', {}, title), h('p', {}, sub)), right);
}

function pageSettings() {
  el.search = h('input', { class: 'search', placeholder: 'Buscar ajustes…', value: S.q, oninput: (e) => { S.q = e.target.value; paintSettingsList(); } });
  el.chips = h('div', { class: 'chips' });
  el.list = h('div');
  const simBox = h('input', { type: 'checkbox', checked: S.sim, onchange: async (e) => { await SetSimulation(e.target.checked); S.sim = e.target.checked; await refresh(); } });
  el.simBox = simBox;
  el.revertAll = h('button', { class: 'danger', onclick: async () => {
    if ((await dialog('Revertir todo', 'Todos los ajustes aplicados vuelven a sus valores originales.', { ok: 'Revertir', danger: true })).ok) run(RevertAll, undefined, 'Todo revertido');
  } }, 'Revertir todo');
  el.simBanner = h('div', { class: 'banner info', hidden: true }, 'Modo simulación: no se modifica nada en tu Windows.');
  setTimeout(() => { paintChips(); paintSettingsList(); paintState(); }, 0);
  return [
    pageHead('Ajustes', 'Cada ajuste es reversible y guarda su valor original.', el.search),
    el.simBanner,
    h('div', { class: 'toolbar' }, el.chips,
      h('div', { class: 'row' }, h('label', { class: 'chk' }, simBox, 'Modo simulación'), el.revertAll)),
    el.list,
  ];
}

function pageClean() {
  const wrap = h('div');
  const render = () => {
    const items = S.clean;
    if (!items) { wrap.replaceChildren(h('div', { class: 'glass empty' }, h('div', { class: 'spin' }), ' Calculando tamaños…')); return; }
    const sel = new Set(items.filter((i) => i.bytes > 0 && !(i.admin && !S.info.admin)).map((i) => i.id));
    const totalEl = h('b', {});
    const btn = h('button', { class: 'primary', 'data-needbusy': 1 }, 'Limpiar');
    const update = () => {
      const total = items.filter((i) => sel.has(i.id)).reduce((a, i) => a + i.bytes, 0);
      totalEl.textContent = fmtBytes(total);
      btn.disabled = S.busy || sel.size === 0;
    };
    const rows = items.map((i) => {
      const blocked = i.admin && !S.info.admin;
      const cb = h('input', { type: 'checkbox', checked: sel.has(i.id), disabled: blocked || i.bytes === 0, onchange: (e) => { e.target.checked ? sel.add(i.id) : sel.delete(i.id); update(); } });
      return h('div', { class: 'tweak' + (blocked ? ' off' : '') },
        h('div', { class: 'info' }, h('div', { class: 't' }, i.title, i.admin && h('span', { class: 'tag adm' }, 'Admin')), h('div', { class: 'd' }, i.desc)),
        h('div', { class: 'size' }, h('b', {}, fmtBytes(i.bytes)), h('span', { class: 'k' }, `${i.files} archivos`)),
        h('label', { class: 'switch' }, cb, h('span')));
    });
    btn.onclick = async () => {
      const chosen = items.filter((i) => sel.has(i.id));
      const r = await dialog('Limpiar archivos', [
        h('p', {}, `Se borrarán ${fmtBytes(chosen.reduce((a, i) => a + i.bytes, 0))} de:`),
        h('ul', { class: 'mini' }, chosen.map((i) => h('li', {}, i.title))),
        h('p', { class: 'warn' }, 'Esta acción no se puede deshacer. Los archivos modificados en la última hora y los que están en uso se dejan.')], { ok: 'Borrar', danger: true });
      if (!r.ok) return;
      S.busy = true; paintState();
      try {
        const res = await RunCleanup([...sel]);
        if (res.error) toast(res.error, 'err');
        else toast(`Liberaste ${fmtBytes(res.freed)}`, 'ok');
        if (res.errors?.length) console.warn(res.errors);
      } finally { S.busy = false; S.clean = null; render(); loadClean(); paintState(); }
    };
    wrap.replaceChildren(
      h('div', { class: 'glass safety' }, h('div', {}, h('b', {}, 'Se pueden liberar '), totalEl, h('p', { class: 'd' }, 'Solo se limpian carpetas temporales conocidas. Los archivos recientes o en uso se respetan.')),
        h('div', { class: 'row' }, h('button', { 'data-needbusy': 1, onclick: () => { S.clean = null; render(); loadClean(); } }, 'Volver a calcular'), btn)),
      h('div', { style: 'height:12px' }), ...rows);
    update();
  };
  const loadClean = async () => { S.clean = await ScanCleanup(); if (el.content.contains(wrap)) render(); };
  render();
  if (!S.clean) loadClean();
  return [pageHead('Limpieza', 'Libera espacio de archivos temporales y cachés. Vas a ver el tamaño antes de borrar.'), wrap];
}

function pageStartup() {
  const wrap = h('div');
  const render = () => {
    const items = S.startup;
    if (!items) { wrap.replaceChildren(h('div', { class: 'glass empty' }, h('div', { class: 'spin' }), ' Leyendo programas de inicio…')); return; }
    if (!items.length) { wrap.replaceChildren(h('div', { class: 'glass empty' }, 'No hay programas de inicio registrados.')); return; }
    wrap.replaceChildren(...items.map((it) => {
      const blocked = it.admin && !S.info.admin;
      const input = h('input', { type: 'checkbox', checked: it.enabled, disabled: blocked, onchange: async (e) => {
        const want = e.target.checked;
        const err = await SetStartup(it.id, want);
        if (err) { toast(err, 'err'); e.target.checked = !want; return; }
        it.enabled = want; toast(want ? 'Se iniciará con Windows' : 'Ya no se inicia con Windows', 'ok');
      } });
      return h('div', { class: 'tweak' + (blocked ? ' off' : ''), title: blocked ? 'Requiere administrador' : '' },
        h('div', { class: 'info' }, h('div', { class: 't' }, it.name, h('span', { class: 'tag' }, it.label)), h('div', { class: 'd mono' }, it.command)),
        h('label', { class: 'switch' }, input, h('span')));
    }));
  };
  render();
  ListStartup().then((l) => { S.startup = l || []; if (el.content.contains(wrap)) render(); });
  return [pageHead('Inicio de Windows', 'Desactivá programas que arrancan con Windows. No se borra nada: podés volver a activarlos.'), wrap];
}

const PAGES = [
  { id: 'home', label: 'Inicio', icon: 'home', render: pageHome },
  { id: 'settings', label: 'Ajustes', icon: 'sliders', render: pageSettings },
  { id: 'clean', label: 'Limpieza', icon: 'trash', render: pageClean },
  { id: 'startup', label: 'Inicio de Windows', icon: 'rocket', render: pageStartup },
];

function go(id) {
  S.page = id;
  S.startup = id === 'startup' ? null : S.startup; // siempre se relee al entrar
  if (id === 'clean') S.clean = null;
  el.nav.replaceChildren(...PAGES.map((p) => h('button', { class: 'nav' + (p.id === id ? ' on' : ''), onclick: () => go(p.id) }, icon(ICONS[p.icon]), p.label)));
  const page = PAGES.find((p) => p.id === id);
  el.content.replaceChildren(h('div', { class: 'page' }, page.render()));
  el.main.scrollTo({ top: 0 });
  paintState();
  tick();
}

// ---------- layout ----------
function build() {
  el.nav = h('nav');
  el.content = h('div');
  el.busy = h('div', { class: 'busy-bar', hidden: true });
  el.adminBanner = h('div', { class: 'banner', hidden: true }, 'Algunos ajustes necesitan permisos de administrador.',
    h('button', { onclick: async () => { const e = await RelaunchAsAdmin(); if (e) toast(e, 'err'); } }, 'Reiniciar como administrador'));
  el.main = h('main', {}, el.adminBanner, el.content);
  document.getElementById('app').replaceChildren(
    h('aside', {},
      h('div', { class: 'brand' }, h('div', { class: 'logo' }, 'FPS'), h('div', {}, h('b', {}, 'FPS Booster Pro'), h('small', {}, 'v2 · Go'))),
      el.nav,
      h('div', { class: 'side-foot' }, h('small', { class: 'k', style: 'text-transform:none;letter-spacing:0' }, S.info.admin ? 'Ejecutando como administrador' : 'Sin permisos de administrador'))),
    el.main);
  document.body.append(el.busy, toasts);
}

async function tick() {
  try { const l = await GetLive(); S.live = l; setGauge('cpu', l.cpu); setGauge('ram', l.ram); } catch { /* cerrando */ }
}

(async () => {
  const [initErr, info, sim] = await Promise.all([InitError(), GetSystemInfo(), GetSimulation()]);
  S.info = info; S.sim = sim;
  if (initErr) {
    document.getElementById('app').replaceChildren(h('div', { class: 'banner err', style: 'margin:30px;grid-column:1/-1' }, 'No se pudo iniciar de forma segura: ' + initErr));
    return;
  }
  build();
  S.profiles = (await ListProfiles()) || [];
  await refresh();
  go('home');
  setInterval(tick, 1500);
})();
