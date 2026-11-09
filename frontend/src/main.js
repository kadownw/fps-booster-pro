import './style.css';
import {
  GetSystemInfo, GetLive, ListTweaks, ApplyTweak, RevertTweak, RevertAll,
  SetSimulation, GetSimulation, RelaunchAsAdmin, InitError,
  CreateRestorePoint,
  ScanCleanup, RunCleanup, ListStartup, SetStartup,
  CurrentGame, RunOnce, GameLibrary, GameCover, AppIcons,
  DiscordState, DiscordLogin, DiscordLogout, DiscordSetClientID,
} from '../wailsjs/go/main/App';
import { h, svg, icon, fmtBytes, fmtDuration, zone, worst, GLOW, colorAt, CAT_ICON } from './lib.js';
import { tickGauge } from './gauge.js';
import { lineChart, barChart } from './chart.js';

// ---------- estado ----------
const HIST_MAX = 150;                       // 150 muestras x 2 s = 5 minutos
const S = {
  info: null, tweaks: [], cat: 'Todos', q: '', sim: false, busy: false, page: 'home',
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
          else err = await (x.a === 'apply' ? ApplyTweak : RevertTweak)(x.t.id);
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
  const row = h('div', { class: 'tcard', style: `animation-delay:${Math.min(S.rows.size, 12) * 20}ms` },
    h('div', { class: 'tc-head' }, catIco(t.category), h('div', { class: 'tc-title' }, t.title),
      brand && h('span', { class: 'vbadge ' + brand, title: `Exclusivo de ${BRAND[brand]}` }, BRAND[brand])),
    h('div', { class: 'tc-rule' }),
    h('p', { class: 'tc-desc' }, t.description),
    h('div', { class: 'tc-foot' }, h('div', { class: 'tc-tags' }, tags, last), control));
  const r = { row, input, btn, tags, last, t };
  S.rows.set(t.id, r);
  return r;
}

function paintRow(r, t) {
  r.t = t;
  r.row.classList.toggle('on', t.active);
  r.row.classList.toggle('off', t.disabled);
  const blocked = t.disabled || (t.blocked && !t.active);
  const q = Q.get(t.id);
  r.row.classList.toggle('queued', !!q);
  if (r.input) { r.input.checked = q ? q === 'apply' : t.active; r.input.disabled = S.busy || blocked; }
  if (r.btn) {
    r.btn.disabled = S.busy || t.disabled || t.blocked;
    r.btn.textContent = q === 'once' ? 'En la cola ✓' : 'Agregar';
    r.btn.classList.toggle('primary', q === 'once');
  }
  r.row.title = t.disabled ? 'No aplica a tu equipo' : (t.blocked ? 'Requiere administrador' : '');
  r.last.textContent = t.once && t.lastRun ? `Última vez: ${fmtWhen(t.lastRun)}` : '';
  r.tags.replaceChildren(...[
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
  for (const c of cats) {
    const items = S.tweaks.filter((x) => x.category === c && visible(x));
    frag.append(h('div', { class: 'section' }, catIco(c), h('h2', {}, c), h('span', { class: 'count' }, items.length)));
    frag.append(h('div', { class: 'tgrid' }, items.map((t) => S.rows.get(t.id).row)));
  }
  if (!cats.length) frag.append(h('p', { class: 'dim', style: 'padding:40px 0;text-align:center' }, 'Nada coincide con tu búsqueda.'));
  el.list.replaceChildren(frag);
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

async function refresh() {
  const list = (await ListTweaks()) || [];
  const newIds = list.map((t) => t.id).join();
  const rebuild = newIds !== S.tweaks.map((t) => t.id).join();
  S.tweaks = list;
  if (rebuild) { S.rows.clear(); for (const t of list) buildRow(t); }
  pruneQueue();
  paintState();
  paintQueue();
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
function hwCard({ ico, title, sub, to, gauge, stats }) {
  return h('div', { class: 'glass hw' },
    h('div', { class: 'hw-l' },
      h('div', { class: 'hw-ico' }, icon(ico)),
      h('div', { class: 'hw-t' }, title),
      h('div', { class: 'hw-s', title: sub }, sub),
      h('div', { class: 'hw-stats' }, stats)),
    h('div', { class: 'hw-g' }, gauge.el));
}

const stat = (k, v) => h('div', {}, h('span', {}, k), h('b', {}, v));

// ---------- páginas ----------
function pageHead(title, sub, right) {
  return h('div', { class: 'top' }, h('div', {}, h('h1', {}, title), h('p', {}, sub)), right);
}

function pageHome() {
  const i = S.info;
  const gCPU = tickGauge({ size: 150 }), gGPU = tickGauge({ size: 150 }), gRAM = tickGauge({ size: 150 }), gDSK = tickGauge({ size: 150 });
  const cpuStats = h('div', { class: 'stats-mini' }), gpuStats = h('div', { class: 'stats-mini' });
  const ramStats = h('div', { class: 'stats-mini' }), dskStats = h('div', { class: 'stats-mini' });
  const hFill = h('div', { class: 'hbar-fill' }), hVal = h('div', { class: 'hbar-val' }), hChip = h('div', { class: 'hchip' });
  const hIssues = h('ul', { class: 'hissues' });
  el.activeCount = h('b', {}, '–');

  const cards = [
    hwCard({ ico: 'cpu', title: 'Procesador', sub: i.cpu || 'Desconocido', to: 'pc', gauge: gCPU, stats: cpuStats }),
    hwCard({ ico: 'gpu', title: 'Tarjeta de video', sub: (i.gpus || [])[0] || 'No detectada', to: 'pc', gauge: gGPU, stats: gpuStats }),
    hwCard({ ico: 'ram', title: 'Memoria RAM', sub: `${i.ramGB.toFixed(1)} GB instalados`, to: 'pc', gauge: gRAM, stats: ramStats }),
    hwCard({ ico: 'disk', title: 'Almacenamiento', sub: 'Unidad del sistema', to: 'pc', gauge: gDSK, stats: dskStats }),
  ];

  const repaint = () => {
    const l = S.live; if (!l) return;
    const g = l.gpu;
    gCPU.set({ frac: l.cpu / 100, state: zone(l.cpu, ZONES.cpu), text: Math.round(l.cpu), unitText: '%', labelText: 'En uso' });
    cpuStats.replaceChildren(stat('Frecuencia', l.cpuMHz ? `${(l.cpuMHz / 1000).toFixed(2)} GHz` : '–'), stat('Núcleos', `${i.cpuInfo?.cores || '–'} / ${i.cpuInfo?.threads || i.cores} hilos`));
    if (g && g.temp >= 0) {
      gGPU.set({ frac: g.temp / 100, state: zone(g.temp, ZONES.temp), text: g.temp, unitText: '°C', labelText: 'Temperatura' });
      gpuStats.replaceChildren(stat('Uso', `${g.util}%`), stat('VRAM', `${(g.vramUsed / 1024).toFixed(1)} / ${(g.vramTotal / 1024).toFixed(0)} GB`));
    } else {
      gGPU.set({ frac: null, text: '–', labelText: 'Sin datos' });
      gpuStats.replaceChildren(h('div', { class: 'dim' }, 'Temperatura y uso disponibles solo en NVIDIA.'));
    }
    gRAM.set({ frac: l.ram / 100, state: zone(l.ram, ZONES.ram), text: Math.round(l.ram), unitText: '%', labelText: 'En uso' });
    ramStats.replaceChildren(stat('Usada', `${l.ramUsedGB.toFixed(1)} GB`), stat('Libre', `${(l.ramTotalGB - l.ramUsedGB).toFixed(1)} GB`));
    const d = systemDisk(l);
    if (d) {
      gDSK.set({ frac: d.usedPct / 100, state: zone(d.usedPct, ZONES.disk), text: Math.round(d.usedPct), unitText: '%', labelText: `Usado · ${d.letter}` });
      dskStats.replaceChildren(stat('Libre', `${d.freeGB.toFixed(0)} GB`), stat('Actividad', `${Math.round(l.diskBusy)}%`));
    }
    const hl = health(l), st = healthState(hl.score);
    hFill.style.height = hl.score + '%';
    hFill.style.background = `linear-gradient(0deg, ${colorAt(st, 0)}, ${colorAt(st, 1)})`;
    hVal.textContent = hl.score + '%';
    hChip.textContent = healthText(hl.score);
    hChip.className = 'hchip ' + st;
    hIssues.replaceChildren(...(hl.issues.length ? hl.issues.map((x) => h('li', {}, x)) : [h('li', { class: 'ok' }, 'Todo en orden')]));
  };
  const panel = pcPanel();
  paintLiveFns = [repaint, panel.repaint];
  setTimeout(() => { repaint(); panel.repaint(); }, 0);

  const tile = (ico, title, sub, onclick) => h('button', { class: 'tile', onclick },
    h('div', { class: 'tile-ico' }, icon(ico)), h('div', { class: 'tile-t' }, h('b', {}, title), h('span', {}, sub)), icon('chevron', 'tile-go'));

  return [
    h('div', { class: 'top' },
      h('div', {}, h('h1', {}, `Bienvenido${S.discord && S.discord.loggedIn ? ', ' + S.discord.name : (i.user ? ', ' + i.user : '')}`), h('p', {}, '¿Cómo anda tu PC hoy?'))),
    h('div', { class: 'dash' },
      h('div', { class: 'glass health' },
        h('div', { class: 'k' }, 'Salud'), hChip,
        h('div', { class: 'hbar' }, hFill), hVal, hIssues),
      h('div', { class: 'hwgrid' }, cards)),
    h('div', { class: 'tiles' },
      tile('sliders', 'Optimizar', 'Elegí y aplicá ajustes', () => go('settings')),
      tile('trash', 'Limpiar archivos', 'Liberá espacio', () => go('clean')),
      tile('gamepad', 'Mis juegos', 'Detector y sesiones', () => go('games')),
      tile('rocket', 'Programas de inicio', 'Arranque más rápido', () => go('startup')),
      tile('restore', 'Punto de restauración', 'Antes de cambiar nada', makeRestorePoint)),
    ...panel.nodes,
    h('div', { class: 'section' }, h('h2', {}, 'Seguridad')),
    h('div', { class: 'glass safety' },
      h('div', {}, h('b', {}, 'Todo es reversible'),
        h('p', { class: 'd' }, 'Cada ajuste guarda su valor original antes de aplicarse. Además podés crear un punto de restauración de Windows antes de empezar.'),
        h('p', { class: 'd' }, 'Ajustes activos: ', el.activeCount)),
      h('div', { class: 'row' }, h('button', { class: 'danger', 'data-needbusy': 1, onclick: revertAllFlow }, 'Revertir todo'))),
  ];
}

// Detalle en tiempo real (gráfico + tarjetas). Va dentro de Inicio.
function pcPanel() {
  const i = S.info;
  const series = [
    { key: 'cpu', label: 'CPU', color: '#2fe3a2' }, { key: 'gpu', label: 'GPU', color: '#6c8cff' },
    { key: 'ram', label: 'RAM', color: '#f6c945' }, { key: 'disk', label: 'Disco', color: '#ff7a8a' },
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
      h('div', { class: 'glass dcard' }, h('div', { class: 'dhead' }, icon('cpu'), h('b', {}, 'Procesador')), h('div', { class: 'dsub' }, i.cpu),
        row('Núcleos / hilos', `${ci.cores || '–'} / ${ci.threads || i.cores}`),
        row('Frecuencia actual', l.cpuMHz ? `${(l.cpuMHz / 1000).toFixed(2)} GHz` : '–'),
        row('Frecuencia base', ci.baseMHz ? `${(ci.baseMHz / 1000).toFixed(2)} GHz` : '–'),
        row('Caché L2', ci.l2KB ? `${(ci.l2KB / 1024).toFixed(ci.l2KB % 1024 ? 1 : 0)} MB` : '–'),
        row('Caché L3', ci.l3KB ? `${(ci.l3KB / 1024).toFixed(ci.l3KB % 1024 ? 1 : 0)} MB` : '–'),
        h('div', { class: 'dim', style: 'margin-top:8px' }, 'La temperatura de la CPU no se muestra: Windows no la expone sin un driver de hardware.')),
      h('div', { class: 'glass dcard' }, h('div', { class: 'dhead' }, icon('gpu'), h('b', {}, 'Tarjeta de video')), h('div', { class: 'dsub' }, g ? g.name : (i.gpus || [])[0] || 'No detectada'),
        g ? [row('Temperatura', g.temp >= 0 ? `${g.temp} °C` : '–'), row('Uso', `${g.util}%`),
          row('VRAM', `${(g.vramUsed / 1024).toFixed(1)} / ${(g.vramTotal / 1024).toFixed(0)} GB`),
          row('Reloj del núcleo', `${g.clock} / ${g.clockMax} MHz`), row('Consumo', g.power >= 0 ? `${g.power.toFixed(0)} W` : '–'),
          row('Ventilador', g.fan >= 0 ? `${g.fan}%` : '–'), row('Driver', g.driver)]
          : h('div', { class: 'dim' }, 'Los datos en vivo de la GPU están disponibles para tarjetas NVIDIA.')),
      h('div', { class: 'glass dcard' }, h('div', { class: 'dhead' }, icon('ram'), h('b', {}, 'Memoria')), h('div', { class: 'dsub' }, `${l.ramTotalGB.toFixed(1)} GB instalados`),
        row('En uso', `${l.ramUsedGB.toFixed(1)} GB (${Math.round(l.ram)}%)`), row('Disponible', `${(l.ramTotalGB - l.ramUsedGB).toFixed(1)} GB`)),
      h('div', { class: 'glass dcard' }, h('div', { class: 'dhead' }, icon('disk'), h('b', {}, 'Almacenamiento')), h('div', { class: 'dsub' }, `Actividad del disco: ${Math.round(l.diskBusy)}%`),
        (l.disks || []).map((d) => h('div', { class: 'drive' },
          h('div', { class: 'kv' }, h('span', {}, d.letter), h('b', {}, `${d.freeGB.toFixed(0)} GB libres de ${d.totalGB.toFixed(0)} GB`)),
          h('div', { class: 'meter' }, h('i', { style: `width:${d.usedPct}%;background:linear-gradient(90deg,${colorAt(zone(d.usedPct, ZONES.disk), 0)},${colorAt(zone(d.usedPct, ZONES.disk), 1)})` }))))));
  };
  return {
    repaint,
    nodes: [
      h('div', { class: 'section' }, h('h2', {}, 'Tu equipo en tiempo real')),
      h('div', { class: 'glass chartcard' }, h('div', { class: 'chead' }, h('b', {}, 'Uso en tiempo real'), rangeBtns), chart.el, legend),
      h('div', { style: 'height:12px' }),
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
    if (src) { cover.replaceChildren(h('img', { src, alt: g.game, draggable: 'false' })); cover.classList.add('has-img'); return; }
    return AppIcons([g.path]).then((m) => { if (m && m[g.path]) cover.prepend(h('img', { class: 'cv-exe', src: m[g.path], alt: '' })); });
  }).catch(() => {});
  const stat2 = (label, v) => h('div', { class: 'cv-stat' }, h('span', {}, label), h('b', {}, fpsNum(v)));
  const over = g.measured
    ? h('div', { class: 'cv-over' },
      h('div', { class: 'cv-t' }, g.game),
      h('div', { class: 'cv-k' }, 'Media de FPS en la sesión'), h('div', { class: 'cv-big' }, fpsNum(g.avgFps)),
      h('div', { class: 'cv-row' }, stat2('0.1% low', g.low01Fps), stat2('1% low', g.low1Fps)),
      h('div', { class: 'cv-sub' }, fmtWhen(g.sessionAt)))
    : h('div', { class: 'cv-over' },
      h('div', { class: 'cv-t' }, g.game),
      h('div', { class: 'cv-k' }, 'Media de FPS en la sesión'), h('div', { class: 'cv-big' }, '–'),
      h('div', { class: 'cv-sub' }, S.info && S.info.admin ? 'Esta sesión no tuvo datos de FPS (duró muy poco o el juego bloquea los eventos del sistema).' : 'Abrí South Tweaks como administrador para medir los FPS.'));
  return h('div', { class: 'cv' + (playing ? ' live' : '') }, cover, playing && h('span', { class: 'cv-live' }, 'Jugando'), h('div', { class: 'cv-name' }, g.game), over);
}

function pageGames() {
  const grid = h('div', { class: 'cover-grid' });
  const draw = (games) => {
    if (!games.length) {
      grid.replaceChildren(h('div', { class: 'glass empty' }, 'Todavía no hay juegos. Abrí uno (Steam, Epic, Riot, Xbox, GOG, Ubisoft, EA, Rockstar, Roblox) y su portada aparece acá al cerrarlo.'));
      return;
    }
    grid.replaceChildren(...games.map((g) => gameCard(g, S.game && S.game.exe && S.game.exe.toLowerCase() === g.exe.toLowerCase())));
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
  return [pageHead('Mis juegos', 'Pasá el mouse por un juego para ver sus FPS de la última sesión.'), grid];
}
function pageSettings() {
  el.search = h('input', { class: 'search', placeholder: 'Buscar ajustes…', value: S.q, oninput: (e) => { S.q = e.target.value; paintSettingsList(); } });
  el.chips = h('div', { class: 'chips' });
  el.list = h('div');
  el.simBox = h('input', { type: 'checkbox', checked: S.sim, onchange: async (e) => { await SetSimulation(e.target.checked); S.sim = e.target.checked; await refresh(); } });
  el.revertAll = h('button', { class: 'danger', onclick: revertAllFlow }, 'Revertir todo');
  el.simBanner = h('div', { class: 'banner info', hidden: true }, 'Modo simulación: no se modifica nada en tu Windows.');
  setTimeout(() => { paintChips(); paintSettingsList(); paintState(); }, 0);
  return [
    pageHead('Optimizar', 'Elegí lo que querés activar o desactivar y aplicalo todo junto desde la barra de abajo. Las acciones únicas no se pueden deshacer.', el.search),
    el.simBanner,
    h('div', { class: 'toolbar' }, el.chips,
      h('div', { class: 'row' }, h('label', { class: 'chk' }, el.simBox, 'Modo simulación'), el.revertAll)),
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
const PAGES = [
  { id: 'home', label: 'Inicio', icon: 'home', render: pageHome },
  { id: 'games', label: 'Mis juegos', icon: 'gamepad', render: pageGames },
  { id: 'settings', label: 'Optimizar', icon: 'sliders', render: pageSettings },
  { id: 'clean', label: 'Limpieza', icon: 'trash', render: pageClean },
  { id: 'startup', label: 'Inicio de Windows', icon: 'rocket', render: pageStartup },
];

function go(id) {
  S.page = id;
  if (id === 'startup') S.startup = null;     // siempre se relee al entrar
  if (id === 'clean') S.clean = null;
  paintLiveFns = [];
  el.nav.replaceChildren(...PAGES.map((p) => h('button', { class: 'nav' + (p.id === id ? ' on' : ''), onclick: () => go(p.id) }, icon(p.icon), p.label)));
  const page = PAGES.find((p) => p.id === id);
  el.content.replaceChildren(h('div', { class: 'page' }, page.render()));
  el.main.scrollTo({ top: 0 });
  paintState();
}

// ---------- usuario / Discord ----------
const TIER_CLASS = { 'Propietario': 'owner', 'Pro Tier': 'pro', 'Free Tier': 'free' };
const REDIRECT = 'http://127.0.0.1:53682/callback';

async function configureDiscord() {
  const input = h('input', { class: 'search', style: 'width:100%;margin-top:10px', placeholder: 'Application ID (solo números)', value: '' });
  const r = await dialog('Conectar con Discord', [
    h('p', {}, 'Para iniciar sesión con Discord necesitás una aplicación propia (gratis):'),
    h('ol', { class: 'mini steps' },
      h('li', {}, 'Entrá a discord.com/developers/applications y creá una aplicación.'),
      h('li', {}, 'En «OAuth2 → Redirects» agregá exactamente:', h('code', { class: 'copy' }, REDIRECT)),
      h('li', {}, 'Copiá el «Application ID» de «General Information» y pegalo acá.')),
    input,
    h('p', { class: 'dim', style: 'margin-top:8px' }, 'Solo se pide el permiso «identify» (nombre, id y foto). No se guarda ningún token.')], { ok: 'Guardar y continuar' });
  if (!r.ok) return false;
  const err = await DiscordSetClientID(input.value);
  if (err) { toast(err, 'err'); return false; }
  S.discord = { ...(S.discord || {}), configured: true };
  return true;
}

async function discordLogin() {
  if (!(S.discord && S.discord.configured) && !(await configureDiscord())) return;
  toast('Se abrió tu navegador: autorizá el acceso en Discord.');
  S.discordBusy = true; paintUser();
  try {
    const r = await DiscordLogin();
    if (r.error) toast(r.error, 'err'); else { S.discord = r; toast(`Hola, ${r.name} · ${r.tier}`, 'ok'); }
  } catch (e) { toast(String(e), 'err'); }
  S.discordBusy = false; paintUser();
  if (S.page === 'home') go('home');
}

function closeUserMenu() { document.querySelectorAll('.umenu').forEach((m) => m.remove()); }

function paintUser() {
  if (!el.user) return;
  const d = S.discord || {};
  const logged = !!d.loggedIn;
  const avatar = h('div', { class: 'avatar' + (logged && d.avatar ? ' img' : '') }, logged && d.avatar ? h('img', { src: d.avatar, alt: '', draggable: 'false' }) : icon('user'));
  const name = S.discordBusy ? 'Esperando a Discord…' : (logged ? d.name : 'Iniciar sesión');
  const sub = S.discordBusy ? 'Autorizá en el navegador' : (logged ? d.tier : 'con Discord');
  el.user.replaceChildren(avatar,
    h('div', { class: 'u-txt' }, h('b', {}, name), h('small', { class: logged ? 'tier ' + (TIER_CLASS[d.tier] || 'free') : '' }, sub)),
    logged ? icon('chevron', 'u-chev') : null);
  el.user.onclick = (e) => {
    e.stopPropagation();
    if (S.discordBusy) return;
    if (!logged) { discordLogin(); return; }
    if (document.querySelector('.umenu')) { closeUserMenu(); return; }
    const menu = h('div', { class: 'umenu' },
      h('button', { onclick: async () => { closeUserMenu(); await DiscordLogout(); S.discord = { ...d, loggedIn: false, name: '', avatar: '', tier: '' }; paintUser(); if (S.page === 'home') go('home'); toast('Sesión cerrada'); } }, 'Cerrar sesión'),
      h('button', { onclick: async () => { closeUserMenu(); await configureDiscord(); } }, 'Cambiar Application ID'));
    el.user.append(menu);
    setTimeout(() => document.addEventListener('click', closeUserMenu, { once: true }), 0);
  };
}
// ---------- layout ----------
function build() {
  el.nav = h('nav');
  el.user = h('div', { class: 'user' });
  el.content = h('div');
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
      h('div', { class: 'brand' }, h('div', { class: 'logo' }, 'ST'), h('div', {}, h('b', {}, 'South Tweaks'), h('small', {}, 'v2 · Go'))),
      el.nav,
      el.user),
    el.main);
  document.body.append(el.busy, el.queue, toasts);
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
  DiscordState().then((d) => { S.discord = d; paintUser(); if (S.page === 'home') go('home'); }).catch(() => {});
  await refresh();
  S.live = await GetLive();
  go('home');
  setInterval(tick, 2000);
})();