import './style.css';
import {
  GetSystemInfo, GetLive, ListTweaks, ApplyTweak, RevertTweak, RevertAll,
  SetSimulation, GetSimulation, RelaunchAsAdmin, InitError,
  CreateRestorePoint,
  ScanCleanup, RunCleanup, ListStartup, SetStartup,
  CurrentGame, RunOnce, GameLibrary, GameCover, AppIcons,
  DiscordState, DiscordLogin, DiscordLogout, SetViewAs, ListFixes, FixCheck, RunFix,
} from '../wailsjs/go/main/App';
import { h, icon, fmtBytes, zone, worst, GLOW, colorAt, CAT_ICON, TWEAK_ICON } from './lib.js';
import { ringGauge } from './gauge.js';
import { lineChart } from './chart.js';
import { initReveals, reveal, revealPage, swapPage, splitTitle, tweenNumber, bump, initPointerFX, initScrollParallax, initTooltips } from './motion.js';

// ---------- estado ----------
const HIST_MAX = 150;                       // 150 muestras x 2 s = 5 minutos
const S = {
  info: null, scanAt: 0, scanning: false, tweaks: [], cat: 'Todos', q: '', sim: false, busy: false, page: 'home',
  rows: new Map(), clean: null, startup: null, live: null, game: null,
  hist: { cpu: [], ram: [], gpu: [], disk: [] }, range: 30, visible: { cpu: true, ram: true, gpu: true, disk: false },
};
const el = {};
let paintLiveFns = [];                     // funciones de la página actual que se repintan con cada muestra

// ---------- toasts y diálogos ----------
const toasts = h('div', { class: 'toasts' });
function toast(msg, kind = '') {
  const t = h('div', { class: 'toast ' + kind }, msg);
  toasts.append(t);
  setTimeout(() => { t.style.transition = 'opacity .3s'; t.style.opacity = '0'; setTimeout(() => t.remove(), 300); }, kind === 'err' ? 6500 : 2600);
}
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

// ---------- salud y estados ----------
const ZONES = { temp: [70, 85], cpu: [75, 92], ram: [78, 90], disk: [80, 92] };

function health(l) {
  let score = 100; const issues = [];
  if (!l) return { score: 100, issues };
  if (l.ram > 90) { score -= 20; issues.push('La RAM está casi llena'); } else if (l.ram > 80) { score -= 8; issues.push('Mucha RAM en uso'); }
  if (l.cpu > 95) { score -= 10; issues.push('La CPU está al límite'); }
  if (l.gpu && l.gpu.temp >= 85) { score -= 25; issues.push('La GPU está muy caliente'); } else if (l.gpu && l.gpu.temp >= 75) { score -= 10; issues.push('La GPU está caliente'); }
  for (const d of l.disks || []) {
    const free = 100 - d.usedPct;
    if (free < 10) { score -= 15; issues.push(`${d.letter} casi sin espacio`); } else if (free < 20) { score -= 5; issues.push(`${d.letter} con poco espacio`); }
  }
  return { score: Math.max(0, score), issues };
}
const healthState = (s) => (s >= 85 ? 'cool' : s >= 60 ? 'warm' : 'hot');
const healthText = (s) => (s >= 85 ? 'Bueno' : s >= 60 ? 'Atención' : 'Crítico');

function systemDisk(l) {
  const ds = (l && l.disks) || [];
  return ds.find((d) => d.letter === 'C:') || ds[0] || null;
}

// ---------- ajustes ----------
const RISK = { safe: 'Seguro', moderate: 'Moderado' };
const BRAND = { nvidia: 'NVIDIA', amd: 'AMD', intel: 'Intel' };
const brandOf = (t) => t.vendor || (t.nvidia && t.nvidia.length ? 'nvidia' : '');
const catIco = (c) => h('div', { class: 'cat-ico', 'data-cat': c }, icon(CAT_ICON[c] || 'cog'));
const tweakIco = (t) => h('div', { class: 'cat-ico', 'data-cat': t.category }, icon(TWEAK_ICON[t.id] || CAT_ICON[t.category] || 'cog'));
const dangerMark = (t) => h('button', {
  type: 'button', class: 'danger-mark', 'aria-label': 'Advertencia: ' + t.warn,
  'data-tip-title': 'Desactiva algo importante', 'data-tip': t.warn, 'data-tip-tone': 'warn',
  onclick: (e) => e.stopPropagation(),
}, icon('alert'));
const fmtWhen = (iso) => new Date(iso).toLocaleString('es-AR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });

// ---------- cola de cambios ----------
// Nada se aplica al mover un interruptor: queda «en cola» y se aplica todo junto desde la barra
// de abajo. La cola guarda solo lo que DIFIERE del estado real (activar, desactivar o acción única).
const Q = new Map();                            // id -> 'apply' | 'revert' | 'once'
const qTweak = (id) => S.tweaks.find((t) => t.id === id);

function queueSet(t, action) {
  // action: 'apply' | 'revert' | 'once' | null (quitar)
  if (action === null) Q.delete(t.id); else Q.set(t.id, action);
  paintQueue();
  const r = S.rows.get(t.id); if (r) paintRow(r, t);
}

function pruneQueue() {
  // Si el estado real ya coincide con lo pedido (p. ej. tras aplicar), se saca de la cola.
  for (const [id, a] of [...Q]) {
    const t = qTweak(id);
    if (!t || t.disabled || (a === 'apply' && t.active) || (a === 'revert' && !t.active)) Q.delete(id);
  }
}

function paintQueue() {
  if (!el.queue) return;
  const items = [...Q].map(([id, a]) => ({ t: qTweak(id), a })).filter((x) => x.t);
  el.queue.hidden = items.length === 0;
  if (!items.length) return;
  const n = (a) => items.filter((x) => x.a === a).length;
  const parts = [n('apply') && `${n('apply')} por activar`, n('revert') && `${n('revert')} por desactivar`,
    n('once') && `${n('once')} acción${n('once') > 1 ? 'es' : ''} única${n('once') > 1 ? 's' : ''}`].filter(Boolean);
  if (el.queueCount.textContent !== String(items.length)) bump(el.queueCount);
  el.queueCount.textContent = String(items.length);
  el.queueLabel.textContent = items.length === 1 ? 'proceso por aplicar' : 'procesos por aplicar';
  el.queueParts.textContent = parts.join(' · ');
  el.queueApply.disabled = S.busy;
  el.queueClear.disabled = S.busy;
}

const QLABEL = { apply: 'Activar', revert: 'Desactivar', once: 'Ejecutar' };

async function applyQueue() {
  if (S.busy || !Q.size) return;
  pruneQueue();
  // Orden: primero lo que se desactiva, después lo que se activa y al final las acciones únicas.
  const rank = { revert: 0, apply: 1, once: 2 };
  const items = [...Q].map(([id, a]) => ({ t: qTweak(id), a })).filter((x) => x.t).sort((x, y) => rank[x.a] - rank[y.a]);
  if (!items.length) { paintQueue(); return; }
  const canRestore = S.info.admin && items.some((x) => x.a !== 'revert');
  const onces = items.filter((x) => x.a === 'once');
  const body = [
    h('p', {}, `Se van a ejecutar ${items.length} proceso${items.length > 1 ? 's' : ''}, uno detrás del otro.`),
    h('ul', { class: 'mini' }, items.slice(0, 10).map((x) => h('li', {}, h('b', {}, QLABEL[x.a] + ': '), x.t.title)),
      items.length > 10 && h('li', { class: 'k' }, `y ${items.length - 10} más…`)),
    onces.length > 0 && h('p', { class: 'warn' }, `${onces.length === 1 ? 'Hay 1 acción única' : `Hay ${onces.length} acciones únicas`} que no se pueden deshacer.`),
  ];
  const risky = items.filter((x) => x.a !== 'revert' && x.t.warn);
  if (risky.length) {
    body.push(h('p', { class: 'warn' }, `${risky.length === 1 ? 'Este cambio desactiva' : 'Estos cambios desactivan'} algo importante:`),
      h('ul', { class: 'mini warnlist' }, risky.map((x) => h('li', {}, h('b', {}, x.t.title + ': '), x.t.warn))));
  }
  const r = await dialog('Aplicar cambios', body, { ok: 'Aplicar', check: canRestore ? { label: 'Crear un punto de restauración antes', checked: true } : null });
  if (!r.ok) return;
  runQueue(items, r.checked);
}

function runQueue(items, restorePoint) {
  S.busy = true; paintState(); paintQueue();
  const rows = items.map((x) => {
    const st = h('span', { class: 'qs wait' });
    const li = h('li', { class: 'qi' }, st, h('span', { class: 'qt' }, x.t.title), h('span', { class: 'qa' }, QLABEL[x.a]));
    return { ...x, st, li };
  });
  const bar = h('div', { class: 'pbar-fill' });
  const head = h('h3', {}, 'Aplicando cambios…');
  const cur = h('p', { class: 'qcur' }, 'Preparando…');
  const pct = h('b', { class: 'qpct' }, '0%');
  const foot = h('div', { class: 'actions' });
  const list = h('ul', { class: 'qlist' }, rows.map((x) => x.li));
  const scrim = h('div', { class: 'scrim loading' },
    h('div', { class: 'modal loader' }, head, cur,
      h('div', { class: 'pbar' }, bar), h('div', { class: 'pinfo' }, pct, h('span', { class: 'k qn' }, `0 de ${rows.length}`)),
      list, foot));
  document.body.append(scrim);
  const qn = scrim.querySelector('.qn');
  const setProgress = (done) => {
    const p = Math.round((done / rows.length) * 100);
    bar.style.width = p + '%'; pct.textContent = p + '%'; qn.textContent = `${done} de ${rows.length}`;
  };
  const mark = (x, cls, txt) => { x.st.className = 'qs ' + cls; if (txt) x.li.title = txt; };

  (async () => {
    const res = { ok: 0, fail: [], reboot: false };
    try {
      if (restorePoint) {
        cur.textContent = 'Creando punto de restauración…';
        const err = await CreateRestorePoint();
        if (err) toast('No se creó el punto de restauración: ' + err, 'err');
      }
      let done = 0;
      for (const x of rows) {
        cur.textContent = `${QLABEL[x.a]}: ${x.t.title}`;
        x.st.className = 'qs run';
        x.li.scrollIntoView({ block: 'nearest' });
        let err = '', msg = '';
        try {
          if (x.a === 'once') { const o = await RunOnce(x.t.id); err = o.error; msg = o.message; }
          else { const o = await (x.a === 'apply' ? ApplyTweak : RevertTweak)(x.t.id); err = o.error; msg = o.message; }
        } catch (e) { err = String(e); }
        if (err) { mark(x, 'bad', err); res.fail.push(`${x.t.title}: ${err}`); x.li.append(h('span', { class: 'qerr' }, err)); }
        else { mark(x, 'ok', msg); res.ok++; if (x.t.reboot) res.reboot = true; if (msg) x.li.append(h('span', { class: 'qerr okmsg' }, msg)); }
        setProgress(++done);
        await new Promise((r) => setTimeout(r, 120));      // deja ver el avance aunque el ajuste sea instantáneo
      }
    } finally {
      S.busy = false;
      for (const x of rows) { const cur2 = Q.get(x.t.id); if (cur2 === x.a && !res.fail.some((f) => f.startsWith(x.t.title + ':'))) Q.delete(x.t.id); }
      await refresh();
      pruneQueue(); paintQueue();
    }
    head.textContent = res.fail.length ? 'Terminó con errores' : 'Listo';
    cur.textContent = `${res.ok} aplicado${res.ok === 1 ? '' : 's'}` + (res.fail.length ? `, ${res.fail.length} con error` : '') +
      (res.reboot ? '. Reiniciá Windows para que se noten todos los cambios.' : '.');
    scrim.classList.add('finished');
    foot.append(h('button', { class: 'primary', onclick: () => scrim.remove() }, 'Cerrar'));
  })();
}

function buildRow(t) {
  const brand = brandOf(t);
  let input = null, btn = null, control;
  if (t.once) {
    btn = h('button', {
      class: 'sm', onclick: () => {
        const cur = S.rows.get(t.id).t;
        queueSet(cur, Q.get(cur.id) === 'once' ? null : 'once');
      },
    }, 'Agregar');
    control = btn;
  } else {
    input = h('input', {
      type: 'checkbox',
      onchange: (e) => {
        const cur = S.rows.get(t.id).t;
        const want = e.target.checked;
        queueSet(cur, want === cur.active ? null : (want ? 'apply' : 'revert'));
      },
    });
    control = h('label', { class: 'switch' }, input, h('span'));
  }
  const tags = h('div', { class: 'tc-tags' });
  const last = h('span', { class: 'tc-last' });
  const detail = h('p', { class: 'tc-detail' });
  const row = h('div', { class: 'tcard spot', 'data-vendor': brand || null },
    h('div', { class: 'tc-head' }, tweakIco(t), h('div', { class: 'tc-title' }, t.title),
      brand && h('span', { class: 'vbadge ' + brand, 'data-tip': `Este ajuste solo se aplica a placas ${BRAND[brand]}.` }, BRAND[brand]),
      t.warn && dangerMark(t)),
    h('div', { class: 'tc-rule' }),
    h('p', { class: 'tc-desc' }, t.description),
    detail,
    h('div', { class: 'tc-foot' }, h('div', { class: 'tc-tags' }, tags, last), control));
  const lock = h('button', { class: 'lock-cover', hidden: true, onclick: () => upgradeDialog(S.rows.get(t.id).t.title) },
    icon('lock'), h('b', {}, 'Pro Tier'), h('span', {}, 'Upgrade · próximamente'));
  row.append(lock);
  // Las explicaciones no se ven en la lista: aparecen al poner el ajuste en la cola o, si no aplica a
  // esta PC, al tocar la tarjeta (para saber por qué).
  row.addEventListener('click', (ev) => {
    if (ev.target.closest('.switch, button, input')) return;
    const cur = S.rows.get(t.id).t;
    if (cur.disabled) row.classList.toggle('show-why');
  });
  const r = { row, input, btn, tags, last, t, lock, detail };
  S.rows.set(t.id, r);
  return r;
}

function paintRow(r, t) {
  r.t = t;
  r.row.classList.toggle('on', !!t.active);
  r.row.classList.toggle('off', !!t.disabled);
  r.row.classList.toggle('locked', !!t.locked);
  r.lock.hidden = !t.locked;
  const blocked = t.disabled || (t.blocked && !t.active);
  const q = Q.get(t.id);
  r.row.classList.toggle('queued', !!q);
  if (r.input) { r.input.checked = q ? q === 'apply' : t.active; r.input.disabled = S.busy || blocked || (t.locked && !t.active); }
  if (r.btn) {
    r.btn.disabled = S.busy || t.disabled || t.blocked || t.locked;
    r.btn.textContent = q === 'once' ? 'En la cola ✓' : 'Agregar';
    r.btn.classList.toggle('primary', q === 'once');
  }
  r.row.title = t.locked ? 'Incluido en Pro Tier' : t.disabled ? 'No aplica a tu equipo' : (t.blocked ? 'Requiere administrador' : '');
  r.detail.textContent = S.scanAt ? (t.detail || '') : '';
  r.detail.className = 'tc-detail ' + (t.status || '') + (S.scanning ? ' stale' : '');
  r.last.textContent = t.once && t.lastRun ? `Última vez: ${fmtWhen(t.lastRun)}` : '';
  r.tags.replaceChildren(...[
    t.locked && h('span', { class: 'tag pro' }, 'Pro'),
    t.once ? h('span', { class: 'tag once' }, 'Una vez') : h('span', { class: 'tag ' + t.risk }, RISK[t.risk] || t.risk),
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
  if (el.upgradeBanner) {
    const n = S.tweaks.filter((t) => t.locked).length;
    el.upgradeBanner.hidden = !n;
    el.upgradeBanner.replaceChildren(icon('lock'), h('div', {}, h('b', {}, 'Free Tier: ajustes básicos'),
      h('p', {}, `Tenés ${S.tweaks.filter((t) => !t.locked && !t.disabled).length} ajustes disponibles. ${n} más están en Pro Tier (el upgrade llega pronto).`)));
  }
  for (const c of cats) {
    const items = S.tweaks.filter((x) => x.category === c && visible(x)).sort((a, b) => (a.locked ? 1 : 0) - (b.locked ? 1 : 0));
    frag.append(h('div', { class: 'section' }, catIco(c), h('h2', {}, c), h('span', { class: 'count' }, items.length)));
    frag.append(h('div', { class: 'tgrid' }, items.map((t) => S.rows.get(t.id).row)));
  }
  if (!cats.length) frag.append(h('div', { class: 'glass empty' }, icon('search'), 'Nada coincide con tu búsqueda.'));
  el.list.replaceChildren(frag);
  // las tarjetas nuevas (o que todavía no entraron) se revelan al aparecer en pantalla
  el.list.querySelectorAll('.tgrid').forEach((g) => reveal([...g.children], { base: 40, step: 35 }));
  reveal([...el.list.querySelectorAll(':scope > .section, :scope > .empty')], { kind: 'fade', step: 30 });
}

function paintChips() {
  if (!el.chips || !el.chips.isConnected) return;
  const cats = ['Todos', ...new Set(S.tweaks.map((t) => t.category))];
  el.chips.replaceChildren(...cats.map((c) => h('button', {
    class: 'chip' + (c === S.cat ? ' on' : ''),
    onclick: () => { S.cat = c; paintChips(); paintSettingsList(); },
  }, icon(CAT_ICON[c] || 'cog'), c, h('i', {}, c === 'Todos' ? S.tweaks.length : S.tweaks.filter((t) => t.category === c).length))));
}

function paintState() {
  for (const t of S.tweaks) { const r = S.rows.get(t.id); if (r) paintRow(r, t); }
  if (el.revertAll && el.revertAll.isConnected) {
    el.revertAll.disabled = S.busy || !S.tweaks.some((t) => t.canUndo);
    el.simBox.disabled = S.busy;
    el.simBox.checked = S.sim;
  }
  el.busy.hidden = !S.busy;
  if (el.simBanner) el.simBanner.hidden = !S.sim;
  if (el.adminBanner) el.adminBanner.hidden = !(S.info && !S.info.admin && S.tweaks.some((t) => t.blocked));
  if (el.activeCount && el.activeCount.isConnected) el.activeCount.textContent = `${S.tweaks.filter((t) => t.active).length} de ${S.tweaks.filter((t) => !t.disabled).length}`;
  document.querySelectorAll('[data-needbusy]').forEach((b) => { b.disabled = S.busy; });
  paintQueue();
}

// Leer la PC: todo lo que la app muestra (estados, explicaciones, hardware) sale de esta lectura del
// equipo donde se está ejecutando; nada viene guardado de otra computadora.
function paintScan() {
  if (!el.scanBar || !el.scanBar.isConnected) return;
  const i = S.info || {};
  el.scanBar.classList.toggle('busy', S.scanning);
  if (S.scanning) { el.scanBar.replaceChildren(h('div', { class: 'spin' }), h('span', {}, 'Leyendo tu PC…')); return; }
  if (!S.scanAt) { el.scanBar.replaceChildren(h('span', {}, 'Todavía no se leyó tu PC.')); return; }
  const when = new Date(S.scanAt).toLocaleTimeString('es-AR', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  el.scanBar.replaceChildren(
    h('div', { class: 'sb-txt' }, h('b', {}, 'Leído de tu equipo'), h('span', {}, [i.cpu, (i.gpus || [])[0], i.ramGB ? `${i.ramGB.toFixed(0)} GB RAM` : '', i.windows].filter(Boolean).join(' · ')), h('small', {}, `a las ${when}`)),
    h('button', { class: 'sm', 'data-needbusy': 1, onclick: () => refresh() }, 'Volver a leer'));
}

async function refresh() {
  S.scanning = true; paintScan();
  try { S.info = await GetSystemInfo(); } catch (e) { /* se conserva la lectura anterior */ }
  const list = (await ListTweaks()) || [];
  S.scanAt = Date.now(); S.scanning = false;
  const newIds = list.map((t) => t.id).join();
  const rebuild = newIds !== S.tweaks.map((t) => t.id).join();
  S.tweaks = list;
  if (rebuild) { S.rows.clear(); for (const t of list) buildRow(t); }
  pruneQueue();
  paintState();
  paintQueue();
  paintScan();
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

async function makeRestorePoint() {
  if (!S.info.admin) { toast('Necesitás ejecutar como administrador.', 'err'); return; }
  S.busy = true; paintState();
  try {
    const err = await CreateRestorePoint();
    if (err) toast(err, 'err'); else toast('Punto de restauración creado', 'ok');
  } finally { S.busy = false; paintState(); }
}

const revertAllFlow = async () => {
  if ((await dialog('Revertir todo', 'Todos los ajustes aplicados vuelven a sus valores originales.', { ok: 'Revertir', danger: true })).ok) run(RevertAll, undefined, 'Todo revertido');
};

// ---------- tarjetas de hardware ----------
function hwCard({ ico, title, sub, gauge, stats }) {
  return h('div', { class: 'glass hw spot' },
    h('div', { class: 'card-head' },
      h('div', { class: 'head-ico' }, icon(ico)),
      h('div', { class: 'head-txt' }, h('b', {}, title), h('span', { class: 'hw-s', title: sub }, sub))),
    h('div', { class: 'hw-body' }, gauge.el, stats));
}

// Fila de métrica con barra fina (como en los paneles de monitoreo). frac null = sin barra.
function metric(label, value, frac = null) {
  return h('div', { class: 'metric' },
    h('div', { class: 'm-top' }, h('span', {}, label), h('b', {}, value)),
    frac == null ? null : h('div', { class: 'm-bar' }, h('i', { style: `width:${Math.round(Math.min(1, Math.max(0, frac)) * 100)}%` })));
}

// Actualiza las métricas sin recrear las barras: así la barra anima su ancho en lugar de saltar.
function setMetrics(box, rows) {
  if (box.children.length !== rows.length) { box.replaceChildren(...rows.map((r) => metric(...r))); return; }
  rows.forEach(([label, value, frac], i) => {
    const m = box.children[i];
    m.querySelector('.m-top span').textContent = label;
    m.querySelector('.m-top b').textContent = value;
    const bar = m.querySelector('.m-bar i');
    if (bar && frac != null) bar.style.width = `${Math.round(Math.min(1, Math.max(0, frac)) * 100)}%`;
  });
}

// ---------- páginas ----------
const PAGE_NAME = { home: 'Inicio', games: 'Mis juegos', settings: 'Optimizar', fixes: 'Fixes', clean: 'Limpieza', startup: 'Inicio de Windows' };
function pageHead(title, sub, right) {
  return h('div', { class: 'top' },
    h('div', {},
      h('div', { class: 'crumb' }, icon('home'), 'South Tweaks', icon('chevron'), h('b', {}, title)),
      splitTitle('h1', title),
      sub && h('p', {}, sub)),
    right);
}

function pageHome() {
  const i = S.info;
  el.activeCount = h('b', {}, '–');
  const gCPU = ringGauge(), gGPU = ringGauge(), gRAM = ringGauge(), gDSK = ringGauge();
  const cpuStats = h('div', { class: 'hw-stats' }), gpuStats = h('div', { class: 'hw-stats' });
  const ramStats = h('div', { class: 'hw-stats' }), dskStats = h('div', { class: 'hw-stats' });
  const hFill = h('div', { class: 'hbar-fill' }), hVal = h('b', {}, '–'), hChip = h('div', { class: 'hchip' });
  const hIssues = h('ul', { class: 'hissues' });
  const healthCard = h('div', { class: 'glass health spot' },
    h('div', { class: 'k' }, 'Salud del equipo'),
    h('div', { class: 'h-score' }, hVal, h('span', {}, '/ 100')), hChip,
    h('div', { class: 'hbar' }, hFill), hIssues,
    h('div', { class: 'h-foot' },
      h('div', { class: 'kv' }, h('span', {}, 'Ajustes activos'), el.activeCount),
      h('div', { class: 'kv' }, h('span', {}, 'Permisos'), h('b', {}, i.admin ? 'Administrador' : 'Usuario')),
      h('div', { class: 'kv' }, h('span', {}, 'Windows'), h('b', {}, (i.windows || '').replace(/\s*\(build.*\)/, ''))),
      h('button', { class: 'sm', onclick: () => go('settings') }, icon('sliders'), 'Ir a Optimizar')));

  const cards = [
    hwCard({ ico: 'cpu', title: 'Procesador', sub: i.cpu || 'Desconocido', gauge: gCPU, stats: cpuStats }),
    hwCard({ ico: 'gpu', title: 'Tarjeta de video', sub: (i.gpus || [])[0] || 'No detectada', gauge: gGPU, stats: gpuStats }),
    hwCard({ ico: 'ram', title: 'Memoria RAM', sub: `${i.ramGB.toFixed(1)} GB instalados`, gauge: gRAM, stats: ramStats }),
    hwCard({ ico: 'disk', title: 'Almacenamiento', sub: 'Unidad del sistema', gauge: gDSK, stats: dskStats }),
  ];

  const repaint = () => {
    const l = S.live; if (!l) return;
    const g = l.gpu;
    gCPU.set({ frac: l.cpu / 100, state: zone(l.cpu, ZONES.cpu), num: l.cpu, unitText: '%', labelText: 'En uso' });
    setMetrics(cpuStats, [['Uso', `${Math.round(l.cpu)}%`, l.cpu / 100], ['Frecuencia', l.cpuMHz ? `${(l.cpuMHz / 1000).toFixed(2)} GHz` : '–'],
      ['Núcleos / hilos', `${i.cpuInfo?.cores || '–'} / ${i.cpuInfo?.threads || i.cores}`]]);
    if (g && g.temp >= 0) {
      gGPU.set({ frac: g.temp / 100, state: zone(g.temp, ZONES.temp), num: g.temp, unitText: '°C', labelText: 'Temperatura' });
      setMetrics(gpuStats, [['Uso', `${g.util}%`, g.util / 100], ['VRAM', `${(g.vramUsed / 1024).toFixed(1)} / ${(g.vramTotal / 1024).toFixed(0)} GB`, g.vramTotal ? g.vramUsed / g.vramTotal : 0],
        ['Consumo', g.power >= 0 ? `${g.power.toFixed(0)} W` : '–']]);
    } else {
      gGPU.set({ frac: null, num: null, labelText: 'Sin datos' });
      gpuStats.replaceChildren(h('div', { class: 'dim' }, 'Temperatura y uso en vivo disponibles solo en placas NVIDIA.'));
    }
    gRAM.set({ frac: l.ram / 100, state: zone(l.ram, ZONES.ram), num: l.ram, unitText: '%', labelText: 'En uso' });
    setMetrics(ramStats, [['Usada', `${l.ramUsedGB.toFixed(1)} GB`, l.ram / 100], ['Libre', `${(l.ramTotalGB - l.ramUsedGB).toFixed(1)} GB`],
      ['Total', `${l.ramTotalGB.toFixed(1)} GB`]]);
    const d = systemDisk(l);
    if (d) {
      gDSK.set({ frac: d.usedPct / 100, state: zone(d.usedPct, ZONES.disk), num: d.usedPct, unitText: '%', labelText: `Usado · ${d.letter}` });
      setMetrics(dskStats, [['Libre', `${d.freeGB.toFixed(0)} GB de ${d.totalGB.toFixed(0)} GB`, 1 - d.usedPct / 100], ['Actividad', `${Math.round(l.diskBusy)}%`, l.diskBusy / 100]]);
    }
    const hl = health(l), st = healthState(hl.score);
    hFill.style.width = hl.score + '%';
    hFill.style.background = `linear-gradient(90deg, ${colorAt(st, 0)}, ${colorAt(st, 1)})`;
    tweenNumber(hVal, hl.score);
    healthCard.dataset.state = st;
    hChip.textContent = healthText(hl.score);
    hChip.className = 'hchip ' + st;
    hIssues.replaceChildren(...(hl.issues.length ? hl.issues.map((x) => h('li', {}, x)) : [h('li', { class: 'ok' }, 'Todo en orden')]));
  };
  const panel = pcPanel();
  paintLiveFns = [repaint, panel.repaint];
  setTimeout(() => { repaint(); panel.repaint(); }, 0);

  const tile = (ico, title, sub, onclick, cat) => h('button', { class: 'tile spot', onclick },
    h('div', { class: 'tile-ico', 'data-cat': cat }, icon(ico)), h('div', { class: 'tile-t' }, h('b', {}, title), h('span', {}, sub)), icon('chevron', 'tile-go'));
  const who = S.discord && S.discord.loggedIn ? S.discord.name : (i.user || '');

  return [
    pageHead(`Bienvenido${who ? ', ' + who : ''}`, 'Así está tu PC en este momento. Todo lo que ves se lee de tu equipo en vivo.',
      h('div', { class: 'live-pill' }, h('span', { class: 'live-dot' }), 'En vivo')),
    h('div', { class: 'dash' }, healthCard, h('div', { class: 'hwgrid' }, cards)),
    h('div', { class: 'tiles' },
      tile('wrench', 'Fixes', 'Arreglos de Windows', () => go('fixes'), 'Sistema'),
      tile('trash', 'Limpiar archivos', 'Liberá espacio', () => go('clean'), 'Acciones únicas'),
      tile('gamepad', 'Mis juegos', 'Portadas y FPS', () => go('games'), 'Juegos'),
      tile('restore', 'Punto de restauración', 'Antes de cambiar nada', makeRestorePoint, 'Privacidad')),
    ...panel.nodes,
    h('div', { class: 'section' }, h('h2', {}, 'Seguridad')),
    h('div', { class: 'glass safety spot' },
      h('div', {}, h('b', {}, 'Todo es reversible'),
        h('p', { class: 'd' }, 'Cada ajuste guarda su valor original antes de aplicarse. Además podés crear un punto de restauración de Windows antes de empezar.'),
        h('p', { class: 'd' }, 'Si algo no te convence, «Revertir todo» devuelve cada valor a como estaba.')),
      h('div', { class: 'row' }, h('button', { class: 'danger', 'data-needbusy': 1, onclick: revertAllFlow }, 'Revertir todo'))),
  ];
}

// Detalle en tiempo real (gráfico + tarjetas). Va dentro de Inicio.
function pcPanel() {
  const i = S.info;
  const series = [
    { key: 'cpu', label: 'CPU', color: '#38bdf8' }, { key: 'gpu', label: 'GPU', color: '#a78bfa' },
    { key: 'ram', label: 'RAM', color: '#fbbf24' }, { key: 'disk', label: 'Disco', color: '#fb7185' },
  ];
  const chart = lineChart({ series, height: 250 });
  const legend = h('div', { class: 'legend' });
  const rangeBtns = h('div', { class: 'seg' });
  const detail = h('div', { class: 'detail-grid' });

  const drawChart = () => {
    const data = {};
    for (const k of Object.keys(S.hist)) data[k] = S.hist[k].slice(-S.range);
    chart.update(data, S.visible);
  };
  const drawLegend = () => legend.replaceChildren(...series.map((s) => h('button', {
    class: 'lg' + (S.visible[s.key] ? ' on' : ''), onclick: () => { S.visible[s.key] = !S.visible[s.key]; drawLegend(); drawChart(); },
  }, h('i', { style: `background:${s.color}` }), s.label,
    h('b', {}, S.hist[s.key].length ? `${S.hist[s.key].at(-1).toFixed(0)}%` : '–'))));
  const drawRange = () => rangeBtns.replaceChildren(...[[30, '1 min'], [150, '5 min']].map(([n, t]) => h('button', {
    class: S.range === n ? 'on' : '', onclick: () => { S.range = n; drawRange(); drawChart(); },
  }, t)));
  drawRange(); drawLegend();

  const row = (k, v) => h('div', { class: 'kv' }, h('span', {}, k), h('b', {}, v));
  const repaint = () => {
    const l = S.live; if (!l) return;
    const g = l.gpu;
    drawChart(); drawLegend();
    const ci = i.cpuInfo || {};
    detail.replaceChildren(
      h('div', { class: 'glass dcard spot' }, h('div', { class: 'dhead' }, icon('cpu'), h('b', {}, 'Procesador')), h('div', { class: 'dsub' }, i.cpu),
        row('Núcleos / hilos', `${ci.cores || '–'} / ${ci.threads || i.cores}`),
        row('Frecuencia actual', l.cpuMHz ? `${(l.cpuMHz / 1000).toFixed(2)} GHz` : '–'),
        row('Frecuencia base', ci.baseMHz ? `${(ci.baseMHz / 1000).toFixed(2)} GHz` : '–'),
        row('Caché L2', ci.l2KB ? `${(ci.l2KB / 1024).toFixed(ci.l2KB % 1024 ? 1 : 0)} MB` : '–'),
        row('Caché L3', ci.l3KB ? `${(ci.l3KB / 1024).toFixed(ci.l3KB % 1024 ? 1 : 0)} MB` : '–'),
        h('div', { class: 'dim', style: 'margin-top:8px' }, 'La temperatura de la CPU no se muestra: Windows no la expone sin un driver de hardware.')),
      h('div', { class: 'glass dcard spot' }, h('div', { class: 'dhead' }, icon('gpu'), h('b', {}, 'Tarjeta de video')), h('div', { class: 'dsub' }, g ? g.name : (i.gpus || [])[0] || 'No detectada'),
        g ? [row('Temperatura', g.temp >= 0 ? `${g.temp} °C` : '–'), row('Uso', `${g.util}%`),
          row('VRAM', `${(g.vramUsed / 1024).toFixed(1)} / ${(g.vramTotal / 1024).toFixed(0)} GB`),
          row('Reloj del núcleo', `${g.clock} / ${g.clockMax} MHz`), row('Consumo', g.power >= 0 ? `${g.power.toFixed(0)} W` : '–'),
          row('Ventilador', g.fan >= 0 ? `${g.fan}%` : '–'), row('Driver', g.driver)]
          : h('div', { class: 'dim' }, 'Los datos en vivo de la GPU están disponibles para tarjetas NVIDIA.')),
      h('div', { class: 'glass dcard spot' }, h('div', { class: 'dhead' }, icon('ram'), h('b', {}, 'Memoria')), h('div', { class: 'dsub' }, `${l.ramTotalGB.toFixed(1)} GB instalados`),
        row('En uso', `${l.ramUsedGB.toFixed(1)} GB (${Math.round(l.ram)}%)`), row('Disponible', `${(l.ramTotalGB - l.ramUsedGB).toFixed(1)} GB`)),
      h('div', { class: 'glass dcard spot' }, h('div', { class: 'dhead' }, icon('disk'), h('b', {}, 'Almacenamiento')), h('div', { class: 'dsub' }, `Actividad del disco: ${Math.round(l.diskBusy)}%`),
        (l.disks || []).map((d) => h('div', { class: 'drive' },
          h('div', { class: 'kv' }, h('span', {}, d.letter), h('b', {}, `${d.freeGB.toFixed(0)} GB libres de ${d.totalGB.toFixed(0)} GB`)),
          h('div', { class: 'meter' }, h('i', { style: `width:${d.usedPct}%;background:linear-gradient(90deg,${colorAt(zone(d.usedPct, ZONES.disk), 0)},${colorAt(zone(d.usedPct, ZONES.disk), 1)})` }))))));
  };
  return {
    repaint,
    nodes: [
      h('div', { class: 'section' }, h('h2', {}, 'Tu equipo en tiempo real')),
      h('div', { class: 'glass chartcard spot' }, h('div', { class: 'chead' }, h('b', {}, 'Uso en tiempo real'), rangeBtns), chart.el, legend),
      h('div', { style: 'height:14px' }),
      detail,
    ],
  };
}

// Mis juegos: una portada por cada juego que jugaste. Al pasar el mouse se ven los FPS de la última sesión.
const hue = (s) => { let x = 0; for (const c of s) x = (x * 31 + c.charCodeAt(0)) % 360; return x; };
const fpsNum = (v) => (v > 0 ? Math.round(v) : '–');

function gameCard(g, playing) {
  const cover = h('div', { class: 'cv-art', style: `--h:${hue(g.exe)}` }, h('span', { class: 'cv-init' }, (g.game || '?').slice(0, 1).toUpperCase()));
  GameCover(g.path || '').then((src) => {
    if (src) { const img = h('img', { alt: g.game, draggable: 'false', onload: () => img.classList.add('loaded') }); img.src = src; cover.replaceChildren(img); cover.classList.add('has-img'); return; }
    return AppIcons([g.path]).then((m) => { if (m && m[g.path]) cover.prepend(h('img', { class: 'cv-exe', src: m[g.path], alt: '' })); });
  }).catch(() => {});
  const stat2 = (label, v) => h('div', { class: 'cv-stat' }, h('span', {}, label), h('b', {}, fpsNum(v)));
  const over = g.measured
    ? h('div', { class: 'cv-over' },
      h('div', { class: 'cv-t' }, g.game),
      h('div', { class: 'cv-k' }, 'Media de FPS en la sesión'), h('div', { class: 'cv-big' }, fpsNum(g.avgFps)),
      fullAccess()
        ? h('div', { class: 'cv-row' }, stat2('0.1% low', g.low01Fps), stat2('1% low', g.low1Fps))
        : h('button', { class: 'cv-pro', onclick: () => upgradeDialog('0.1 % y 1 % low') }, icon('lock'), '0.1% y 1% low · Pro Tier'),
      h('div', { class: 'cv-sub' }, fmtWhen(g.sessionAt)))
    : h('div', { class: 'cv-over' },
      h('div', { class: 'cv-t' }, g.game),
      h('div', { class: 'cv-k' }, 'Media de FPS en la sesión'), h('div', { class: 'cv-big' }, '–'),
      h('div', { class: 'cv-sub' }, S.info && S.info.admin ? 'Esta sesión no tuvo datos de FPS (duró muy poco o el juego bloquea los eventos del sistema).' : 'Abrí South Tweaks como administrador para medir los FPS.'));
  return h('div', { class: 'cv' + (playing ? ' live' : '') }, cover, playing && h('span', { class: 'cv-live' }, 'Jugando'), h('div', { class: 'cv-name' }, g.game), over);
}

function pageGames() {
  const grid = h('div', { class: 'cover-grid' });
  const summary = h('div', { class: 'gsum' });
  const draw = (games) => {
    if (!games.length) {
      grid.replaceChildren(h('div', { class: 'glass empty' }, 'Todavía no hay juegos. Abrí uno (Steam, Epic, Riot, Xbox, GOG, Ubisoft, EA, Rockstar, Roblox) y su portada aparece acá al cerrarlo.'));
      return;
    }
    grid.replaceChildren(...games.map((g) => gameCard(g, S.game && S.game.exe && S.game.exe.toLowerCase() === g.exe.toLowerCase())));
    const measured = games.filter((g) => g.measured && g.avgFps > 0);
    const best = measured.slice().sort((a, b) => b.avgFps - a.avgFps)[0];
    const box = (ico, cat, value, label) => h('div', { class: 'glass spot' }, h('div', { class: 'head-ico', 'data-cat': cat }, icon(ico)), h('div', {}, h('b', {}, value), h('span', {}, label)));
    summary.replaceChildren(
      box('gamepad', 'Juegos', String(games.length), games.length === 1 ? 'juego detectado' : 'juegos detectados'),
      box('activity', 'Sistema', String(measured.length), 'con FPS medidos'),
      box('trend', 'GPU', best ? `${Math.round(best.avgFps)} FPS` : '–', best ? `mejor media · ${best.game}` : 'todavía sin mediciones'));
    reveal([...summary.children], { step: 60 });
    reveal([...grid.children], { base: 120, step: 45 });
  };
  const load = async () => {
    S.game = await CurrentGame();
    const lib = (await GameLibrary()) || [];
    if (S.game && !lib.some((g) => g.exe.toLowerCase() === S.game.exe.toLowerCase())) {
      lib.unshift({ game: S.game.game, exe: S.game.exe, path: S.game.path, measured: false });
    }
    if (el.content.contains(grid)) draw(lib);
  };
  grid.replaceChildren(h('div', { class: 'glass empty' }, h('div', { class: 'spin' }), ' Buscando tus juegos…'));
  load();
  return [pageHead('Mis juegos', 'Pasá el mouse por un juego para ver sus FPS de la última sesión.'), summary, grid];
}
function pageSettings() {
  el.search = h('input', { class: 'search', placeholder: 'Buscar ajustes…', value: S.q, oninput: (e) => { S.q = e.target.value; paintSettingsList(); } });
  el.chips = h('div', { class: 'chips' });
  el.list = h('div');
  el.simBox = h('input', { type: 'checkbox', checked: S.sim, onchange: async (e) => { await SetSimulation(e.target.checked); S.sim = e.target.checked; await refresh(); } });
  el.revertAll = h('button', { class: 'danger', onclick: revertAllFlow }, 'Revertir todo');
  el.simBanner = h('div', { class: 'banner info', hidden: true }, 'Modo simulación: no se modifica nada en tu Windows.');
  el.upgradeBanner = h('div', { class: 'banner upsell', hidden: true });
  el.scanBar = h('div', { class: 'scanbar' });
  setTimeout(() => { paintScan(); refresh(); }, 0);        // cada vez que se abre Optimizar se lee la PC de nuevo
  setTimeout(() => { paintChips(); paintSettingsList(); paintState(); }, 0);
  return [
    pageHead('Optimizar', 'Elegí lo que querés activar o desactivar y aplicalo todo junto desde la barra de abajo. Las acciones únicas no se pueden deshacer.',
      h('div', { class: 'head-actions' }, el.search,
        h('label', { class: 'chk pill', 'data-tip': 'Probá los cambios sin tocar tu Windows: la app hace como que los aplica.' }, el.simBox, 'Simulación'),
        el.revertAll)),
    el.simBanner, el.upgradeBanner, el.scanBar,
    h('div', { class: 'toolbar' }, el.chips),
    el.list,
  ];
}

const CLEAN_ICON = { temp_user: 'folder', temp_windows: 'folder', crash_dumps: 'alert', wer_reports: 'info', shader_cache: 'gpu', web_cache: 'globe' };
const CLEAN_CAT = { temp_user: 'Sistema', temp_windows: 'Sistema', crash_dumps: 'Acciones únicas', wer_reports: 'Privacidad', shader_cache: 'GPU', web_cache: 'Red' };

function pageClean() {
  const wrap = h('div', { class: 'tlist' });
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
        h('div', { class: 'cat-ico', 'data-cat': CLEAN_CAT[i.id] || 'Sistema' }, icon(CLEAN_ICON[i.id] || 'trash')),
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
  const wrap = h('div', { class: 'tlist' });
  if (!fullAccess()) {
    return [pageHead('Inicio de Windows', 'Desactivá programas que arrancan con Windows.'),
      h('div', { class: 'glass upsell-card' }, icon('lock', 'big'), h('h3', {}, 'Disponible en Pro Tier'),
        h('p', {}, 'Con Pro Tier elegís qué programas arrancan con Windows, con el icono de cada uno. El upgrade llega pronto.'),
        h('button', { class: 'primary', onclick: () => upgradeDialog('Inicio de Windows') }, 'Ver Pro Tier'))];
  }
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
      const ico = h('div', { class: 'app-ico', 'data-exe': it.exe || '' }, (it.name || '?').slice(0, 1).toUpperCase());
      it._ico = ico;
      return h('div', { class: 'tweak' + (blocked ? ' off' : ''), title: blocked ? 'Requiere administrador' : '' },
        ico,
        h('div', { class: 'info' }, h('div', { class: 't' }, it.name, h('span', { class: 'tag' }, it.label)), h('div', { class: 'd mono' }, it.command)),
        h('label', { class: 'switch' }, input, h('span')));
    }));
    const exes = [...new Set(items.map((x) => x.exe).filter(Boolean))];
    if (exes.length) AppIcons(exes).then((m) => {
      for (const it of items) if (it._ico && m && m[it.exe]) it._ico.replaceChildren(h('img', { src: m[it.exe], alt: '', draggable: 'false' }));
    }).catch(() => {});
  };
  render();
  ListStartup().then((l) => { S.startup = l || []; if (el.content.contains(wrap)) render(); });
  return [pageHead('Inicio de Windows', 'Desactivá programas que arrancan con Windows. No se borra nada: podés volver a activarlos.'), wrap];
}
// ---------- Fixes: arreglos de problemas comunes de Windows ----------
const FIX_BRAND = { valorant: 'valorant', nvidia: 'nvidia' };

async function fixFlow(f, btn) {
  if (S.busy) return;
  btn.disabled = true; const label = btn.textContent; btn.textContent = 'Leyendo tu PC…';
  let chk;
  try { chk = await FixCheck(f.id); } catch (e) { chk = { error: String(e) }; }
  btn.disabled = false; btn.textContent = label;
  if (chk.error) { toast(chk.error, 'err'); return; }
  if (!chk.problems.length && !chk.always) {
    await dialog(f.title, [h('p', {}, 'No se encontró el problema en tu PC.'), h('p', { class: 'dim' }, chk.healthy || ''), h('p', { class: 'dim' }, 'No se cambió nada.')], { ok: 'Entendido', cancel: null });
    return;
  }
  const body = [
    h('p', {}, f.description),
    chk.problems.length > 0 && h('p', { class: 'k', style: 'margin:10px 0 4px' }, 'Lo que encontré en tu PC:'),
    chk.problems.length > 0 && h('ul', { class: 'mini' }, chk.problems.map((p) => h('li', {}, p))),
    f.reboot && h('p', { class: 'warn' }, 'Necesita reiniciar Windows para tener efecto.'),
    !f.admin ? null : (S.info.admin ? null : h('p', { class: 'warn' }, 'Necesita permisos de administrador.')),
  ];
  const r = await dialog(f.title, body, { ok: chk.always ? 'Ejecutar' : 'Arreglar' });
  if (!r.ok) return;
  S.busy = true; paintState(); btn.disabled = true; btn.textContent = 'Arreglando…';
  let res;
  try { res = await RunFix(f.id); } catch (e) { res = { error: String(e), lines: [] }; }
  S.busy = false; paintState();
  await dialog(res.error ? 'No se pudo arreglar todo' : 'Listo', [
    h('p', { class: res.error ? 'warn' : '' }, res.error || res.message),
    res.lines && res.lines.length > 0 && h('ul', { class: 'mini fixlines' }, res.lines.map((l) => h('li', { class: l.startsWith('✗') ? 'bad' : (l.startsWith('✓') ? 'good' : '') }, l))),
    res.reboot && !res.error && h('p', { class: 'warn' }, 'Reiniciá Windows para que el cambio se aplique.'),
  ], { ok: 'Cerrar', cancel: null });
  pageFixesRefresh();
}

let pageFixesRefresh = () => {};

function pageFixes() {
  const grid = h('div', { class: 'fixgrid' });
  const draw = (list) => {
    grid.replaceChildren(...list.map((f) => {
      const run = h('button', { class: 'fix-run', 'data-needbusy': 1 }, 'Ejecutar arreglo');
      run.onclick = () => fixFlow(f, run);
      const undo = f.canUndo ? h('button', { class: 'fix-undo', 'data-needbusy': 1, onclick: async () => {
        if (S.busy) return;
        S.busy = true; paintState();
        const o = await RevertTweak(f.id);
        S.busy = false; paintState();
        if (o.error) toast(o.error, 'err'); else toast(o.message, 'ok');
        pageFixesRefresh();
      } }, 'Deshacer') : null;
      return h('div', { class: 'fixcard spot' },
        h('div', { class: 'fix-head' }, h('div', { class: 'fix-ico', 'data-brand': FIX_BRAND[f.icon] || '' }, icon(f.icon === 'nvidia' ? 'gpu' : f.icon)), h('div', { class: 'fix-title' }, f.title)),
        h('div', { class: 'tc-rule' }),
        h('p', { class: 'fix-desc' }, f.description),
        h('div', { class: 'fix-foot' },
          h('div', { class: 'tc-tags' }, f.admin && h('span', { class: 'tag adm' }, 'Admin'), f.reboot && h('span', { class: 'tag' }, 'Reinicio'), f.lastRun && h('span', { class: 'tc-last' }, `Última vez: ${fmtWhen(f.lastRun)}`)),
          h('div', { class: 'row' }, undo, run)));
    }));
  };
  const load = async () => { const l = (await ListFixes()) || []; if (el.content.contains(grid)) draw(l); };
  pageFixesRefresh = load;
  grid.replaceChildren(h('div', { class: 'glass empty' }, h('div', { class: 'spin' }), ' Cargando arreglos…'));
  load();
  return [pageHead('Fixes', 'Arreglos rápidos para problemas comunes de Windows. Cada uno revisa tu PC primero y solo cambia lo que está mal.'), grid];
}

const PAGES = [
  { id: 'home', label: 'Inicio', icon: 'home', render: pageHome },
  { id: 'games', label: 'Mis juegos', icon: 'gamepad', render: pageGames },
  { id: 'settings', label: 'Optimizar', icon: 'sliders', render: pageSettings },
  { id: 'fixes', label: 'Fixes', icon: 'wrench', render: pageFixes },
  { id: 'clean', label: 'Limpieza', icon: 'trash', render: pageClean },
  { id: 'startup', label: 'Inicio de Windows', icon: 'rocket', render: pageStartup },
];

// Menú lateral: se arma una vez; al cambiar de página solo se mueve el indicador (se desliza con la curva
// de entrada) y cambia la clase del botón activo.
function buildNav() {
  el.navInd = h('div', { class: 'nav-ind', 'aria-hidden': 'true' });
  el.navBtns = new Map();
  const btns = PAGES.map((p) => {
    const b = h('button', { class: 'nav', onclick: () => { if (S.page !== p.id || !el.content.firstChild) go(p.id); } }, icon(p.icon), p.label,
      p.id === 'fixes' ? h('span', { class: 'new' }, 'Nuevo') : null);
    el.navBtns.set(p.id, b);
    return b;
  });
  el.nav.replaceChildren(h('div', { class: 'nav-label' }, 'Menú'), h('div', { class: 'nav-wrap' }, el.navInd, btns));
}

function moveNav(id) {
  const b = el.navBtns && el.navBtns.get(id);
  if (!b) return;
  for (const [k, v] of el.navBtns) { v.classList.toggle('on', k === id); v.setAttribute('aria-current', k === id ? 'page' : 'false'); }
  el.navInd.style.transform = `translate3d(0, ${b.offsetTop}px, 0)`;
}

function go(id) {
  S.page = id;
  if (id === 'startup') S.startup = null;     // siempre se relee al entrar
  if (id === 'clean') S.clean = null;
  paintLiveFns = [];
  moveNav(id);
  const page = PAGES.find((p) => p.id === id);
  const node = h('div', { class: 'page' }, page.render());
  document.title = `${PAGE_NAME[id] || ''} · South Tweaks`;
  swapPage(el.content, node, (n) => {
    el.main.scrollTop = 0;
    revealPage(n);
    paintState();
  });
}

// ---------- sesión, niveles y Discord ----------
const TIER_CLASS = { 'Propietario': 'owner', 'Pro Tier': 'pro', 'Free Tier': 'free' };
const effective = () => (S.discord && S.discord.effective) || '';
const fullAccess = () => effective() === 'Pro Tier' || effective() === 'Propietario';

function upgradeDialog(what) {
  return dialog('Pro Tier', [
    h('p', {}, `«${what}» es parte de Pro Tier.`),
    h('p', { class: 'dim' }, 'Pro Tier desbloquea todos los ajustes, las acciones únicas, el control del inicio de Windows y los FPS 0.1% / 1% low. Muy pronto vas a poder hacer el upgrade desde acá.')], { ok: 'Entendido', cancel: null });
}

// Pantalla de inicio de sesión: tapa toda la app hasta que haya sesión.
let gate = null;
function showGate(errMsg) {
  if (gate) gate.remove();
  const btn = h('button', { class: 'discord-btn', onclick: () => discordLogin() }, 'Iniciar sesión con Discord');
  const note = h('p', { class: 'gate-err' }, errMsg || '');
  gate = h('div', { class: 'gate' },
    h('div', { class: 'gate-card' },
      h('div', { class: 'logo big' }, 'ST'),
      h('h1', {}, 'South Tweaks'),
      h('p', { class: 'gate-sub' }, 'Iniciá sesión con tu cuenta de Discord para entrar. Solo leemos tu nombre y tu foto: no se guarda ninguna contraseña ni token.'),
      btn, note,
      h('p', { class: 'dim', style: 'margin-top:16px' }, 'Se abre tu navegador para autorizar el acceso.')));
  gate.btn = btn; gate.note = note;
  document.body.append(gate);
}
function hideGate() { if (gate) { gate.remove(); gate = null; } }

async function onSession(u) {
  S.discord = u;
  paintUser();
  if (u.loggedIn) { hideGate(); await refresh(); go(S.page === 'home' || !S.page ? 'home' : S.page); }
  else showGate();
}

async function discordLogin() {
  if (S.discordBusy) return;
  S.discordBusy = true;
  if (gate) { gate.btn.disabled = true; gate.btn.textContent = 'Esperando a Discord…'; gate.note.textContent = ''; }
  paintUser();
  let res;
  try { res = await DiscordLogin(); } catch (e) { res = { error: String(e) }; }
  S.discordBusy = false;
  if (res.error) {
    if (gate) { gate.btn.disabled = false; gate.btn.textContent = 'Iniciar sesión con Discord'; gate.note.textContent = res.error; }
    else toast(res.error, 'err');
    paintUser();
    return;
  }
  await onSession(res);
  toast(`Hola, ${res.name} · ${res.tier}`, 'ok');
}

function closeUserMenu() { document.querySelectorAll('.umenu').forEach((m) => m.remove()); }

async function setView(tier) {
  closeUserMenu();
  const u = await SetViewAs(tier);
  S.discord = u;
  Q.clear();
  paintUser(); await refresh();
  go(S.page);
  toast(`Vista: ${tier}`, 'ok');
}

function paintUser() {
  if (!el.user) return;
  const d = S.discord || {};
  const logged = !!d.loggedIn;
  const avatar = h('div', { class: 'avatar' + (logged && d.avatar ? ' img' : '') }, logged && d.avatar ? h('img', { src: d.avatar, alt: '', draggable: 'false' }) : icon('user'));
  const viewing = logged && d.owner && d.effective && d.effective !== d.tier;
  const sub = S.discordBusy ? 'Autorizá en el navegador' : (logged ? d.tier + (viewing ? ' · vista ' + d.effective : '') : 'con Discord');
  el.user.replaceChildren(avatar,
    h('div', { class: 'u-txt' }, h('b', {}, S.discordBusy ? 'Esperando a Discord…' : (logged ? d.name : 'Sin sesión')), h('small', { class: logged ? 'tier ' + (TIER_CLASS[d.tier] || 'free') : '' }, sub)),
    ...(logged ? [icon('chevron', 'u-chev')] : []));
  el.user.onclick = (e) => {
    e.stopPropagation();
    if (!logged || S.discordBusy) return;
    if (document.querySelector('.umenu')) { closeUserMenu(); return; }
    const opt = (tier, label) => h('button', { class: d.effective === tier ? 'cur' : '', onclick: () => setView(tier) }, d.effective === tier ? '✓ ' + label : label);
    const menu = h('div', { class: 'umenu' },
      d.owner && h('div', { class: 'um-h' }, 'Ver la app como'),
      d.owner && opt('Propietario', 'Propietario (todo)'),
      d.owner && opt('Pro Tier', 'Pro Tier'),
      d.owner && opt('Free Tier', 'Free Tier'),
      d.owner && h('div', { class: 'um-sep' }),
      h('button', { onclick: async () => { closeUserMenu(); await DiscordLogout(); Q.clear(); await onSession({ loggedIn: false }); } }, 'Cerrar sesión'));
    el.user.append(menu);
    setTimeout(() => document.addEventListener('click', closeUserMenu, { once: true }), 0);
  };
}
// ---------- layout ----------
function build() {
  el.nav = h('nav');
  el.user = h('div', { class: 'user' });
  el.content = h('div', { class: 'content' });
  el.busy = h('div', { class: 'busy-bar', hidden: true });
  el.adminBanner = h('div', { class: 'banner', hidden: true }, 'Algunos ajustes necesitan permisos de administrador.',
    h('button', { onclick: async () => { const e = await RelaunchAsAdmin(); if (e) toast(e, 'err'); } }, 'Reiniciar como administrador'));
  el.queueCount = h('b', { class: 'q-n' }, '0');
  el.queueLabel = h('span', { class: 'q-l' }, 'procesos por aplicar');
  el.queueParts = h('span', { class: 'q-p k' });
  el.queueClear = h('button', { onclick: () => { for (const id of [...Q.keys()]) { Q.delete(id); } for (const r of S.rows.values()) paintRow(r, r.t); paintQueue(); } }, 'Vaciar');
  el.queueApply = h('button', { class: 'primary', onclick: applyQueue }, 'Aplicar');
  el.queue = h('div', { class: 'queue', hidden: true },
    h('div', { class: 'q-info' }, el.queueCount, h('div', {}, el.queueLabel, el.queueParts)),
    h('div', { class: 'row' }, el.queueClear, el.queueApply));
  el.main = h('main', {}, el.adminBanner, el.content);
  document.getElementById('app').replaceChildren(
    h('aside', {},
      h('div', { class: 'brand' }, h('div', { class: 'logo' }, 'ST'), h('div', {}, h('b', {}, 'South Tweaks'), h('small', {}, 'Optimizador para juegos'))),
      el.nav,
      el.user),
    el.main);
  buildNav();
  document.body.prepend(h('div', { class: 'bg', 'aria-hidden': 'true' }, h('div', { class: 'orb o1' }), h('div', { class: 'orb o2' }), h('div', { class: 'orb o3' })));
  document.body.append(el.busy, el.queue, toasts);
  initReveals(el.main);
  initScrollParallax(el.main);
  initPointerFX();
  initTooltips();
  window.addEventListener('resize', () => moveNav(S.page));
}

// ---------- muestras en vivo ----------
function pushHist(l) {
  const d = systemDisk(l);
  const add = (k, v) => { S.hist[k].push(v); if (S.hist[k].length > HIST_MAX) S.hist[k].shift(); };
  add('cpu', l.cpu); add('ram', l.ram); add('gpu', l.gpu ? l.gpu.util : 0); add('disk', l.diskBusy || 0);
  void d;
}

function ambient(l) {
  const g = l.gpu;
  const st = worst(zone(l.cpu, ZONES.cpu), g && g.temp >= 0 ? zone(g.temp, ZONES.temp) : 'cool', zone(l.ram, ZONES.ram));
  document.documentElement.style.setProperty('--glow', GLOW[st]);
}

let ticking = false;
async function tick() {
  if (ticking) return;
  ticking = true;
  try {
    const l = await GetLive();
    S.live = l; pushHist(l); ambient(l);
    for (const f of paintLiveFns) f();
  } catch { /* ventana cerrándose */ } finally { ticking = false; }
}

(async () => {
  const [initErr, info, sim] = await Promise.all([InitError(), GetSystemInfo(), GetSimulation()]);
  S.info = info; S.sim = sim;
  if (initErr) {
    document.getElementById('app').replaceChildren(h('div', { class: 'banner err', style: 'margin:30px;grid-column:1/-1' }, 'No se pudo iniciar de forma segura: ' + initErr));
    return;
  }
  build();
  paintUser();
  S.live = await GetLive();
  S.discord = await DiscordState().catch(() => ({ loggedIn: false }));
  paintUser();
  if (S.discord.loggedIn) { await refresh(); go('home'); } else showGate();
  setInterval(tick, 2000);
})();