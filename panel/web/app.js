'use strict';
/* Serverwerk – Weboberfläche. Kein Framework, nur kleine Helfer. */

const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const root = $('#root');

/* ---------- Symbole (SVG, keine Emoji) ---------- */
const P = {
  server: '<rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  egg: '<path d="M12 3c3.9 0 7 6 7 10a7 7 0 0 1-14 0c0-4 3.1-10 7-10z"/>',
  users: '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20c.8-3.5 3.4-5.5 6.5-5.5s5.7 2 6.5 5.5"/><path d="M16 4.5a3.5 3.5 0 0 1 0 7M18.5 14.8c1.6.8 2.7 2.5 3 5.2"/>',
  cpu: '<rect x="6" y="6" width="12" height="12" rx="2"/><path d="M9 2v4M15 2v4M9 18v4M15 18v4M2 9h4M2 15h4M18 9h4M18 15h4"/>',
  logout: '<path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3M10 17l5-5-5-5M15 12H4"/>',
  play: '<path d="M7 4.5v15l12-7.5z"/>',
  stop: '<rect x="6" y="6" width="12" height="12" rx="2"/>',
  restart: '<path d="M20 12a8 8 0 1 1-2.3-5.7M20 4v5h-5"/>',
  kill: '<path d="M18 6 6 18M6 6l12 12"/>',
  folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  file: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>',
  dots: '<circle cx="12" cy="5" r="1.2"/><circle cx="12" cy="12" r="1.2"/><circle cx="12" cy="19" r="1.2"/>',
  upload: '<path d="M12 16V4M7 9l5-5 5 5M4 16v3a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-3"/>',
  download: '<path d="M12 4v12M7 11l5 5 5-5M4 16v3a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-3"/>',
  trash: '<path d="M4 7h16M10 11v6M14 11v6M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12M9 7V4h6v3"/>',
  edit: '<path d="M4 20h4L19 9l-4-4L4 16z"/>',
  back: '<path d="M15 6l-6 6 6 6"/>',
  refresh: '<path d="M20 12a8 8 0 1 1-2.3-5.7M20 4v5h-5"/>',
  archive: '<rect x="3" y="4" width="18" height="5" rx="1"/><path d="M5 9v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V9M10 13h4"/>',
  key: '<circle cx="8" cy="15" r="4"/><path d="M11 12l9-9M17 6l3 3"/>',
  check: '<path d="M5 12.5l4.5 4.5L19 7.5"/>',
  copy: '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/>',
  clock: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
  search: '<circle cx="11" cy="11" r="6.5"/><path d="M20 20l-4.2-4.2"/>',
  newfile: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5M12 12v6M9 15h6"/>',
  term: '<rect x="3" y="4" width="18" height="16" rx="2.5"/><path d="M7 9l3 3-3 3M12.5 15H17"/>',
  puzzle: '<path d="M10 4a2 2 0 1 1 4 0v2h4v4h-2a2 2 0 1 0 0 4h2v4h-4v-2a2 2 0 1 0-4 0v2H6v-4h2a2 2 0 1 0 0-4H6V6h4z"/>',
  sliders: '<path d="M4 7h10M18 7h2M4 17h4M12 17h8"/><circle cx="16" cy="7" r="2"/><circle cx="10" cy="17" r="2"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M12 2.5v2.2M12 19.3v2.2M4.6 4.6l1.6 1.6M17.8 17.8l1.6 1.6M2.5 12h2.2M19.3 12h2.2M4.6 19.4l1.6-1.6M17.8 6.2l1.6-1.6"/>',
  send: '<path d="M4 12l16-8-6 16-2.5-6.5z"/>',
  plug: '<path d="M9 3v5M15 3v5M6 8h12v3a6 6 0 0 1-12 0zM12 17v4"/>',
  newdir: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2zM12 10v6M9 13h6"/>',
};
const I = (n, cls = '') => `<svg class="i ${cls}" viewBox="0 0 24 24">${P[n] || ''}</svg>`;

/* ---------- Helfer ---------- */
async function api(method, path, body, opt = {}) {
  const o = { method, headers: { 'X-Serverwerk': '1' } };
  if (body instanceof FormData || typeof body === 'string' || body instanceof Blob) o.body = body;
  else if (body !== undefined) { o.body = JSON.stringify(body); o.headers['Content-Type'] = 'application/json'; }
  const r = await fetch('/api' + path, o);
  if (r.status === 401 && !opt.noAuth) { S.me = null; render(); throw new Error('Bitte anmelden'); }
  if (opt.raw) { if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText); return r; }
  const d = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(d.error || ('Fehler ' + r.status));
  return d;
}
function toast(msg, kind = '') {
  const t = document.createElement('div');
  t.className = 'toast ' + kind; t.textContent = msg;
  $('#toasts').append(t);
  setTimeout(() => { t.style.transition = 'opacity .3s'; t.style.opacity = 0; setTimeout(() => t.remove(), 300); }, kind === 'bad' ? 6000 : 3200);
}
const fail = e => toast(e.message || String(e), 'bad');
function bytes(n) {
  if (!n) return '0 B';
  const u = ['B', 'KB', 'MB', 'GB', 'TB']; let i = 0;
  while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
  return (i ? n.toFixed(n < 10 ? 1 : 0) : n) + ' ' + u[i];
}
function dur(s) {
  if (!s) return '–';
  const d = Math.floor(s / 86400), h = Math.floor(s % 86400 / 3600), m = Math.floor(s % 3600 / 60);
  const [D, H, M] = I18N.lang === 'en' ? ['d', 'h', 'min'] : ['T', 'Std', 'Min'];
  return d ? `${d} ${D} ${h} ${H}` : h ? `${h} ${H} ${m} ${M}` : m ? `${m} ${M} ${s % 60} s` : `${s} s`;
}
const when = t => t ? new Date(t).toLocaleString(I18N.lang === 'en' ? 'en-GB' : 'de-DE', { dateStyle: 'medium', timeStyle: 'short' }) : '–';
const stateText = { offline: 'Aus', starting: 'Startet', running: 'Läuft', stopping: 'Stoppt', installing: 'Installiert' };
const host = location.hostname;

function modal(html, { wide = false } = {}) {
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = `<div class="modal ${wide ? 'wide' : ''}">${html}</div>`;
  document.body.append(bg);
  const close = () => { bg.remove(); document.removeEventListener('keydown', key); };
  const key = e => { if (e.key === 'Escape') close(); };
  document.addEventListener('keydown', key);
  bg.addEventListener('mousedown', e => { if (e.target === bg) close(); });
  $$('[data-close]', bg).forEach(b => b.onclick = close);
  const f = $('input,select,textarea', bg); if (f) setTimeout(() => f.focus(), 30);
  return { el: $('.modal', bg), close };
}
function confirmBox(title, text, { ok = 'Ja', danger = false } = {}) {
  return new Promise(res => {
    const m = modal(`<h3>${esc(title)}</h3><p>${esc(text)}</p><div class="acts"><button class="btn" data-close>Abbrechen</button><button class="btn ${danger ? 'danger fill' : 'primary'}" data-ok>${esc(ok)}</button></div>`);
    $('[data-ok]', m.el).onclick = () => { m.close(); res(true); };
    $$('[data-close]', m.el).forEach(b => b.addEventListener('click', () => res(false)));
  });
}
function promptBox(title, label, value = '', { ok = 'OK', type = 'text' } = {}) {
  return new Promise(res => {
    const m = modal(`<h3>${esc(title)}</h3><form class="stack" style="margin-top:14px"><label class="f">${esc(label)}<input class="input" type="${type}" value="${esc(value)}"></label><div class="acts"><button type="button" class="btn" data-close>Abbrechen</button><button class="btn primary">${esc(ok)}</button></div></form>`);
    const inp = $('input', m.el);
    setTimeout(() => { inp.focus(); const dot = value.lastIndexOf('.'); inp.setSelectionRange(0, dot > 0 ? dot : value.length); }, 40);
    $('form', m.el).onsubmit = e => { e.preventDefault(); m.close(); res(inp.value); };
    $$('[data-close]', m.el).forEach(b => b.addEventListener('click', () => res(null)));
  });
}
function popMenu(anchor, items) {
  $$('.menu').forEach(m => m.remove());
  const m = document.createElement('div');
  m.className = 'menu';
  for (const it of items) {
    if (!it) continue;
    const b = document.createElement('button');
    b.className = it.danger ? 'danger' : '';
    b.innerHTML = (it.icon ? I(it.icon) : '') + esc(it.label);
    b.onclick = () => { m.remove(); it.run(); };
    m.append(b);
  }
  document.body.append(m);
  const r = anchor.getBoundingClientRect();
  const mw = m.offsetWidth, mh = m.offsetHeight;
  m.style.left = Math.max(8, Math.min(r.right - mw, innerWidth - mw - 8)) + 'px';
  m.style.top = (r.bottom + mh + 8 > innerHeight ? r.top - mh - 4 : r.bottom + 4) + 'px';
  setTimeout(() => document.addEventListener('mousedown', function off(e) { if (!m.contains(e.target)) { m.remove(); document.removeEventListener('mousedown', off); } }), 0);
}
function copy(text) {
  (navigator.clipboard?.writeText(text) || Promise.reject()).then(() => toast('Kopiert: ' + text), () => {
    const t = document.createElement('textarea'); t.value = text; document.body.append(t); t.select(); document.execCommand('copy'); t.remove(); toast('Kopiert: ' + text);
  });
}

/* ---------- ANSI-Farben der Konsole ---------- */
const ANSI = ['#2b2b30', '#e5807a', '#8fcf9a', '#e6c071', '#7fa7e8', '#c99ae0', '#76c7cf', '#d9d9de',
  '#6c6c74', '#ff9d96', '#a9e8b3', '#ffd98c', '#9cc0ff', '#e2b5f7', '#93e1e8', '#ffffff'];
function c256(n) {
  if (n < 16) return ANSI[n];
  if (n >= 232) { const g = 8 + (n - 232) * 10; return `rgb(${g},${g},${g})`; }
  n -= 16; const v = x => x ? x * 40 + 55 : 0;
  return `rgb(${v(Math.floor(n / 36))},${v(Math.floor(n / 6) % 6)},${v(n % 6)})`;
}
function ansiHtml(line) {
  // Minecraft-§-Codes, die ungefiltert durchkommen, in ANSI übersetzen.
  line = line.replace(/§([0-9a-fk-or])/gi, (_, c) => {
    const map = { 0: 30, 1: 34, 2: 32, 3: 36, 4: 31, 5: 35, 6: 33, 7: 37, 8: 90, 9: 94, a: 92, b: 96, c: 91, d: 95, e: 93, f: 97, l: 1, o: 3, n: 4, r: 0 };
    const k = map[c.toLowerCase()]; return k === undefined ? '' : `\x1b[${k}m`;
  });
  let out = '', st = {};
  const re = /\x1b\[([0-9;]*)m|\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07]*\x07/g;
  let last = 0, m;
  const span = txt => {
    if (!txt) return;
    const css = [];
    if (st.fg) css.push('color:' + st.fg);
    if (st.bg) css.push('background:' + st.bg);
    if (st.b) css.push('font-weight:700');
    if (st.i) css.push('font-style:italic');
    if (st.u) css.push('text-decoration:underline');
    if (st.d) css.push('opacity:.7');
    out += css.length ? `<span style="${css.join(';')}">${esc(txt)}</span>` : esc(txt);
  };
  while ((m = re.exec(line))) {
    span(line.slice(last, m.index)); last = re.lastIndex;
    if (m[1] === undefined) continue;
    const p = m[1] === '' ? [0] : m[1].split(';').map(Number);
    for (let i = 0; i < p.length; i++) {
      const c = p[i];
      if (c === 0) st = {};
      else if (c === 1) st.b = 1; else if (c === 2) st.d = 1; else if (c === 3) st.i = 1; else if (c === 4) st.u = 1;
      else if (c === 22) { st.b = 0; st.d = 0; } else if (c === 23) st.i = 0; else if (c === 24) st.u = 0;
      else if (c >= 30 && c <= 37) st.fg = ANSI[c - 30]; else if (c >= 90 && c <= 97) st.fg = ANSI[c - 82];
      else if (c === 39) st.fg = null; else if (c >= 40 && c <= 47) st.bg = ANSI[c - 40]; else if (c === 49) st.bg = null;
      else if (c >= 100 && c <= 107) st.bg = ANSI[c - 92];
      else if ((c === 38 || c === 48) && p[i + 1] === 5) { const col = c256(p[i + 2]); c === 38 ? st.fg = col : st.bg = col; i += 2; }
      else if ((c === 38 || c === 48) && p[i + 1] === 2) { const col = `rgb(${p[i + 2]},${p[i + 3]},${p[i + 4]})`; c === 38 ? st.fg = col : st.bg = col; i += 4; }
    }
  }
  span(line.slice(last));
  return out;
}

/* ---------- Sprache ----------
   Die Oberfläche ist auf Deutsch geschrieben. Für Englisch übersetzt ein
   Beobachter alle angezeigten Texte über i18n/en.json (dasselbe Wörterbuch
   wie im Server). Benutzerinhalte (.notr, Konsole, Editor, Beschreibungen) bleiben. */
const I18N = { lang: 'de', map: null, re: null, seen: new WeakMap(), attrs: new WeakMap() };
const LETTER = /[A-Za-zÄÖÜäöüß]/;
const NOTR = '.notr, .term, .editor, .prose, pre, code, textarea, script, style, .ini';
const tt = (de, en) => I18N.lang === 'en' ? en : de;
async function initLang(l) {
  I18N.lang = l === 'en' ? 'en' : 'de';
  document.documentElement.lang = I18N.lang;
  if (I18N.lang !== 'en' || I18N.re) return;
  try {
    const m = await fetch('i18n/en.json', { cache: 'no-cache' }).then(r => r.json());
    const keys = Object.keys(m).sort((a, b) => b.length - a.length);
    I18N.map = m;
    I18N.re = new RegExp(keys.map(k => k.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|'), 'g');
    trWalk(document.body);
    new MutationObserver(muts => {
      for (const mu of muts) {
        if (mu.type === 'characterData') trNode(mu.target);
        else if (mu.type === 'attributes') trAttr(mu.target, mu.attributeName);
        else mu.addedNodes.forEach(n => n.nodeType === 3 ? trNode(n) : n.nodeType === 1 && trWalk(n));
      }
    }).observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ['placeholder', 'title'] });
  } catch { }
}
function tr(str) {
  if (!I18N.re || !str) return str;
  return str.replace(I18N.re, (m, off, all) => {
    if ((LETTER.test(m[0]) && LETTER.test(all[off - 1] || '')) || (LETTER.test(m[m.length - 1]) && LETTER.test(all[off + m.length] || ''))) return m;
    return I18N.map[m];
  });
}
function trNode(n) {
  if (I18N.seen.get(n) === n.nodeValue) return;
  const el = n.parentElement;
  if (!el || el.closest(NOTR)) return;
  const v = tr(n.nodeValue);
  if (v !== n.nodeValue) n.nodeValue = v;
  I18N.seen.set(n, n.nodeValue);
}
function trAttr(el, a) {
  if (!el.getAttribute || el.closest(NOTR + ', input.notr')) return;
  const cur = el.getAttribute(a);
  const done = I18N.attrs.get(el) || {};
  if (cur == null || done[a] === cur) return;
  const v = tr(cur);
  done[a] = v; I18N.attrs.set(el, done);
  if (v !== cur) el.setAttribute(a, v);
}
function trWalk(root) {
  if (!I18N.re || !root) return;
  if (root.nodeType === 1 && root.closest(NOTR)) return;
  const w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  for (let n = w.nextNode(); n; n = w.nextNode()) trNode(n);
  const els = root.querySelectorAll ? [root, ...root.querySelectorAll('[placeholder],[title]')] : [];
  els.forEach(el => { if (el.hasAttribute?.('placeholder')) trAttr(el, 'placeholder'); if (el.hasAttribute?.('title')) trAttr(el, 'title'); });
}

/* ---------- Zustand & Routing ---------- */
const S = { me: null, cleanup: [] };
function cleanup() { S.cleanup.forEach(f => { try { f(); } catch { } }); S.cleanup = []; }
function onLeave(f) { S.cleanup.push(f); }
const go = h => { location.hash = h; };
window.addEventListener('hashchange', render);

async function boot() {
  const st = await api('GET', '/setup', undefined, { noAuth: true }).catch(() => ({}));
  await initLang(st.lang);
  try { S.me = await api('GET', '/me', undefined, { noAuth: true }); } catch { S.me = null; }
  if (S.me) { applyAccent(S.me.accent); if (S.me.panelName) document.title = S.me.panelName; }
  else S.setup = st.needed;
  render();
}

function render() {
  cleanup();
  $$('.menu').forEach(m => m.remove());
  if (!S.me) return S.setup ? renderSetup() : renderLogin();
  const h = location.hash.replace(/^#\/?/, '');
  const parts = h.split('/').filter(Boolean);
  if (!$('.app')) layout();
  const nav = parts[0] || '';
  $$('.nav a').forEach(a => a.classList.toggle('on', a.dataset.nav === (nav === 's' || nav === 'neu' ? '' : nav)));
  const main = $('.main');
  main.innerHTML = '<div class="page"></div>';
  const page = $('.page', main);
  const a = S.me.admin;
  if (!nav) return pageServers(page);
  if (nav === 's' && parts[1]) return pageServer(page, parts[1], parts[2] || 'konsole');
  if (nav === 'neu' && a) return pageCreate(page);
  if (nav === 'eggs' && a) return pageEggs(page, parts[1] || '', parts[2]);
  if (nav === 'einstellungen' && a) return pageSettings(page, parts[1] || 'allgemein');
  if (nav === 'allocations' && a) return pageAllocations(page);
  if (nav === 'benutzer' && a) return pageUsers(page);
  if (nav === 'system' && a) return pageSystem(page);
  if (nav === 'konto') return pageAccount(page);
  page.innerHTML = `<div class="empty"><span class="term-label">$ cd /${esc(h)}</span>Diese Seite gibt es nicht.</div>`;
}

// Erste Einrichtung: Admin-Konto selbst anlegen (mit Code aus der Installation).
function renderSetup() {
  const m = location.hash.match(/^#\/?einrichten\/([A-Za-z0-9]+)/);
  const code = m ? m[1] : '';
  root.innerHTML = `<div class="gate">
    <div class="brand-big"><h1>Server<em>werk</em></h1><p>$ einrichten</p></div>
    <div class="card"><form>
      <div><b style="font-size:16px">Willkommen</b><p class="muted" style="margin:4px 0 0;font-size:14px">Lege dein Admin-Konto an. Damit verwaltest du später alles.</p></div>
      ${code ? `<input type="hidden" name="code" value="${esc(code)}">` : `<label class="f">Einrichtungs-Code<input class="input mono" name="code" required autocomplete="off"><small>Steht am Ende der Installation, oder auf dem Server mit: sudo cat /var/lib/serverwerk/einrichtung.txt</small></label>`}
      <label class="f">Benutzername<input class="input" name="name" autocomplete="username" required minlength="2" maxlength="32" placeholder="z. B. admin"></label>
      <label class="f">Passwort<input class="input" name="pw" type="password" autocomplete="new-password" required minlength="8"><small>mindestens 8 Zeichen</small></label>
      <label class="f">Passwort wiederholen<input class="input" name="pw2" type="password" autocomplete="new-password" required minlength="8"></label>
      <div class="err"></div>
      <button class="btn primary big">Konto anlegen und loslegen</button>
    </form></div></div>`;
  const f = $('form');
  setTimeout(() => (code ? f.name : f.code).focus(), 50);
  f.onsubmit = async e => {
    e.preventDefault();
    const err = $('.err', f);
    if (f.pw.value !== f.pw2.value) { err.textContent = 'Die Passwörter sind nicht gleich.'; return; }
    try {
      await api('POST', '/setup', { code: f.code.value, name: f.name.value, password: f.pw.value }, { noAuth: true });
      S.setup = false;
      S.me = await api('GET', '/me');
      applyAccent(S.me.accent);
      history.replaceState(null, '', '#/allocations');
      root.innerHTML = '';
      render();
      toast('Konto angelegt. Als Nächstes: Ports festlegen.', 'good');
    } catch (ex) {
      if (/schon eingerichtet/.test(ex.message)) { S.setup = false; render(); return; }
      err.textContent = ex.message;
    }
  };
}

function renderLogin() {
  root.innerHTML = `<div class="gate">
    <div class="brand-big"><h1>Server<em>werk</em></h1><p>$ login</p></div>
    <div class="card"><form>
      <label class="f">Benutzername<input class="input" name="name" autocomplete="username" required></label>
      <label class="f">Passwort<input class="input" name="pw" type="password" autocomplete="current-password" required></label>
      <div class="err"></div>
      <button class="btn primary big">Anmelden</button>
    </form></div></div>`;
  const f = $('form');
  setTimeout(() => f.name.focus(), 50);
  f.onsubmit = async e => {
    e.preventDefault();
    try {
      await api('POST', '/login', { name: f.name.value, password: f.pw.value }, { noAuth: true });
      S.me = await api('GET', '/me');
      applyAccent(S.me.accent);
      root.innerHTML = '';
      render();
    } catch (err) { $('.err', f).textContent = err.message; }
  };
}

function layout() {
  const a = S.me.admin;
  root.innerHTML = `<div class="app">
    <aside class="side">
      <a class="brand" href="#/">Server<em>werk</em></a>
      <nav class="nav">
        <a href="#/" data-nav="">${I('server')}<span>${tt('Server', 'Servers')}</span></a>
      </nav>
      ${a ? `<div class="nav-h">Verwaltung</div><nav class="nav">
        <a href="#/eggs" data-nav="eggs">${I('egg')}<span>Eggs</span></a>
        <a href="#/allocations" data-nav="allocations">${I('plug')}<span>Allocations</span></a>
        <a href="#/benutzer" data-nav="benutzer">${I('users')}<span>Benutzer</span></a>
        <a href="#/system" data-nav="system">${I('cpu')}<span>System</span></a>
        <a href="#/einstellungen" data-nav="einstellungen">${I('gear')}<span>Einstellungen</span></a>
      </nav>` : ''}
      <div class="side-foot">
        <a href="#/konto" class="row grow" style="text-decoration:none;min-width:0"><span class="avatar">${esc(S.me.name[0])}</span><span class="who ell notr" style="font-size:14px">${esc(S.me.name)}</span></a>
        <button class="ib" id="logout" title="Abmelden">${I('logout')}</button>
      </div>
    </aside>
    <main class="main"></main></div>`;
  $('#logout').onclick = async () => { await api('POST', '/logout').catch(() => { }); S.me = null; root.innerHTML = ''; render(); };
}

/* ---------- Logos ---------- */
function initials(name) {
  const w = String(name || '?').replace(/[^\p{L}\p{N} ]/gu, ' ').split(/\s+/).filter(Boolean);
  return esc(((w[0]?.[0] || '?') + (w[1]?.[0] || '')).toUpperCase());
}
// Bild mit unscharfem Hintergrund; ohne Bild erscheinen Initialen.
function thumb(src, name, cls = '') {
  const s = esc(src);
  return `<span class="thumb ${cls}"><span class="ini">${initials(name)}</span>${src ? `<img class="bg" src="${s}" alt="" loading="lazy" onerror="this.parentNode.classList.add('none')"><img class="fg" src="${s}" alt="" loading="lazy" onload="this.parentNode.classList.add(this.naturalWidth/this.naturalHeight>1.5?'wide':'small')" onerror="this.parentNode.classList.add('none')">` : ''}</span>`;
}
const eggIcon = id => id ? `/api/eggs/${id}/icon?v=3` : '';

/* ---------- Server-Übersicht ---------- */
function barHtml(label, val, max, fmt) {
  const pct = max ? Math.min(100, val / max * 100) : 0;
  return `<div class="bar ${pct > 90 ? 'hot' : ''}"><small>${label}<b>${fmt}</b></small><i style="--w:${pct}%"></i></div>`;
}
function cardStats(s) {
  const st = s.stats || {}, on = s.state !== 'offline';
  const cpuMax = s.cpu || 100;
  return barHtml('CPU', on ? st.cpu : 0, cpuMax, on ? (st.cpu || 0).toFixed(0) + ' %' : '–') +
    barHtml('RAM', on ? st.mem : 0, s.memory ? s.memory * 1048576 : st.memMax, on ? bytes(st.mem) : '–') +
    barHtml('Platte', st.disk || 0, s.disk ? s.disk * 1048576 : 0, bytes(st.disk || 0));
}
function stateOf(s) {
  if (s.install === 'importing') return ['installing', 'Importiert'];
  if (s.install === 'installing' || s.state === 'installing') return ['installing', 'Installiert'];
  if (s.install === 'failed' && s.state === 'offline') return ['failed', 'Installation fehlgeschlagen'];
  return [s.state, stateText[s.state] || s.state];
}

async function pageServers(page) {
  page.innerHTML = `<div class="head"><div><h1>${tt('Server', 'Servers')}</h1><div class="sub term-label">$ ls /server</div></div>
    <div class="tools">${S.me.admin ? `<a class="btn primary" href="#/neu">${I('plus')}Neuer Server</a>` : ''}</div></div>
    <div id="portwarn"></div><div class="cards" id="cards"><div class="muted">Lädt ...</div></div>`;
  if (S.me.admin) api('GET', '/ports/free').then(p => {
    if (p.configured || !$('#portwarn')) return;
    $('#portwarn').innerHTML = `<div class="banner"><span class="grow"><b>Erst Ports festlegen.</b> Bevor du Server anlegen kannst, trag unter Allocations ein, welche Ports im Router freigegeben sind.</span><a class="btn small primary" href="#/allocations">Zu den Allocations</a></div>`;
  }).catch(() => { });
  const draw = list => {
    const el = $('#cards'); if (!el) return;
    if (!list.length) {
      el.outerHTML = `<div class="empty" id="cards"><span class="term-label">$ ls /server<br>(leer)</span>${S.me.admin ? tt('Noch keine Server. <a href="#/neu">Ersten Server anlegen</a> – vorher unter <a href="#/eggs/katalog">Eggs</a> ein Spiel holen.', 'No servers yet. <a href="#/neu">Create your first server</a> – get a game under <a href="#/eggs/katalog">Eggs</a> first.') : 'Dir wurde noch kein Server freigegeben.'}</div>`;
      return;
    }
    el.className = 'cards';
    el.innerHTML = list.map(s => {
      const [cls, txt] = stateOf(s);
      return `<a class="scard" href="#/s/${s.id}">
        <div class="row">${thumb(eggIcon(s.egg), s.eggName || s.name, 'sq')}
          <div class="grow" style="min-width:0"><h3 class="ell notr">${esc(s.name)}</h3><div class="meta"><span class="ell">${esc(s.eggName || 'Egg fehlt')}</span></div></div>
          <span class="status ${cls}">${txt}</span></div>
        <div class="row"><span class="addr" data-copy="${esc(serverAddr(s))}">${esc(serverAddr(s))}</span></div>
        <div class="bars">${cardStats(s)}</div></a>`;
    }).join('');
    $$('[data-copy]', el).forEach(b => b.onclick = e => { e.preventDefault(); copy(b.dataset.copy); });
  };
  const load = () => api('GET', '/servers').then(draw).catch(fail);
  await load();
  const t = setInterval(load, 3000);
  onLeave(() => clearInterval(t));
}

/* ---------- Einzelner Server ---------- */
const TABS = [
  ['konsole', 'Konsole', 'console', 'term'], ['dateien', 'Dateien', 'files', 'folder'], ['addons', 'Plugins', 'files', 'puzzle'],
  ['backups', 'Backups', 'backups', 'archive'], ['zeitplaene', 'Zeitpläne', 'schedules', 'clock'], ['start', 'Startparameter', 'startup', 'sliders'],
  ['benutzer', 'Benutzer', 'owner', 'users'], ['einstellungen', 'Einstellungen', 'settings', 'gear'],
];

async function pageServer(page, id, tab) {
  let d;
  try { d = await api('GET', '/servers/' + id); } catch (e) { page.innerHTML = `<div class="empty">${esc(e.message)}</div>`; return; }
  const s = d.server, perms = s.perms || [];
  const has = p => perms.includes(p) || perms.includes('owner');
  const tabs = TABS.filter(t => (t[0] !== 'addons' || d.addons) && (t[2] === 'owner' ? perms.includes('owner') : has(t[2])));
  if (!tabs.find(t => t[0] === tab)) tab = tabs[0]?.[0] || 'konsole';
  const tabName = t => t[0] === 'addons' ? d.addons.label : t[1];
  page.innerHTML = `
    <div class="shead">
      <a class="ib" href="#/" title="Zurück zur Übersicht">${I('back')}</a>
      ${thumb(eggIcon(s.egg), s.eggName || s.name, 'sq big')}
      <div class="grow" style="min-width:0"><h1 class="ell notr">${esc(s.name)}</h1>
        <div class="row wrap" style="margin-top:6px;gap:8px"><span class="status" id="st"></span><span class="faint" id="upt"></span>
          <span class="tag">${esc(s.eggName)}</span><span class="addr" id="addr" title="Adresse kopieren">${I('copy', 'xs')}${esc(serverAddr(s))}</span></div></div>
      ${has('power') ? `<div class="pgroup">
        <button class="btn primary" id="pmain"></button>
        <button class="ib ring" data-p="restart" title="Neustarten">${I('restart')}</button>
        <button class="ib ring danger" data-p="kill" title="Sofort beenden (wie Strom ziehen)">${I('kill')}</button>
      </div>` : ''}
    </div>
    <nav class="tabs">${tabs.map(t => `<a href="#/s/${id}/${t[0]}" class="${t[0] === tab ? 'on' : ''}">${I(t[3])}${esc(tabName(t))}</a>`).join('')}</nav>
    <div id="tab"></div>`;
  $('#addr').onclick = () => copy(serverAddr(s));

  // Live-Verbindung für Status, Werte und Konsole.
  const live = { state: s.state, stats: s.stats, hint: '', lines: [], on: {} };
  const setState = st => {
    live.state = st;
    const [cls, txt] = stateOf({ ...s, state: st, install: st === 'installing' ? 'installing' : (s.install === 'installing' && st === 'offline' ? 'ok' : s.install) });
    const el = $('#st'); if (el) { el.className = 'status ' + cls; el.textContent = txt; }
    const m = $('#pmain');
    if (m) {
      const on = st === 'running' || st === 'starting';
      m.innerHTML = st === 'stopping' ? '<span class="spin"></span>Stoppt ...' : on ? I('stop') + 'Stoppen' : I('play') + 'Starten';
      m.className = 'btn ' + (on || st === 'stopping' ? '' : 'primary');
      m.disabled = st === 'installing' || st === 'stopping';
      m.dataset.p = on ? 'stop' : 'start';
    }
    $$('.pgroup .ib[data-p]').forEach(b => { b.disabled = st === 'installing' || st === 'offline'; });
    upt();
    live.on.state?.(st);
  };
  const upt = () => {
    const e = $('#upt'); if (!e) return;
    e.textContent = live.state === 'running' && live.stats?.uptime ? tt('seit ', 'for ') + dur(live.stats.uptime) : '';
  };
  setState(s.state);
  $$('.pgroup [data-p], #pmain').forEach(b => b.onclick = async () => {
    const p = b.dataset.p;
    if (p === 'kill' && !await confirmBox('Server sofort beenden?', 'Wie Stecker ziehen: Nicht gespeicherte Daten können verloren gehen.', { ok: 'Beenden', danger: true })) return;
    api('POST', `/servers/${id}/power`, { action: p }).catch(fail);
  });

  // EULA und Java-Hinweis als Popup über allem.
  let hintShown = '';
  const showHint = h => {
    if (!h || h === hintShown) { hintShown = h; return; }
    hintShown = h;
    if (h === 'eula') {
      const m = modal(`<div class="mhead">${thumb(eggIcon(s.egg), s.eggName, 'sq')}<div><h3>Minecraft-EULA</h3><p style="margin:0">${esc(s.name)}</p></div></div>
        <p style="margin-top:14px">Damit der Server startet, musst du die Lizenzbedingungen von Mojang akzeptieren: <a href="https://aka.ms/MinecraftEULA" target="_blank" rel="noopener">Minecraft-EULA lesen</a>.</p>
        <div class="acts"><button class="btn" data-close>Später</button><button class="btn primary" data-ok>${I('check')}Akzeptieren und starten</button></div>`);
      $('[data-ok]', m.el).onclick = () => { m.close(); api('POST', `/servers/${id}/eula`).then(() => toast('EULA akzeptiert, Server startet', 'good')).catch(fail); };
    } else if (h === 'java') {
      const m = modal(`<h3>Falsche Java-Version</h3><p>Der Server braucht eine andere Java-Version als das gewählte Image. Unter Startparameter kannst du ein anderes Java auswählen.</p>
        <div class="acts"><button class="btn" data-close>Schließen</button><a class="btn primary" href="#/s/${id}/start" data-go>Zu den Startparametern</a></div>`);
      $('[data-go]', m.el).addEventListener('click', m.close);
    }
  };

  const es = new EventSource(`/api/servers/${id}/events`);
  es.onmessage = ev => {
    const e = JSON.parse(ev.data);
    if (e.t === 'init') { live.lines = e.d.lines || []; live.stats = e.d.stats; setState(e.d.state); live.on.init?.(); showHint(e.d.hint); }
    else if (e.t === 'line') { live.lines.push(e.d); if (live.lines.length > 1000) live.lines.shift(); live.on.line?.(e.d); }
    else if (e.t === 'status') setState(e.d);
    else if (e.t === 'stats') { live.stats = e.d; upt(); live.on.stats?.(e.d); }
    else if (e.t === 'hint') showHint(e.d);
    else if (e.t === 'install') { s.install = e.d; setState(live.state); toast(e.d === 'ok' ? 'Installation fertig' : 'Installation fehlgeschlagen', e.d === 'ok' ? 'good' : 'bad'); }
  };
  onLeave(() => es.close());

  const el = $('#tab');
  const ctx = { id, s, d, live, has, el, reload: () => render() };
  ({ konsole: tabConsole, dateien: tabFiles, addons: tabAddons, backups: tabBackups, zeitplaene: tabSchedules, start: tabStartup, benutzer: tabSubusers, einstellungen: tabSettings })[tab](ctx);
}

/* ---------- Konsole ---------- */
function spark(vals, max) {
  if (vals.length < 2) return '<svg class="spark"></svg>';
  const m = max || Math.max(...vals, 1);
  const pts = vals.map((v, i) => `${(i / (vals.length - 1) * 100).toFixed(1)},${(30 - Math.min(v, m) / m * 28).toFixed(1)}`);
  return `<svg class="spark" viewBox="0 0 100 30" preserveAspectRatio="none"><polygon points="0,30 ${pts.join(' ')} 100,30"/><polyline points="${pts.join(' ')}"/></svg>`;
}

function tabConsole(c) {
  const { id, s, live, has } = c;
  c.el.innerHTML = `
    <div class="stats">
      <div class="tile"><small>CPU</small><b id="t-cpu">–</b><div id="g-cpu"></div></div>
      <div class="tile"><small>Arbeitsspeicher</small><b id="t-mem">–</b><div id="g-mem"></div></div>
      <div class="tile"><small>Speicherplatz</small><b id="t-disk">–</b><div class="bar" id="g-disk"><i></i></div></div>
      <div class="tile"><small>Netzwerk</small><b id="t-net">–</b><div class="faint" style="font-size:12px;margin-top:8px">seit Start</div></div>
    </div>
    <div class="term"><div class="term-out" id="out"></div>
      ${has('console') ? `<form class="term-in" id="cmdf"><span>$</span><input id="cmd" placeholder="${tt('Befehl eingeben ...', 'Enter a command ...')}" autocomplete="off" spellcheck="false"><button class="ib" title="Senden">${I('send')}</button></form>` : ''}
    </div>`;
  const out = $('#out');
  const near = () => out.scrollHeight - out.scrollTop - out.clientHeight < 60;
  const lineEl = l => {
    const div = document.createElement('div');
    div.innerHTML = ansiHtml(l) || '&nbsp;';
    return div;
  };
  const drawAll = () => {
    const f = document.createDocumentFragment();
    live.lines.forEach(l => f.append(lineEl(l)));
    out.replaceChildren(f);
    out.scrollTop = out.scrollHeight;
  };
  let queue = [], raf = 0;
  live.on.line = l => {
    queue.push(l);
    if (raf) return;
    raf = requestAnimationFrame(() => {
      const stick = near();
      const f = document.createDocumentFragment();
      queue.forEach(x => f.append(lineEl(x)));
      queue = []; raf = 0;
      out.append(f);
      while (out.childElementCount > 1000) out.firstElementChild.remove();
      if (stick) out.scrollTop = out.scrollHeight;
    });
  };
  live.on.init = drawAll;
  drawAll();

  const cpuH = [], memH = [];
  const stats = st => {
    if (!$('#t-cpu')) return;
    const on = live.state !== 'offline';
    const cpuMax = s.cpu || 0;
    $('#t-cpu').innerHTML = on ? `${(st.cpu || 0).toFixed(1)} %<span> / ${cpuMax ? cpuMax + ' %' : 'unbegrenzt'}</span>` : '<span>aus</span>';
    $('#t-mem').innerHTML = on ? `${bytes(st.mem)}<span> / ${s.memory ? bytes(s.memory * 1048576) : 'unbegrenzt'}</span>` : '<span>aus</span>';
    $('#t-disk').innerHTML = `${bytes(st.disk)}<span> / ${s.disk ? bytes(s.disk * 1048576) : 'unbegrenzt'}</span>`;
    $('#g-disk').style.setProperty('--w', s.disk ? Math.min(100, (st.disk || 0) / (s.disk * 1048576) * 100) + '%' : '0%');
    $('#t-net').innerHTML = on ? `${bytes(st.rx)}<span> ${tt('rein', 'in')}</span> &nbsp;${bytes(st.tx)}<span> ${tt('raus', 'out')}</span>` : '<span>aus</span>';
    if (on) {
      cpuH.push(st.cpu || 0); memH.push(st.mem || 0);
      if (cpuH.length > 60) { cpuH.shift(); memH.shift(); }
    }
    $('#g-cpu').innerHTML = spark(cpuH, cpuMax || 100);
    $('#g-mem').innerHTML = spark(memH, s.memory ? s.memory * 1048576 : 0);
  };
  live.on.stats = stats;
  stats(live.stats || {});
  live.on.state = () => stats(live.stats || {});

  const f = $('#cmdf');
  if (f) {
    const inp = $('#cmd');
    const hist = JSON.parse(sessionStorage.getItem('sw-hist-' + id) || '[]');
    let hi = hist.length;
    f.onsubmit = e => {
      e.preventDefault();
      const v = inp.value.trim(); if (!v) return;
      hist.push(v); if (hist.length > 50) hist.shift(); hi = hist.length;
      try { sessionStorage.setItem('sw-hist-' + id, JSON.stringify(hist)); } catch { }
      inp.value = '';
      api('POST', `/servers/${id}/command`, { command: v }).catch(fail);
    };
    inp.onkeydown = e => {
      if (e.key === 'ArrowUp' && hi > 0) { inp.value = hist[--hi]; e.preventDefault(); }
      else if (e.key === 'ArrowDown') { hi = Math.min(hist.length, hi + 1); inp.value = hist[hi] || ''; e.preventDefault(); }
    };
    if (matchMedia('(hover:hover)').matches) inp.focus();
  }
}

/* ---------- Plugins / Mods (Modrinth) ---------- */
const MR = 'https://api.modrinth.com/v2';
function compact(n) { return n >= 1e6 ? (n / 1e6).toFixed(1).replace('.0', '') + ' Mio.' : n >= 1e3 ? Math.round(n / 1e3) + ' Tsd.' : String(n); }

async function tabAddons(c) {
  const { id, d } = c;
  const info = d.addons;
  let view = sessionStorage.getItem('sw-addview') || 'entdecken';
  let q = '', sort = 'relevance', ver = info.mcVersion || '', offset = 0, hits = [], total = 0;
  let installed = [];
  const what = { plugin: 'Plugins', mod: 'Mods', datapack: 'Datapacks' }[info.kind];
  c.el.innerHTML = `<div class="row wrap" style="margin-bottom:16px">
      <div class="chips"><button class="chip" data-v="entdecken">${I('search')}Entdecken</button><button class="chip" data-v="installiert">Installiert <span id="icount"></span></button></div>
      <span class="grow"></span><span class="term-label">$ ${esc(info.folder)}/ &nbsp;·&nbsp; ${esc(info.loaders.join(', '))} &nbsp;·&nbsp; MC ${esc(info.mcVersion || '?')}</span></div>
    <div id="abody"></div>`;
  const loadInstalled = async () => {
    const r = await api('GET', `/servers/${id}/addons`).catch(fail);
    installed = r?.installed || [];
    $('#icount').textContent = installed.length ? `(${installed.length})` : '';
  };
  const byProject = p => installed.find(x => x.addon?.project === p);
  const setView = v => {
    view = v; sessionStorage.setItem('sw-addview', v);
    $$('[data-v]').forEach(b => b.classList.toggle('on', b.dataset.v === v));
    v === 'entdecken' ? discover() : list();
  };
  $$('[data-v]').forEach(b => b.onclick = () => setView(b.dataset.v));

  const restartHint = `<div class="banner"><span class="grow">Neue ${what} werden beim nächsten Neustart geladen.</span><button class="btn small" id="rst">${I('restart')}Jetzt neu starten</button></div>`;
  const bindRestart = () => { const b = $('#rst'); if (b) b.onclick = () => api('POST', `/servers/${id}/power`, { action: c.live.state === 'offline' ? 'start' : 'restart' }).then(() => { toast('Server startet neu'); $('#rst').closest('.banner').remove(); }).catch(fail); };
  let changed = false;

  const search = async (more = false) => {
    if (!more) { offset = 0; hits = []; }
    const facets = [info.loaders.map(l => 'categories:' + l)];
    if (ver) facets.push(['versions:' + ver]);
    if (info.kind === 'mod') { facets.push(['project_type:mod']); facets.push(['server_side:required', 'server_side:optional']); }
    const url = `${MR}/search?limit=24&offset=${offset}&index=${sort}&query=${encodeURIComponent(q)}&facets=${encodeURIComponent(JSON.stringify(facets))}`;
    const r = await fetch(url).then(x => x.json()).catch(() => null);
    if (!r?.hits) { $('#alist').innerHTML = '<div class="empty">Modrinth ist gerade nicht erreichbar.</div>'; return; }
    total = r.total_hits; hits = hits.concat(r.hits); offset += r.hits.length;
    drawHits();
  };
  const drawHits = () => {
    const box = $('#alist'); if (!box) return;
    if (!hits.length) { box.innerHTML = `<div class="empty"><span class="term-label">$ search "${esc(q)}"<br>0 Treffer</span>Nichts gefunden${ver ? ` für Minecraft ${esc(ver)}` : ''}.</div>`; return; }
    box.innerHTML = `<div class="addons">${hits.map((h, i) => {
      const inst = byProject(h.project_id);
      return `<div class="addon" data-open="${i}" title="Details ansehen">${thumb(h.icon_url, h.title, 'sq')}
        <div class="grow" style="min-width:0"><div class="row" style="gap:6px"><b class="ell">${esc(h.title)}</b></div>
        <div class="faint ell" style="font-size:12px">${tt('von', 'by')} ${esc(h.author)} · ${I('download', 'xs')} ${compact(h.downloads)}</div>
        <p>${esc(h.description)}</p></div>
        <div class="acol"><button class="btn small ${inst ? '' : 'primary'}" data-i="${i}">${inst ? I('refresh') + 'Neu laden' : I('download') + 'Installieren'}</button></div></div>`;
    }).join('')}</div>
    ${offset < total ? `<div style="text-align:center;margin-top:16px"><button class="btn" id="more">Mehr laden (${total - offset} weitere)</button></div>` : ''}`;
    $$('[data-i]', box).forEach(b => b.onclick = ev => { ev.stopPropagation(); install(hits[b.dataset.i], b); });
    $$('[data-open]', box).forEach(card => card.onclick = () => openDetail(hits[card.dataset.open]));
    const m = $('#more'); if (m) m.onclick = () => search(true);
  };
  const installVersion = async (h, v) => {
    const file = v.files.find(f => f.primary) || v.files[0];
    await api('POST', `/servers/${id}/addons`, { url: file.url, file: file.filename, project: h.project_id, title: h.title, icon: h.icon_url || '', version: v.version_number, replace: byProject(h.project_id)?.file?.replace(/\.disabled$/, '') || '' });
    toast(`${h.title} ${v.version_number} installiert`, 'good');
    changed = true;
    await loadInstalled();
  };
  const install = async (h, btn) => {
    btn.disabled = true; btn.innerHTML = '<span class="spin"></span>Lädt';
    try {
      const p = new URLSearchParams({ loaders: JSON.stringify(info.loaders) });
      if (ver) p.set('game_versions', JSON.stringify([ver]));
      const vs = await fetch(`${MR}/project/${h.project_id}/version?${p}`).then(r => r.json());
      if (!Array.isArray(vs) || !vs.length) throw new Error(`Keine passende Version${ver ? ' für Minecraft ' + ver : ''} gefunden`);
      await installVersion(h, vs.find(x => x.version_type === 'release') || vs[0]);
      btn.disabled = false; btn.className = 'btn small'; btn.innerHTML = I('check') + 'Installiert';
      if (!$('#rhint').innerHTML) { $('#rhint').innerHTML = restartHint; bindRestart(); }
    } catch (e) { fail(e); btn.disabled = false; btn.innerHTML = I('download') + 'Installieren'; }
  };
  const openDetail = h => {
    const y = scrollY;
    addonDetail($('#abody'), {
      hit: h, info, ver, installedFile: byProject(h.project_id)?.file,
      onInstall: (p, v) => installVersion({ project_id: p.id, title: p.title, icon_url: p.icon_url }, v),
      onBack: () => { discover(); requestAnimationFrame(() => scrollTo(0, y)); },
    });
    scrollTo(0, 0);
  };
  const discover = () => {
    $('#abody').innerHTML = `<div id="rhint">${changed ? restartHint : ''}</div>
      <div class="row wrap" style="margin-bottom:16px">
        <div class="search grow" style="max-width:460px">${I('search')}<input class="input" id="aq" placeholder="${tt(`${what} suchen (z. B.`, `Search ${what.toLowerCase()} (e.g.`)} ${info.kind === 'plugin' ? 'LuckPerms, EssentialsX' : info.kind === 'mod' ? 'Lithium, Sodium' : 'Vanilla Tweaks'}) ..." value="${esc(q)}"></div>
        <select class="input" id="asort" style="width:auto;height:38px"><option value="relevance">Relevanz</option><option value="downloads">Downloads</option><option value="follows">Beliebt</option><option value="updated">Aktualisiert</option><option value="newest">Neu</option></select>
        <label class="row" style="gap:6px"><span class="muted" style="font-size:13px">Minecraft</span><input class="input mono" id="aver" value="${esc(ver)}" placeholder="alle" style="width:90px;height:38px"></label>
      </div>
      <div id="alist"><div class="row muted"><span class="spin"></span>Sucht bei Modrinth ...</div></div>
      <p class="term-label" style="margin-top:14px">Quelle: modrinth.com</p>`;
    bindRestart();
    $('#asort').value = sort;
    let t;
    $('#aq').oninput = e => { q = e.target.value; clearTimeout(t); t = setTimeout(() => search(), 300); };
    $('#asort').onchange = e => { sort = e.target.value; search(); };
    $('#aver').onchange = e => { ver = e.target.value.trim(); search(); };
    if (hits.length) drawHits(); else search();
  };
  const list = () => {
    $('#abody').innerHTML = `<div id="rhint">${changed ? restartHint : ''}</div>` + (installed.length ? `<div class="panel">${installed.map((x, i) => `
      <div class="irow ${x.off ? 'off' : ''}">${thumb(x.addon?.icon || '', x.addon?.title || x.file, 'sq sm')}
        <div class="grow" style="min-width:0"><b class="ell" style="display:block">${esc(x.addon?.title || x.file.replace(/\.(jar|zip)(\.disabled)?$/, ''))}</b>
        <span class="faint mono ell" style="font-size:12px;display:block">${esc(x.file)}${x.addon?.version ? ' · ' + esc(x.addon.version) : ''} · ${bytes(x.size)}</span></div>
        <label class="check" title="An/aus"><span class="switch"><input type="checkbox" data-t="${i}" ${x.off ? '' : 'checked'}><i></i></span></label>
        <button class="ib danger" data-r="${i}" title="Entfernen">${I('trash')}</button></div>`).join('')}</div>`
      : `<div class="empty"><span class="term-label">$ ls ${esc(info.folder)}/<br>(leer)</span>${tt(`Noch keine ${what}. Unter Entdecken findest du tausende.`, `No ${what.toLowerCase()} yet. You'll find thousands under Discover.`)}</div>`);
    bindRestart();
    $$('[data-t]').forEach(x => x.onchange = () => api('POST', `/servers/${id}/addons/remove`, { file: installed[x.dataset.t].file, toggle: true }).then(async () => { changed = true; await loadInstalled(); list(); }).catch(fail));
    $$('[data-r]').forEach(b => b.onclick = async () => {
      const x = installed[b.dataset.r];
      if (!await confirmBox('Entfernen?', x.addon?.title || x.file, { ok: 'Entfernen', danger: true })) return;
      api('POST', `/servers/${id}/addons/remove`, { file: x.file }).then(async () => { changed = true; await loadInstalled(); list(); }).catch(fail);
    });
  };
  await loadInstalled();
  setView(view);
}

/* ---------- Dateien ---------- */
const ARCH = /\.(zip|tar|tar\.gz|tgz)$/i;
function tabFiles(c) {
  const { id } = c;
  let path = sessionStorage.getItem('sw-path-' + id) || '/';
  let entries = [], sel = new Set();
  c.el.innerHTML = `<div class="panel" id="drop">
    <div class="ph"><div class="crumbs grow" id="crumbs"></div>
      <div class="row" id="selacts" hidden><span class="muted" id="selcount" style="font-size:13px"></span>
        <button class="btn small" id="s-zip">${I('archive')}Packen</button>
        <button class="btn small danger" id="s-del">${I('trash')}Löschen</button></div>
      <button class="ib" id="f-newfile" title="Neue Datei">${I('newfile')}</button>
      <button class="ib" id="f-newdir" title="Neuer Ordner">${I('newdir')}</button>
      <button class="ib" id="f-up" title="Hochladen">${I('upload')}</button>
      <button class="ib" id="f-reload" title="Neu laden">${I('refresh')}</button>
      <input type="file" id="f-input" multiple hidden></div>
    <div class="list" id="files"></div></div>
    <p class="term-label" style="margin-top:12px">Tipp: Dateien oder ganze Ordner einfach irgendwo auf die Seite ziehen. Archive (.zip, .tar.gz) lassen sich über das Menü entpacken.</p>`;
  const join = n => (path.endsWith('/') ? path : path + '/') + n;
  const load = async () => {
    try { entries = await api('GET', `/servers/${id}/files?path=${encodeURIComponent(path)}`); }
    catch (e) { if (path !== '/') { path = '/'; return load(); } fail(e); entries = []; }
    sessionStorage.setItem('sw-path-' + id, path);
    sel.clear(); draw();
  };
  const crumbs = () => {
    const parts = path.split('/').filter(Boolean);
    let acc = '';
    $('#crumbs').innerHTML = `<a href="#" data-to="/">/home/container</a>` + parts.map(p => { acc += '/' + p; return `<span class="faint">/</span><a href="#" data-to="${esc(acc)}">${esc(p)}</a>`; }).join('');
    $$('#crumbs a').forEach(a => a.onclick = e => { e.preventDefault(); path = a.dataset.to; load(); });
  };
  const draw = () => {
    crumbs();
    const list = $('#files');
    if (!entries.length) { list.innerHTML = `<div class="empty" style="margin:14px;border:0">Ordner ist leer.</div>`; updSel(); return; }
    list.innerHTML = `<div class="li headrow"><input type="checkbox" id="all"><span>Name</span><span class="sz">Größe</span><span class="dt">Geändert</span><span></span></div>` +
      entries.map((e, i) => `<div class="li ${sel.has(e.name) ? 'sel' : ''}" data-i="${i}">
        <input type="checkbox" ${sel.has(e.name) ? 'checked' : ''}>
        <span class="nm ${e.dir ? 'dir' : ''}">${I(e.dir ? 'folder' : 'file')}<span class="ell notr">${esc(e.name)}${e.link ? ' <span class="faint">(Link)</span>' : ''}</span></span>
        <span class="sz">${e.dir ? '' : bytes(e.size)}</span><span class="dt">${when(e.mod)}</span>
        <button class="ib" data-m>${I('dots')}</button></div>`).join('');
    $('#all').onchange = ev => { entries.forEach(e => ev.target.checked ? sel.add(e.name) : sel.delete(e.name)); draw(); };
    $$('#files .li[data-i]').forEach(row => {
      const e = entries[row.dataset.i];
      $('input', row).onchange = ev => { ev.target.checked ? sel.add(e.name) : sel.delete(e.name); row.classList.toggle('sel', ev.target.checked); updSel(); };
      $('.nm', row).onclick = () => e.dir ? (path = join(e.name), load()) : e.editable ? openEditor(id, join(e.name), load) : download(e);
      $('[data-m]', row).onclick = ev => popMenu(ev.currentTarget, [
        !e.dir && e.editable && { label: 'Bearbeiten', icon: 'edit', run: () => openEditor(id, join(e.name), load) },
        { label: 'Umbenennen / Verschieben', icon: 'edit', run: () => rename(e) },
        { label: e.dir ? 'Als .tar.gz laden' : 'Herunterladen', icon: 'download', run: () => download(e) },
        ARCH.test(e.name) && { label: 'Hier entpacken', icon: 'archive', run: () => api('POST', `/servers/${id}/files/decompress`, { path, name: e.name }).then(() => { toast('Entpackt', 'good'); load(); }).catch(fail) },
        { label: 'Packen (.zip)', icon: 'archive', run: () => compress([e.name]) },
        { label: 'Löschen', icon: 'trash', danger: true, run: () => del([e.name]) },
      ]);
    });
    updSel();
  };
  const updSel = () => {
    $('#selacts').hidden = !sel.size;
    $('#selcount').textContent = sel.size + ' ausgewählt';
  };
  const download = e => { location.href = `/api/servers/${id}/files/download?path=${encodeURIComponent(join(e.name))}`; };
  const rename = async e => {
    const to = await promptBox('Umbenennen', 'Neuer Name oder Pfad (mit / verschieben)', e.name);
    if (!to || to === e.name) return;
    const target = to.startsWith('/') ? to : join(to);
    api('POST', `/servers/${id}/files/rename`, { from: join(e.name), to: target }).then(load).catch(fail);
  };
  const del = async names => {
    if (!await confirmBox(`${names.length === 1 ? names[0] : names.length + ' Einträge'} löschen?`, 'Das lässt sich nicht rückgängig machen.', { ok: 'Löschen', danger: true })) return;
    api('POST', `/servers/${id}/files/delete`, { path, names }).then(load).catch(fail);
  };
  const compress = names => api('POST', `/servers/${id}/files/compress`, { path, names }).then(r => { toast('Gepackt: ' + r.name, 'good'); load(); }).catch(fail);
  $('#s-del').onclick = () => del([...sel]);
  $('#s-zip').onclick = () => compress([...sel]);
  $('#f-reload').onclick = load;
  $('#f-newdir').onclick = async () => {
    const n = await promptBox('Neuer Ordner', 'Name'); if (!n) return;
    api('POST', `/servers/${id}/files/mkdir`, { path: join(n) }).then(load).catch(fail);
  };
  $('#f-newfile').onclick = async () => {
    const n = await promptBox('Neue Datei', 'Name', 'neu.txt'); if (!n) return;
    openEditor(id, join(n), load, true);
  };
  const upload = async files => {
    if (!files.length) return;
    const fd = new FormData();
    let total = 0;
    for (const f of files) { fd.append('f', f, f.webkitRelativePath || f.relpath || f.name); total += f.size; }
    const t = document.createElement('div'); t.className = 'toast'; $('#toasts').append(t);
    const xhr = new XMLHttpRequest();
    xhr.open('POST', `/api/servers/${id}/files/upload?path=${encodeURIComponent(path)}`);
    xhr.setRequestHeader('X-Serverwerk', '1');
    xhr.upload.onprogress = e => { t.textContent = `Lädt hoch ... ${Math.round(e.loaded / (e.total || total) * 100)} %`; };
    xhr.onload = () => { t.remove(); if (xhr.status < 300) { toast(`${files.length} Datei(en) hochgeladen`, 'good'); load(); } else fail(new Error(JSON.parse(xhr.responseText || '{}').error || 'Upload fehlgeschlagen')); };
    xhr.onerror = () => { t.remove(); fail(new Error('Upload abgebrochen')); };
    xhr.send(fd);
  };
  $('#f-up').onclick = () => $('#f-input').click();
  $('#f-input').onchange = e => { upload([...e.target.files]); e.target.value = ''; };
  // Dateien irgendwo auf die Seite ziehen: ganzes Fenster wird Ablage.
  const ov = document.createElement('div');
  ov.className = 'dropzone'; ov.hidden = true;
  document.body.append(ov);
  let depth = 0;
  const isFiles = e => [...(e.dataTransfer?.types || [])].includes('Files');
  const onEnter = e => { if (!isFiles(e)) return; e.preventDefault(); depth++; ov.innerHTML = `<div>${I('upload')}<b>Loslassen zum Hochladen</b><span class="mono">${esc('/home/container' + (path === '/' ? '' : path))}</span></div>`; ov.hidden = false; };
  const onOver = e => { if (isFiles(e)) e.preventDefault(); };
  const onLeaveDrag = e => { if (!isFiles(e)) return; if (--depth <= 0) { depth = 0; ov.hidden = true; } };
  const onDrop = async e => {
    if (!isFiles(e)) return;
    e.preventDefault(); depth = 0; ov.hidden = true;
    const items = [...(e.dataTransfer.items || [])].map(i => i.webkitGetAsEntry?.()).filter(Boolean);
    if (!items.length) return upload([...e.dataTransfer.files]);
    const files = [];
    const walk = (entry, pre) => new Promise(res => {
      if (entry.isFile) entry.file(f => { f.relpath = pre + f.name; files.push(f); res(); }, res);
      else {
        const r = entry.createReader(); const all = [];
        const more = () => r.readEntries(async es => { if (!es.length) { for (const x of all) await walk(x, pre + entry.name + '/'); res(); } else { all.push(...es); more(); } }, res);
        more();
      }
    });
    for (const it of items) await walk(it, '');
    upload(files);
  };
  window.addEventListener('dragenter', onEnter);
  window.addEventListener('dragover', onOver);
  window.addEventListener('dragleave', onLeaveDrag);
  window.addEventListener('drop', onDrop);
  onLeave(() => {
    window.removeEventListener('dragenter', onEnter); window.removeEventListener('dragover', onOver);
    window.removeEventListener('dragleave', onLeaveDrag); window.removeEventListener('drop', onDrop); ov.remove();
  });
  load();
}

async function openEditor(id, path, after, isNew = false) {
  let text = '';
  if (!isNew) {
    try { text = await (await api('GET', `/servers/${id}/files/content?path=${encodeURIComponent(path)}`, undefined, { raw: true })).text(); }
    catch (e) { return fail(e); }
  }
  const crlf = text.includes('\r\n');
  if (crlf) text = text.replace(/\r\n/g, '\n');
  const ed = document.createElement('div');
  ed.className = 'editor';
  ed.innerHTML = `<div class="eh"><button class="ib" data-x title="Schließen">${I('back')}</button>
    <span class="mono grow ell">${esc(path)}</span><span class="faint" id="e-state" style="font-size:13px"></span>
    <button class="btn primary" data-save>${I('check')}Speichern</button></div>
    <div class="eb"><div class="gut"></div><textarea spellcheck="false" autocapitalize="off"></textarea></div>`;
  document.body.append(ed);
  const ta = $('textarea', ed), gut = $('.gut', ed);
  ta.value = text;
  let dirty = false;
  const lines = () => { const n = ta.value.split('\n').length; gut.textContent = Array.from({ length: n }, (_, i) => i + 1).join('\n'); };
  lines();
  ta.oninput = () => { dirty = true; $('#e-state').textContent = 'ungespeichert'; lines(); };
  ta.onscroll = () => { gut.scrollTop = ta.scrollTop; };
  ta.onkeydown = e => {
    if (e.key === 'Tab') { e.preventDefault(); const s = ta.selectionStart; ta.setRangeText('    ', s, ta.selectionEnd, 'end'); ta.oninput(); }
  };
  const save = async () => {
    try {
      await api('PUT', `/servers/${id}/files/content?path=${encodeURIComponent(path)}`, crlf ? ta.value.replace(/\n/g, '\r\n') : ta.value);
      dirty = false; $('#e-state').textContent = 'gespeichert'; toast('Gespeichert', 'good');
    } catch (e) { fail(e); }
  };
  const key = e => { if ((e.ctrlKey || e.metaKey) && e.key === 's') { e.preventDefault(); save(); } if (e.key === 'Escape') close(); };
  const close = async () => {
    if (dirty && !await confirmBox('Änderungen verwerfen?', 'Die Datei hat ungespeicherte Änderungen.', { ok: 'Verwerfen', danger: true })) return;
    document.removeEventListener('keydown', key); ed.remove(); after?.();
  };
  document.addEventListener('keydown', key);
  $('[data-save]', ed).onclick = save;
  $('[data-x]', ed).onclick = close;
  ta.focus();
}

/* ---------- Backups ---------- */
function tabBackups(c) {
  const { id } = c;
  c.el.innerHTML = `<div class="panel"><div class="ph"><h2 class="grow">Backups</h2><span class="muted" id="bmax" style="font-size:13px"></span>
    <button class="btn primary small" id="bnew">${I('plus')}Backup erstellen</button></div><div id="blist"></div></div>
    <p class="term-label" style="margin-top:12px">${tt('Dateien ausschließen: Muster zeilenweise in <b>.swignore</b> (oder .pteroignore) im Serverordner.', 'Exclude files: one pattern per line in <b>.swignore</b> (or .pteroignore) in the server folder.')}</p>`;
  const load = async () => {
    const r = await api('GET', `/servers/${id}/backups`).catch(fail); if (!r) return;
    $('#bmax').textContent = `${r.backups.length} von ${r.max || 5}, ältere werden automatisch gelöscht`;
    $('#blist').innerHTML = r.backups.length ? `<table class="table"><thead><tr><th>Erstellt</th><th>Notiz</th><th class="r">Größe</th><th></th></tr></thead><tbody>${r.backups.map(b => `
      <tr><td>${when(b.created)}</td><td class="muted">${esc(b.note)}</td><td class="r">${bytes(b.size)}</td>
      <td class="r"><button class="ib" data-m="${esc(b.name)}">${I('dots')}</button></td></tr>`).join('')}</tbody></table>`
      : `<div class="empty" style="margin:14px;border:0"><span class="term-label">$ ls backups/<br>(leer)</span>Noch keine Backups.</div>`;
    $$('[data-m]').forEach(b => b.onclick = () => {
      const n = b.dataset.m;
      popMenu(b, [
        { label: 'Herunterladen', icon: 'download', run: () => location.href = `/api/servers/${id}/backups/${encodeURIComponent(n)}` },
        { label: 'Wiederherstellen', icon: 'restart', run: async () => { if (await confirmBox('Backup einspielen?', 'Alle aktuellen Dateien des Servers werden ersetzt. Der Server muss aus sein.', { ok: 'Einspielen', danger: true })) api('POST', `/servers/${id}/backups/${encodeURIComponent(n)}/restore`).then(() => toast('Backup eingespielt', 'good')).catch(fail); } },
        { label: 'Löschen', icon: 'trash', danger: true, run: async () => { if (await confirmBox('Backup löschen?', n, { ok: 'Löschen', danger: true })) api('DELETE', `/servers/${id}/backups/${encodeURIComponent(n)}`).then(load).catch(fail); } },
      ]);
    });
  };
  $('#bnew').onclick = async () => {
    const note = await promptBox('Backup erstellen', 'Notiz (optional)', '', { ok: 'Erstellen' });
    if (note === null) return;
    api('POST', `/servers/${id}/backups`, { note }).then(() => { toast('Backup läuft, Fortschritt in der Konsole'); setTimeout(load, 2500); }).catch(fail);
  };
  load();
  const t = setInterval(load, 8000);
  onLeave(() => clearInterval(t));
}

/* ---------- Zeitpläne ---------- */
const CRON_PRESETS = [['0 4 * * *', 'Täglich 4 Uhr'], ['0 */6 * * *', 'Alle 6 Stunden'], ['0 * * * *', 'Stündlich'], ['*/30 * * * *', 'Alle 30 Minuten'], ['0 5 * * 1', 'Montags 5 Uhr']];
function tabSchedules(c) {
  const { id } = c;
  let list = JSON.parse(JSON.stringify(c.s.schedules || []));
  const draw = () => {
    c.el.innerHTML = `<div class="head" style="margin-bottom:14px"><div class="sub term-label">$ crontab -l &nbsp; Minute Stunde Tag Monat Wochentag</div>
      <div class="tools"><button class="btn" id="sadd">${I('plus')}Zeitplan</button><button class="btn primary" id="ssave">${I('check')}Speichern</button></div></div>
      <div class="stack" id="slist">${list.length ? '' : `<div class="empty"><span class="term-label">$ crontab -l<br>(leer)</span>Zum Beispiel: täglicher Neustart oder ein Backup jede Nacht.</div>`}</div>`;
    const box = $('#slist');
    list.forEach((sc, i) => {
      const d = document.createElement('div');
      d.className = 'sched';
      d.innerHTML = `<div class="row wrap">
          <input class="input grow" style="max-width:280px" data-k="name" value="${esc(sc.name)}" placeholder="Name">
          <input class="input mono" style="max-width:170px" data-k="cron" value="${esc(sc.cron)}" placeholder="0 4 * * *">
          <select class="input" style="max-width:170px" data-preset><option value="">Vorlage ...</option>${CRON_PRESETS.map(p => `<option value="${p[0]}">${p[1]}</option>`).join('')}</select>
          <label class="check"><span class="switch"><input type="checkbox" data-k="enabled" ${sc.enabled ? 'checked' : ''}><i></i></span>aktiv</label>
          <label class="check"><input type="checkbox" data-k="onlyIfOn" ${sc.onlyIfOn ? 'checked' : ''}>nur wenn an</label>
          <span class="grow"></span>
          ${sc.id ? `<button class="btn small" data-run>Jetzt ausführen</button>` : ''}
          <button class="ib danger" data-del title="Zeitplan löschen">${I('trash')}</button></div>
        <div class="stack" style="gap:8px">${sc.tasks.map((t, j) => `<div class="task" data-j="${j}">
          <select class="input" data-tk="action"><option value="command" ${t.action === 'command' ? 'selected' : ''}>Befehl</option><option value="power" ${t.action === 'power' ? 'selected' : ''}>Strom</option><option value="backup" ${t.action === 'backup' ? 'selected' : ''}>Backup</option></select>
          <label class="row" style="gap:6px"><input class="input" type="number" min="0" data-tk="delay" value="${t.delay || 0}" style="width:80px"><span class="muted" style="font-size:13px">s warten</span></label>
          ${t.action === 'power' ? `<select class="input" data-tk="payload">${['start', 'stop', 'restart', 'kill'].map(p => `<option value="${p}" ${t.payload === p ? 'selected' : ''}>${{ start: 'Starten', stop: 'Stoppen', restart: 'Neustarten', kill: 'Beenden' }[p]}</option>`).join('')}</select>`
          : t.action === 'backup' ? `<span class="muted" style="font-size:13px">erstellt ein Backup</span>` : `<input class="input mono" data-tk="payload" value="${esc(t.payload)}" placeholder="say Neustart in 5 Minuten">`}
          <button class="ib" data-tdel title="Aufgabe entfernen">${I('kill')}</button></div>`).join('')}
          <div><button class="btn small ghost" data-tadd>${I('plus')}Aufgabe</button></div></div>
        <div class="faint" style="font-size:12.5px">${sc.lastRun ? 'Zuletzt: ' + when(sc.lastRun) : 'Noch nie gelaufen'}</div>`;
      $$('[data-k]', d).forEach(x => x.onchange = x.oninput = () => { sc[x.dataset.k] = x.type === 'checkbox' ? x.checked : x.value; });
      $('[data-preset]', d).onchange = e => { if (e.target.value) { sc.cron = e.target.value; $('[data-k=cron]', d).value = sc.cron; } };
      $$('.task', d).forEach(row => {
        const t = sc.tasks[row.dataset.j];
        $$('[data-tk]', row).forEach(x => x.onchange = x.oninput = () => {
          const k = x.dataset.tk;
          t[k] = k === 'delay' ? +x.value : x.value;
          if (k === 'action') { t.payload = t.action === 'power' ? 'restart' : ''; draw(); }
        });
        $('[data-tdel]', row).onclick = () => { sc.tasks.splice(row.dataset.j, 1); draw(); };
      });
      $('[data-tadd]', d).onclick = () => { sc.tasks.push({ action: 'command', payload: '', delay: 0 }); draw(); };
      $('[data-del]', d).onclick = () => { list.splice(i, 1); draw(); };
      const run = $('[data-run]', d);
      if (run) run.onclick = () => api('POST', `/servers/${id}/schedules/${sc.id}/run`).then(() => toast('Läuft, siehe Konsole')).catch(fail);
      box.append(d);
    });
    $('#sadd').onclick = () => { list.push({ name: 'Täglicher Neustart', cron: '0 4 * * *', enabled: true, onlyIfOn: true, tasks: [{ action: 'command', payload: 'say Neustart in 1 Minute', delay: 0 }, { action: 'power', payload: 'restart', delay: 60 }] }); draw(); };
    $('#ssave').onclick = () => api('PUT', `/servers/${id}/schedules`, list).then(r => { list = r; c.s.schedules = r; toast('Gespeichert', 'good'); draw(); }).catch(fail);
  };
  draw();
}

/* ---------- Startparameter ---------- */
function tabStartup(c) {
  const { id, s, d } = c;
  const a = S.me.admin;
  const vars = d.variables || [];
  const preview = () => {
    const env = { SERVER_MEMORY: s.memory, SERVER_PORT: s.ports[0], SERVER_IP: '0.0.0.0' };
    $$('[data-env]').forEach(x => env[x.dataset.env] = x.value);
    vars.forEach(v => { if (env[v.env] === undefined) env[v.env] = v.value; });
    const cmd = ($('#startup')?.value || s.startup || d.eggStartup);
    $('#prev').textContent = cmd.replace(/\{\{\s*([A-Za-z0-9_.]+)\s*\}\}/g, (m, k) => env[k] ?? env[k.replace(/^env\./, '')] ?? m).replace(/\$\{?([A-Z0-9_]+)\}?/g, (m, k) => env[k] ?? m);
  };
  c.el.innerHTML = `<div class="stack">
    <div class="panel"><div class="ph"><h2>Startbefehl</h2></div><div class="pb stack">
      ${a ? `<label class="f">Befehl (leer lassen = wie im Egg)<input class="input mono" id="startup" value="${esc(s.startup || d.eggStartup)}"></label>` : ''}
      <div><div class="term-label" style="margin-bottom:6px">$ so startet der Server</div><div class="mono" id="prev" style="padding:12px 14px;background:var(--term);color:#d9d9de;border-radius:10px;word-break:break-all"></div></div>
      <label class="f">Docker-Image<select class="input" id="image">${(d.images || []).map(im => `<option value="${esc(im.image)}" ${im.image === s.image ? 'selected' : ''}>${esc(im.name)} – ${esc(im.image)}</option>`).join('')}
        ${!(d.images || []).find(im => im.image === s.image) ? `<option value="${esc(s.image)}" selected>${esc(s.image)}</option>` : ''}</select>
        <small>Für Minecraft: neuere Versionen brauchen ein neueres Java.</small></label>
    </div></div>
    <div class="panel"><div class="ph"><h2>Variablen</h2></div><div class="pb grid2">
      ${vars.map(v => {
        const opts = v.options || (/(^|\|)boolean(\||$)/.test(v.rules) ? ['0', '1'] : null);
        const input = opts ? `<select class="input" data-env="${esc(v.env)}" ${v.editable ? '' : 'disabled'}>${opts.map(o => `<option ${o === v.value ? 'selected' : ''}>${esc(o)}</option>`).join('')}</select>`
          : `<input class="input" data-env="${esc(v.env)}" value="${esc(v.value)}" ${v.editable ? '' : 'readonly'}>`;
        return `<label class="f">${esc(v.name)}${input}<small>${esc(v.description).slice(0, 220)}</small><small class="mono">${esc(v.env)}</small></label>`;
      }).join('') || '<div class="muted">Dieses Egg hat keine Variablen.</div>'}
    </div></div>
    <div><button class="btn primary" id="save">${I('check')}Speichern</button> <span class="muted" style="font-size:13px;margin-left:8px">Änderungen gelten beim nächsten Start.</span></div></div>`;
  $$('[data-env], #startup').forEach(x => x.oninput = preview);
  preview();
  $('#save').onclick = () => {
    const env = {};
    $$('[data-env]').forEach(x => env[x.dataset.env] = x.value);
    const body = { env, image: $('#image').value };
    if (a) body.startup = $('#startup').value;
    api('PUT', `/servers/${id}/startup`, body).then(() => toast('Gespeichert', 'good')).catch(fail);
  };
}

/* ---------- Unterbenutzer ---------- */
const PERM_NAMES = { console: 'Konsole', power: 'Starten/Stoppen', files: 'Dateien', backups: 'Backups', schedules: 'Zeitpläne', startup: 'Startparameter', settings: 'Einstellungen' };
async function tabSubusers(c) {
  const { id, s } = c;
  const names = await api('GET', '/usernames').catch(fail) || [];
  let subs = JSON.parse(JSON.stringify(s.subusers || []));
  const nameOf = uid => names.find(n => n.id === uid)?.name || uid;
  const draw = () => {
    const free = names.filter(n => n.id !== s.owner && !subs.find(x => x.userId === n.id || x.user === n.id));
    c.el.innerHTML = `<div class="panel"><div class="ph"><h2 class="grow">Wer darf auf diesen Server?</h2>
      ${free.length ? `<select class="input" id="addu" style="max-width:200px;height:34px"><option value="">Benutzer hinzufügen ...</option>${free.map(n => `<option value="${n.id}">${esc(n.name)}</option>`).join('')}</select>` : ''}</div>
      <div class="pb stack"><div class="row"><span class="avatar">${esc(nameOf(s.owner)[0])}</span><b>${esc(nameOf(s.owner))}</b><span class="tag acc">Besitzer, darf alles</span></div>
      ${subs.map((su, i) => `<div class="stack" style="gap:8px;padding-top:12px;border-top:1px solid var(--border)">
        <div class="row"><span class="avatar">${esc(nameOf(su.user)[0])}</span><b class="grow">${esc(nameOf(su.user))}</b><button class="ib danger" data-rm="${i}">${I('trash')}</button></div>
        <div class="perm-grid">${Object.entries(PERM_NAMES).map(([p, n]) => `<label class="check"><input type="checkbox" data-i="${i}" data-p="${p}" ${su.perms?.includes(p) ? 'checked' : ''}>${n}</label>`).join('')}</div></div>`).join('')}
      ${names.length < 2 ? `<p class="muted" style="margin:0">${tt('Lege zuerst unter <a href="#/benutzer">Benutzer</a> weitere Konten an.', 'First create more accounts under <a href="#/benutzer">Users</a>.')}</p>` : ''}
      <div><button class="btn primary" id="save">${I('check')}Speichern</button></div></div></div>`;
    const add = $('#addu');
    if (add) add.onchange = () => { if (add.value) { subs.push({ user: add.value, perms: ['console', 'power'] }); draw(); } };
    $$('[data-rm]').forEach(b => b.onclick = () => { subs.splice(+b.dataset.rm, 1); draw(); });
    $$('[data-p]').forEach(x => x.onchange = () => {
      const su = subs[x.dataset.i]; su.perms = su.perms || [];
      x.checked ? su.perms.push(x.dataset.p) : su.perms = su.perms.filter(p => p !== x.dataset.p);
    });
    $('#save').onclick = () => api('PUT', `/servers/${id}/subusers`, subs).then(() => { s.subusers = subs; toast('Gespeichert', 'good'); }).catch(fail);
  };
  draw();
}

/* ---------- Einstellungen ---------- */
async function tabSettings(c) {
  const { id, s } = c;
  const a = S.me.admin;
  const users = a ? await api('GET', '/usernames').catch(() => []) : [];
  c.el.innerHTML = `<div class="stack">
    <div class="panel"><div class="ph"><h2>Allgemein</h2></div><form class="pb stack" id="gen">
      <label class="f">Name<input class="input" name="name" value="${esc(s.name)}" required></label>
      <label class="f">Beschreibung<input class="input" name="description" value="${esc(s.description)}"></label>
      <label class="check"><span class="switch"><input type="checkbox" name="autoRestart" ${s.autoRestart ? 'checked' : ''}><i></i></span>Nach Absturz automatisch neu starten</label>
      <label class="check"><span class="switch"><input type="checkbox" name="bootStart" ${s.noBootStart ? '' : 'checked'}><i></i></span>Nach einem Neustart des Rechners wieder starten, wenn er vorher lief</label>
      ${a ? `<div class="grid2">
        <label class="f">Arbeitsspeicher (MB)<input class="input" type="number" min="0" name="memory" value="${s.memory}"><small>0 = unbegrenzt</small></label>
        <label class="f">CPU (% eines Kerns)<input class="input" type="number" min="0" name="cpu" value="${s.cpu}"><small>100 = ein Kern, 0 = unbegrenzt</small></label>
        <label class="f">Speicherplatz (MB)<input class="input" type="number" min="0" name="disk" value="${s.disk}"><small>0 = unbegrenzt</small></label>
        <label class="f">Ports<input class="input mono" name="ports" value="${s.ports.join(', ')}"><small>Erster Port ist der Hauptport. TCP und UDP.</small></label>
        <label class="f">Besitzer<select class="input" name="owner">${users.map(u => `<option value="${u.id}" ${u.id === s.owner ? 'selected' : ''}>${esc(u.name)}</option>`).join('')}</select></label>
        <label class="f">Backups behalten<input class="input" type="number" min="1" name="maxBackups" value="${s.maxBackups || 5}"></label>
      </div>` : ''}
      <div><button class="btn primary">${I('check')}Speichern</button></div></form></div>
    ${a ? `<div class="panel"><div class="ph"><h2>Domain</h2>${s.domain ? `<span class="tag ${s.domain.status === 'ok' ? 'acc' : ''}">${s.domain.status === 'ok' ? 'aktiv' : 'Fehler'}</span>` : ''}</div><div class="pb stack" id="dom"><div class="muted">Lädt ...</div></div></div>` : ''}
    <div class="panel"><div class="ph"><h2>Neu installieren</h2></div><div class="pb row wrap"><p class="muted grow" style="margin:0">Führt das Installationsskript des Eggs erneut aus (z. B. für eine neue Version). Deine Dateien bleiben, werden aber eventuell überschrieben.</p>
      <button class="btn" id="reinst">${I('restart')}Neu installieren</button></div></div>
    ${a ? `<div class="panel"><div class="ph"><h2 style="color:var(--danger)">Server löschen</h2></div><div class="pb row wrap"><p class="muted grow" style="margin:0">Löscht den Server mit allen Dateien und Backups.</p>
      <button class="btn danger" id="del">${I('trash')}Löschen</button></div></div>
    <div class="panel"><div class="ph"><h2>Technik</h2></div><dl class="pb kv"><dt>Server-ID</dt><dd class="mono">${esc(s.id)}</dd><dt>Container</dt><dd class="mono">serverwerk-${esc(s.id.slice(0, 8))}</dd><dt>Ordner</dt><dd class="mono">/var/lib/serverwerk/server/${esc(s.id)}</dd><dt>Erstellt</dt><dd>${when(s.created)}</dd></dl></div>` : ''}
  </div>`;
  const dom = $('#dom');
  if (dom) api('GET', '/settings').then(set => {
    if (!set.cfConnected || !(set.cfZones || []).length) {
      dom.innerHTML = `<p class="muted" style="margin:0">${tt('Verbinde zuerst Cloudflare unter <a href="#/einstellungen/cloudflare">Einstellungen</a>, dann kannst du hier eine Adresse wie <span class="mono">minecraft.deinedomain.de</span> vergeben.', 'Connect Cloudflare under <a href="#/einstellungen/cloudflare">Settings</a> first, then you can assign an address like <span class="mono">minecraft.example.com</span> here.')}</p>`;
      return;
    }
    const d = s.domain || {};
    const isMC = !!c.d.addons;
    dom.innerHTML = `<div class="row wrap" style="gap:8px;align-items:center">
        <input class="input mono" id="d-pre" value="${esc(d.prefix && d.prefix !== '@' ? d.prefix : '')}" placeholder="${isMC ? 'minecraft' : 'name'}" style="max-width:220px">
        <span class="mono muted">.</span>
        <select class="input" id="d-zone" style="max-width:260px">${set.cfZones.map(z => `<option value="${z.id}" ${z.id === d.zoneId ? 'selected' : ''}>${esc(z.name)}</option>`).join('')}</select>
        <button class="btn primary" id="d-save">${I('check')}${s.domain ? 'Ändern' : 'Vergeben'}</button>
        ${s.domain ? `<button class="btn danger" id="d-rm">${I('trash')}Entfernen</button>` : ''}</div>
      ${s.domain ? `<div class="banner ${d.status && d.status !== 'ok' ? 'bad' : ''}" style="margin:0"><span class="grow">${d.status && d.status !== 'ok' ? esc(d.status) : tt(`Spieler verbinden sich mit <b class="mono">${esc(serverAddr(s))}</b>${d.srv ? ' (ohne Port, per SRV-Eintrag)' : ''}.`, `Players connect with <b class="mono">${esc(serverAddr(s))}</b>${d.srv ? ' (no port needed, via SRV record)' : ''}.`)}</span><button class="btn small" id="d-copy">${I('copy')}Kopieren</button></div>` : ''}
      <p class="faint" style="margin:0;font-size:12.5px">${isMC ? tt('Für Minecraft Java legt Serverwerk zusätzlich einen SRV-Eintrag an, damit kein Port nötig ist. ', 'For Minecraft Java, Serverwerk also creates an SRV record so no port is needed. ') : ''}${tt(`Leer lassen = die Domain selbst. Der Port ${s.ports[0]} muss im Router freigegeben sein.`, `Leave empty = the domain itself. Port ${s.ports[0]} must be forwarded in your router.`)}</p>`;
    $('#d-save').onclick = ev => {
      const b = ev.currentTarget; b.disabled = true;
      api('POST', `/servers/${id}/domain`, { zoneId: $('#d-zone').value, prefix: $('#d-pre').value })
        .then(r => { toast(`${r.prefix === '@' ? r.zoneName : r.prefix + '.' + r.zoneName} zeigt jetzt auf diesen Server`, 'good'); render(); })
        .catch(e => { fail(e); b.disabled = false; });
    };
    const rm = $('#d-rm');
    if (rm) rm.onclick = async () => {
      if (!await confirmBox('Domain entfernen?', 'Die DNS-Einträge werden bei Cloudflare gelöscht.', { ok: 'Entfernen', danger: true })) return;
      api('POST', `/servers/${id}/domain`, { remove: true }).then(() => { toast('Domain entfernt'); render(); }).catch(fail);
    };
    const cp = $('#d-copy'); if (cp) cp.onclick = () => copy(serverAddr(s));
  }).catch(() => { dom.innerHTML = '<span class="muted">Einstellungen nicht ladbar.</span>'; });
  $('#gen').onsubmit = e => {
    e.preventDefault();
    const f = e.target;
    const body = { name: f.name.value, description: f.description.value, autoRestart: f.autoRestart.checked, bootStart: f.bootStart.checked };
    if (a) Object.assign(body, {
      memory: +f.memory.value, cpu: +f.cpu.value, disk: +f.disk.value, owner: f.owner.value, maxBackups: +f.maxBackups.value,
      ports: f.ports.value.split(/[\s,;]+/).filter(Boolean).map(Number),
    });
    api('PATCH', '/servers/' + id, body).then(() => { toast('Gespeichert. Grenzen und Ports gelten ab dem nächsten Start.', 'good'); render(); }).catch(fail);
  };
  $('#reinst').onclick = async () => {
    if (!await confirmBox('Neu installieren?', 'Der Server muss aus sein. Das Installationsskript läuft erneut.', { ok: 'Installieren' })) return;
    api('POST', `/servers/${id}/reinstall`).then(() => go(`#/s/${id}/konsole`)).catch(fail);
  };
  const del = $('#del');
  if (del) del.onclick = async () => {
    const n = await promptBox('Server löschen', `Zum Bestätigen den Namen eintippen: ${s.name}`, '', { ok: 'Endgültig löschen' });
    if (n === null) return;
    if (n !== s.name) return toast('Name stimmt nicht', 'bad');
    api('DELETE', '/servers/' + id).then(() => { toast('Server gelöscht'); go('#/'); }).catch(fail);
  };
}

/* ---------- Neuer Server ---------- */
async function pageCreate(page) {
  const [eggsR, users, pinfo, set] = await Promise.all([api('GET', '/eggs'), api('GET', '/usernames'), api('GET', '/ports/free'), api('GET', '/settings')]).catch(e => { fail(e); return [{ eggs: [] }, [], {}, {}]; });
  const eggs = (eggsR.eggs || []).filter(e => e.compatible !== false);
  const hiddenEggs = (eggsR.eggs || []).length - eggs.length;
  if (!pinfo.configured) {
    page.innerHTML = `<div class="head"><div><h1>Neuer Server</h1></div></div>` + `<div class="empty"><span class="term-label">$ cat /etc/serverwerk/ports<br>(leer)</span><b style="display:block;color:var(--text);margin-bottom:6px">Erst Ports festlegen</b>${tt('Trag unter Allocations ein, welche Ports in deinem Router freigegeben sind (z. B. <span class="mono">25565-25575</span>). Serverwerk vergibt dann automatisch freie Ports daraus.', 'Enter under Allocations which ports are forwarded in your router (e.g. <span class="mono">25565-25575</span>). Serverwerk then assigns free ports from them automatically.')}<div style="margin-top:16px"><a class="btn primary" href="#/allocations">${I('plug')}Zu den Allocations</a></div></div>`;
    return;
  }
  if (!pinfo.free.length) {
    page.innerHTML = `<div class="head"><div><h1>Neuer Server</h1></div></div><div class="empty"><b style="display:block;color:var(--text);margin-bottom:6px">Alle Ports sind belegt</b>${tt(`Alle ${pinfo.total} Ports aus deiner Liste sind vergeben. Erweitere die Liste unter Allocations.`, `All ${pinfo.total} ports from your list are taken. Extend the list under Allocations.`)}<div style="margin-top:16px"><a class="btn primary" href="#/allocations">${I('plug')}Zu den Allocations</a></div></div>`;
    return;
  }
  const ports = pinfo.free;
  let egg = null, q = '';
  const step1 = () => {
    page.innerHTML = `<div class="head"><div><h1>Neuer Server</h1><div class="sub steps"><b>1 Egg wählen</b><span>2 Einstellungen</span></div></div>
      <div class="tools"><div class="row" style="position:relative"><input class="input" id="q" placeholder="Suchen ..." style="width:240px;height:38px" value="${esc(q)}"></div><a class="btn" href="#/eggs/katalog">${I('plus')}Eggs holen</a></div></div>
      ${hiddenEggs ? `<p class="faint" style="margin:-10px 0 16px;font-size:13px">${tt(`${hiddenEggs} installierte Eggs sind ausgeblendet, weil sie nicht auf diesem Rechner (${esc(archName(eggsR.arch))}) laufen.`, `${hiddenEggs} installed eggs are hidden because they don't run on this machine (${esc(archName(eggsR.arch))}).`)}</p>` : ''}<div id="eggl"></div>`;
    const draw = () => {
      const f = eggs.filter(e => !q || (e.name + ' ' + e.category + ' ' + e.description).toLowerCase().includes(q.toLowerCase()));
      if (!eggs.length) { $('#eggl').innerHTML = `<div class="empty"><span class="term-label">$ ls /eggs<br>(leer)</span>${tt('Noch keine Eggs installiert. <a href="#/eggs/katalog">Zum Katalog</a> – dort z. B. Paper oder Vanilla für Minecraft holen.', 'No eggs installed yet. <a href="#/eggs/katalog">Go to the catalog</a> – get e.g. Paper or Vanilla for Minecraft there.')}</div>`; return; }
      const cats = [...new Set(f.map(e => e.category))];
      $('#eggl').innerHTML = cats.map(c => `<div class="cat-h">${esc(c)}</div><div class="eggs">${f.filter(e => e.category === c).map(e => `<button class="egg" data-id="${e.id}">${thumb(eggIcon(e.id), e.name, 'banner')}<div class="eb"><b>${esc(e.name)}</b><p>${esc(e.description)}</p></div></button>`).join('')}</div>`).join('') || '<div class="muted">Nichts gefunden.</div>';
      $$('[data-id]').forEach(b => b.onclick = () => { egg = eggs.find(e => e.id === b.dataset.id); step2(); });
    };
    $('#q').oninput = e => { q = e.target.value; draw(); };
    draw();
  };
  const step2 = () => {
    const isMC = /minecraft|paper|spigot|purpur|fabric|forge|velocity|bungee|vanilla|neoforge|folia/i.test(egg.name + egg.category);
    const defPort = ports[0];
    page.innerHTML = `<div class="head"><div class="row" style="gap:16px">${thumb(eggIcon(egg.id), egg.name, 'sq big')}<div><h1>${esc(egg.name)}</h1><div class="sub steps"><span>1 Egg wählen</span><b>2 Einstellungen</b></div></div></div>
      <div class="tools"><button class="btn" id="back">${I('back')}Anderes Egg</button></div></div>
      <form class="stack" id="cf">
      <div class="panel"><div class="ph"><h2>Server</h2></div><div class="pb grid2">
        <label class="f">Name<input class="input" name="name" required placeholder="z. B. Survival"></label>
        <label class="f">Besitzer<select class="input" name="owner">${users.map(u => `<option value="${u.id}" ${u.id === S.me.id ? 'selected' : ''}>${esc(u.name)}</option>`).join('')}</select></label>
        <label class="f">Arbeitsspeicher (MB)<input class="input" type="number" min="0" name="memory" value="${isMC && (set.defaults?.memory ?? 2048) > 0 ? Math.max(set.defaults?.memory ?? 2048, 4096) : (set.defaults?.memory ?? 2048)}"><small>0 = unbegrenzt</small></label>
        <label class="f">CPU (% eines Kerns)<input class="input" type="number" min="0" name="cpu" value="${set.defaults?.cpu ?? 0}"><small>0 = unbegrenzt, 200 = zwei Kerne</small></label>
        <label class="f">Speicherplatz (MB)<input class="input" type="number" min="0" name="disk" value="${set.defaults?.disk ?? 0}"><small>0 = unbegrenzt</small></label>
        <label class="f">Port(s)<input class="input mono" name="ports" value="${defPort}"><small>Automatisch gewählt. Weitere freie aus deiner Liste: ${ports.slice(1).join(', ') || 'keine'}</small></label>
        <label class="f">Docker-Image<select class="input" name="image">${egg.images.map(i => `<option value="${esc(i.image)}">${esc(i.name)}</option>`).join('')}</select></label>
      </div></div>
      <div class="panel"><div class="ph"><h2>Variablen</h2></div><div class="pb grid2">
        ${(() => {
          const field = v => {
            const opts = (v.rules.match(/(?:^|\|)in:([^|]+)/) || [])[1]?.split(',') || (/(^|\|)boolean(\||$)/.test(v.rules) ? ['0', '1'] : null);
            const ph = v.env === 'USER_AGENT' ? 'wird automatisch ausgefüllt' : '';
            return `<label class="f">${esc(v.name)}${opts ? `<select class="input" data-env="${esc(v.env)}">${opts.map(o => `<option ${o === v.default ? 'selected' : ''}>${esc(o)}</option>`).join('')}</select>` : `<input class="input" data-env="${esc(v.env)}" value="${esc(v.default)}" placeholder="${ph}">`}<small>${esc(v.description).slice(0, 200)}</small></label>`;
          };
          const vars = egg.variables || [];
          const main = vars.filter(v => v.userViewable && v.userEditable), extra = vars.filter(v => !(v.userViewable && v.userEditable));
          if (!vars.length) return '<div class="muted">Keine Variablen.</div>';
          return main.map(field).join('') + (extra.length ? `<details class="more" style="grid-column:1/-1"><summary>Erweitert (${extra.length})</summary><div class="grid2" style="margin-top:14px">${extra.map(field).join('')}</div></details>` : '');
        })()}
      </div></div>
      ${set.cfConnected && (set.cfZones || []).length ? `<div class="panel"><div class="ph"><h2>Domain</h2><span class="faint" style="font-size:13px">optional</span></div><div class="pb stack">
        <div class="row wrap" style="gap:8px;align-items:center">
          <input class="input mono" name="dpre" placeholder="${isMC ? 'minecraft' : 'name'}" style="max-width:220px">
          <span class="mono muted">.</span>
          <select class="input" name="dzone" style="max-width:260px">${set.autoDomain ? `<option value="">${tt('automatisch', 'automatic')} (${esc(['name', set.autoDomain.sub, set.autoDomain.zoneName].filter(Boolean).join('.'))})</option><option value="none">${tt('keine Domain', 'no domain')}</option>` : `<option value="">${tt('keine Domain', 'no domain')}</option>`}${set.cfZones.map((z, i) => `<option value="${z.id}">${esc(z.name)}</option>`).join('')}</select></div>
        <small class="faint" style="font-size:12.5px">${isMC ? 'Spieler verbinden sich dann ohne Port, z. B. mit minecraft.deinedomain.de. ' : ''}Kannst du auch später unter Einstellungen des Servers ändern.</small></div></div>` : ''}
      <div class="row wrap"><button class="btn primary big">${I('check')}Server anlegen</button>
        <label class="check"><input type="checkbox" name="start" ${set.defaults?.startAfter !== false ? 'checked' : ''}>nach der Installation starten</label></div>
      </form>`;
    $('#back').onclick = step1;
    $('#cf').onsubmit = async e => {
      e.preventDefault();
      const f = e.target, env = {};
      $$('[data-env]').forEach(x => env[x.dataset.env] = x.value);
      const btn = $('button.primary', f); btn.disabled = true;
      try {
        const srv = await api('POST', '/servers', {
          name: f.name.value, owner: f.owner.value, egg: egg.id, memory: +f.memory.value, cpu: +f.cpu.value, disk: +f.disk.value,
          ports: f.ports.value.split(/[\s,;]+/).filter(Boolean).map(Number), image: f.image.value, env, startAfter: f.start.checked, noAutoDomain: !!f.dzone?.value,
        });
        toast('Server angelegt, Installation läuft', 'good');
        if (f.dzone?.value && f.dzone.value !== 'none') {
          await api('POST', `/servers/${srv.id}/domain`, { zoneId: f.dzone.value, prefix: f.dpre.value })
            .then(r => toast(`Domain ${r.prefix === '@' ? r.zoneName : r.prefix + '.' + r.zoneName} eingerichtet`, 'good'))
            .catch(err => toast('Server ist angelegt, aber die Domain ging nicht: ' + err.message, 'bad'));
        }
        go(`#/s/${srv.id}/konsole`);
      } catch (err) { fail(err); btn.disabled = false; }
    };
  };
  const pre = sessionStorage.getItem('sw-newegg');
  sessionStorage.removeItem('sw-newegg');
  egg = pre && eggs.find(e => e.id === pre);
  if (egg) step2(); else step1();
}

/* ---------- Eggs ---------- */
const ARCH_NAMES = { amd64: 'x86 (amd64)', arm64: 'ARM (arm64)' };
const archName = a => ARCH_NAMES[a] || a;

const catIcon = url => '/api/catalog/icon?v=3&u=' + encodeURIComponent(url);

async function pageEggs(page, sub, arg) {
  if (sub === 'e' && arg) return eggPage(page, { id: arg });
  if (sub === 'k' && arg) return eggPage(page, { url: decodeURIComponent(arg) });
  page.innerHTML = `<div class="head"><div><h1>Eggs</h1><div class="sub term-label">$ ls /eggs &nbsp; Vorlagen für Spiele und Programme, kompatibel mit Pterodactyl und Pelican</div></div>
    <div class="tools"><button class="btn" id="imp">${I('upload')}Datei / Adresse</button></div></div>
    <div class="chips" style="margin-bottom:18px"><a class="chip ${sub ? '' : 'on'}" href="#/eggs">Installiert</a><a class="chip ${sub === 'katalog' ? 'on' : ''}" href="#/eggs/katalog">Katalog</a></div>
    <div id="ebody"><div class="muted">Lädt ...</div></div>`;
  $('#imp').onclick = importDialog;
  if (sub === 'katalog') return catalogView();
  const eggsR = await api('GET', '/eggs').catch(fail) || { eggs: [] };
  const eggs = eggsR.eggs || [];
  const body = $('#ebody');
  if (!eggs.length) { body.innerHTML = `<div class="empty"><span class="term-label">$ ls /eggs<br>(leer)</span>${tt('Noch keine Eggs. Im <a href="#/eggs/katalog">Katalog</a> gibt es über 300 fertige Vorlagen.', 'No eggs yet. The <a href="#/eggs/katalog">catalog</a> has over 300 ready-made templates.')}</div>`; return; }
  const cats = [...new Set(eggs.map(e => e.category))];
  body.innerHTML = cats.map(c => `<div class="cat-h">${esc(c)}</div><div class="eggs">${eggs.filter(e => e.category === c).map(e => `
    <a class="egg pick" href="#/eggs/e/${e.id}">${thumb(eggIcon(e.id), e.name, 'banner')}<div class="eb">
      <div class="row"><b class="grow ell">${esc(e.name)}</b>${e.compatible === false ? `<span class="tag bad">nicht für ${esc(archName(eggsR.arch))}</span>` : `<span class="tag">${e.servers} Server</span>`}</div>
      <p>${esc(e.description)}</p></div></a>`).join('')}</div>`).join('');
}

// Detailseite für ein installiertes Egg (id) oder eins aus dem Katalog (url).
async function eggPage(page, { id, url }) {
  page.innerHTML = '<div class="row muted"><span class="spin"></span>Lädt ...</div>';
  let e, inst = null, cat = null;
  try {
    if (id) { e = await api('GET', '/eggs/' + id); inst = e; }
    else {
      const [egg, list] = await Promise.all([api('GET', '/catalog/egg?u=' + encodeURIComponent(url)), api('GET', '/catalog')]);
      e = egg; cat = list.eggs.find(x => x.url === url);
      if (cat?.installed) inst = { id: cat.installed };
    }
  } catch (err) { page.innerHTML = `<div class="empty">${esc(err.message)}</div>`; return; }
  const icon = id ? eggIcon(id) : catIcon(url);
  const vars = e.variables || [];
  const servers = id ? (await api('GET', '/servers').catch(() => [])).filter(s => s.egg === id) : [];
  page.innerHTML = `
    <div class="ehero">${thumb(icon, e.name, 'banner hero')}
      <div class="ehead"><a class="ib" href="${id ? '#/eggs' : '#/eggs/katalog'}" title="Zurück">${I('back')}</a>
        ${thumb(icon, e.name, 'sq big')}
        <div class="grow" style="min-width:0"><h1 class="ell">${esc(e.name)}</h1>
          <div class="row wrap" style="gap:6px;margin-top:6px"><span class="tag acc">${esc(e.category || cat?.category || '')}</span><span class="tag">${esc(e.format)}</span>${(e.features || []).map(f => `<span class="tag">${esc(f)}</span>`).join('')}</div></div>
        <div class="row wrap" id="eacts"></div></div></div>
    ${e.compatible === false ? `<div class="banner bad"><span class="grow">${tt(`<b>Läuft nicht auf diesem Rechner.</b> Die Docker-Images dieses Eggs gibt es nicht für ${esc(archName(e.arch))}.`, `<b>Doesn't run on this machine.</b> This egg's Docker images are not available for ${esc(archName(e.arch))}.`)}</span></div>` : ''}
    <div class="egrid">
      <div class="stack">
        <div class="panel"><div class="ph"><h2>Beschreibung</h2></div><div class="pb"><p class="prose" style="margin:0">${esc(e.description || 'Keine Beschreibung.').replace(/\n/g, '<br>')}</p></div></div>
        <div class="panel"><div class="ph"><h2>Startbefehl</h2></div><div class="pb"><div class="code">${esc(e.startup)}</div></div></div>
        <div class="panel"><div class="ph"><h2>Variablen</h2><span class="faint" style="font-size:13px">${vars.length}</span></div>
          ${vars.length ? `<table class="table"><thead><tr><th>Name</th><th>Standard</th><th>Regeln</th></tr></thead><tbody>${vars.map(v => `
            <tr><td><b style="font-weight:600">${esc(v.name)}</b><div class="mono faint" style="font-size:12px">${esc(v.env)}</div><div class="muted" style="font-size:12.5px;margin-top:4px">${esc(v.description).slice(0, 260)}</div></td>
            <td class="mono" style="font-size:12.5px;word-break:break-all">${esc(v.default) || '<span class="faint">leer</span>'}</td><td class="mono faint" style="font-size:12px;word-break:break-all">${esc(v.rules)}</td></tr>`).join('')}</tbody></table>` : '<div class="pb muted">Keine Variablen.</div>'}</div>
        ${e.installScript ? `<details class="panel more-panel"><summary class="ph"><h2>Installationsskript</h2><span class="faint" style="font-size:13px">${esc(e.installImage)} · ${esc(e.installEntrypoint)}</span></summary><pre class="code script">${esc(e.installScript)}</pre></details>` : ''}
      </div>
      <div class="stack">
        <div class="panel"><div class="ph"><h2>Infos</h2></div><dl class="pb kv one">
          <dt>Autor</dt><dd>${esc(e.author || '–')}</dd>
          <dt>Fertig, wenn</dt><dd class="mono" style="font-size:12.5px">${(e.done || []).map(esc).join('<br>') || 'sofort'}</dd>
          <dt>Stoppen mit</dt><dd class="mono" style="font-size:12.5px">${esc(e.stop)}</dd>
          ${e.imported && id ? `<dt>Importiert</dt><dd>${when(e.imported)}</dd>` : ''}
          ${e.source?.startsWith('https://') ? `<dt>Quelle</dt><dd><a href="${esc(e.source)}" target="_blank" rel="noopener" class="mono" style="font-size:12px;word-break:break-all">${esc(e.source.replace('https://raw.githubusercontent.com/', ''))}</a></dd>` : ''}
        </dl></div>
        <div class="panel"><div class="ph"><h2>Docker-Images</h2></div><div class="pb stack" style="gap:8px">${(e.images || []).map(i => `<div><b style="font-weight:600;font-size:14px">${esc(i.name)}</b><div class="mono faint" style="font-size:12px;word-break:break-all">${esc(i.image)}</div></div>`).join('')}</div></div>
        ${id ? `<div class="panel"><div class="ph"><h2>Server mit diesem Egg</h2></div><div class="pb stack" style="gap:8px">${servers.map(s => `<a class="row" href="#/s/${s.id}" style="text-decoration:none">${thumb(icon, s.name, 'sq sm')}<span class="grow ell">${esc(s.name)}</span><span class="status ${s.state}">${stateText[s.state] || s.state}</span></a>`).join('') || '<span class="muted">Noch keiner.</span>'}</div></div>` : ''}
      </div>
    </div>`;
  const acts = $('#eacts');
  const newServer = eid => { sessionStorage.setItem('sw-newegg', eid); go('#/neu'); };
  if (inst) {
    acts.innerHTML = `<button class="btn primary" id="a-new">${I('plus')}Server anlegen</button>
      ${id && e.updateUrl ? `<button class="ib ring" id="a-upd" title="Egg aktualisieren">${I('refresh')}</button>` : ''}
      ${id ? `<button class="ib ring danger" id="a-del" title="Egg löschen">${I('trash')}</button>` : `<a class="btn" href="#/eggs/e/${inst.id}">Installiert ansehen</a>`}`;
    $('#a-new').onclick = () => newServer(inst.id);
    const upd = $('#a-upd');
    if (upd) upd.onclick = () => api('POST', `/eggs/${id}/update`).then(() => { toast('Egg aktualisiert', 'good'); render(); }).catch(fail);
    const del = $('#a-del');
    if (del) del.onclick = async () => { if (await confirmBox('Egg löschen?', e.name, { ok: 'Löschen', danger: true })) api('DELETE', '/eggs/' + id).then(() => go('#/eggs')).catch(fail); };
  } else {
    acts.innerHTML = `<button class="btn primary" id="a-get">${I('download')}Holen</button>`;
    $('#a-get').onclick = async ev => {
      const b = ev.currentTarget; b.disabled = true; b.innerHTML = '<span class="spin"></span>Holt ...';
      try {
        const r = await api('POST', '/catalog/import', { url: cat.url, altUrl: cat.altUrl, category: cat.category });
        toast(`${r.name} ist installiert`, 'good');
        go('#/eggs/e/' + r.id);
      } catch (err) { fail(err); b.disabled = false; b.innerHTML = I('download') + 'Holen'; }
    };
  }
}

function importDialog() {
  const m = modal(`<h3>Egg importieren</h3><p>Eine Egg-Datei (.json oder .yaml) von Pterodactyl oder Pelican, oder eine https-Adresse dazu.</p>
    <form class="stack"><label class="f">Datei<input class="input" type="file" accept=".json,.yaml,.yml" style="padding-top:9px"></label>
    <label class="f">oder Adresse<input class="input" name="url" placeholder="https://raw.githubusercontent.com/..."></label>
    <label class="f">Kategorie<input class="input" name="cat" placeholder="z. B. Minecraft"></label>
    <div class="acts"><button type="button" class="btn" data-close>Abbrechen</button><button class="btn primary">Importieren</button></div></form>`);
  $('form', m.el).onsubmit = async e => {
    e.preventDefault();
    const f = e.target, file = $('input[type=file]', f).files[0];
    try {
      let r;
      if (file) r = await api('POST', '/eggs', await file.text());
      else if (f.url.value) r = await api('POST', '/eggs', { url: f.url.value, category: f.cat.value });
      else return toast('Datei oder Adresse angeben', 'bad');
      m.close(); toast(`${r.name} importiert`, 'good'); go('#/eggs/e/' + r.id);
    } catch (err) { fail(err); }
  };
}

async function catalogView() {
  const body = $('#ebody');
  let data;
  const load = async refresh => {
    body.innerHTML = '<div class="row muted"><span class="spin"></span>Katalog wird geladen ...</div>';
    data = await api('GET', '/catalog' + (refresh ? '?refresh=1' : '')).catch(e => { fail(e); return null; });
    if (data) draw();
  };
  let q = sessionStorage.getItem('sw-catq') || '', cat = sessionStorage.getItem('sw-cat') ?? 'Minecraft';
  let poll = 0;
  const draw = () => {
    const ok = data.eggs.filter(e => e.compatible !== false);
    const hidden = data.eggs.length - ok.length, ar = data.arch || {};
    const cats = [...new Set(ok.map(e => e.category))].sort();
    const f = ok.filter(e => (!cat || e.category === cat) && (!q || (e.name + ' ' + e.description).toLowerCase().includes(q.toLowerCase())));
    body.innerHTML = `<div class="row wrap" style="margin-bottom:14px"><div class="search grow" style="max-width:420px">${I('search')}<input class="input" id="cq" placeholder="${tt('Suchen (z. B. Paper, Valheim, Teamspeak) ...', 'Search (e.g. Paper, Valheim, Teamspeak) ...')}" value="${esc(q)}"></div>
      <span class="grow"></span><span class="faint" style="font-size:12.5px">${ok.length} Eggs für ${esc(archName(ar.arch))}${hidden ? `, ${hidden} ${tt('ausgeblendet', 'hidden')}` : ''}${ar.scanning ? ` · prüfe Architektur ${ar.done}/${ar.total}` : ''} · Stand ${when(data.fetched)}</span><button class="ib" id="cref" title="Katalog neu laden">${I('refresh')}</button></div>
      <div class="chips" style="margin-bottom:18px"><button class="chip ${!cat ? 'on' : ''}" data-c="">Alle</button>${cats.map(c => `<button class="chip ${c === cat ? 'on' : ''}" data-c="${esc(c)}">${esc(c)}</button>`).join('')}</div>
      <div class="eggs" id="clist">${f.slice(0, 300).map(e => `<a class="egg pick" href="#/eggs/k/${encodeURIComponent(e.url)}">${thumb(catIcon(e.url), e.name, 'banner')}<div class="eb">
        <div class="row"><b class="grow ell" title="${esc(e.name)}">${esc(e.name)}</b>
        ${e.installed ? `<span class="tag acc">${tt('installiert', 'installed')}</span>` : `<button class="btn small" data-i="${data.eggs.indexOf(e)}">${I('download')}Holen</button>`}</div>
        <p>${esc(e.description)}</p>${!cat ? `<div class="row"><span class="tag">${esc(e.category)}</span></div>` : ''}</div></a>`).join('') || '<div class="muted">Nichts gefunden.</div>'}</div>
      ${f.length > 300 ? `<p class="muted">… und ${f.length - 300} weitere, bitte genauer suchen.</p>` : ''}`;
    const cq = $('#cq');
    cq.oninput = e => { q = e.target.value; sessionStorage.setItem('sw-catq', q); const p = cq.selectionStart; draw(); const n = $('#cq'); n.focus(); n.setSelectionRange(p, p); };
    $('#cref').onclick = () => load(true);
    clearTimeout(poll);
    if (ar.scanning) poll = setTimeout(async () => { if (!$('#clist')) return; const d = await api('GET', '/catalog').catch(() => null); if (d && $('#clist')) { data = d; const y = scrollY; draw(); scrollTo(0, y); } }, 6000);
    $$('[data-c]').forEach(b => b.onclick = () => { cat = b.dataset.c; sessionStorage.setItem('sw-cat', cat); draw(); });
    $$('[data-i]').forEach(b => b.onclick = async ev => {
      ev.preventDefault(); ev.stopPropagation();
      const e = data.eggs[b.dataset.i];
      b.disabled = true; b.innerHTML = '<span class="spin"></span>';
      try {
        const r = await api('POST', '/catalog/import', { url: e.url, altUrl: e.altUrl, category: e.category });
        e.installed = r.id; toast(`${r.name} ist jetzt installiert`, 'good'); draw();
      } catch (err) { fail(err); b.disabled = false; b.innerHTML = I('download') + 'Holen'; }
    });
  };
  load(false);
  // Noch ladende Logos abbrechen, sonst blockieren sie die nächste Seite.
  onLeave(() => { clearTimeout(poll); $$('#ebody img').forEach(i => { if (!i.complete) i.removeAttribute('src'); }); });
}

/* ---------- Markdown (für Plugin-Beschreibungen) ---------- */
const MD_TAGS = new Set(['A', 'B', 'STRONG', 'I', 'EM', 'U', 'S', 'DEL', 'P', 'BR', 'HR', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'UL', 'OL', 'LI', 'CODE', 'PRE', 'BLOCKQUOTE', 'IMG', 'TABLE', 'THEAD', 'TBODY', 'TR', 'TH', 'TD', 'DETAILS', 'SUMMARY', 'SPAN', 'DIV', 'CENTER', 'SUP', 'SUB', 'KBD', 'MARK', 'SMALL', 'PICTURE', 'SOURCE', 'FIGURE', 'FIGCAPTION']);
function sanitize(html) {
  const doc = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html');
  const walk = el => {
    for (const n of [...el.children]) {
      if (!MD_TAGS.has(n.tagName)) { n.replaceWith(...(['SCRIPT', 'STYLE', 'IFRAME', 'OBJECT', 'EMBED', 'FORM'].includes(n.tagName) ? [] : n.childNodes)); continue; }
      for (const a of [...n.attributes]) {
        const k = a.name.toLowerCase(), v = a.value.trim();
        const okAttr = ['href', 'src', 'srcset', 'alt', 'title', 'width', 'height', 'align', 'colspan', 'rowspan', 'open'].includes(k);
        if (!okAttr || ((k === 'href' || k === 'src' || k === 'srcset') && !/^(https?:|#|\/\/)/i.test(v))) n.removeAttribute(a.name);
      }
      if (n.tagName === 'A') { n.target = '_blank'; n.rel = 'noopener noreferrer'; }
      if (n.tagName === 'IMG') { n.loading = 'lazy'; n.referrerPolicy = 'no-referrer'; }
      walk(n);
    }
  };
  walk(doc.body.firstChild);
  return doc.body.firstChild.innerHTML;
}
function mdInline(t) {
  const codes = [];
  t = t.replace(/`([^`]+)`/g, (_, c) => { codes.push(c); return `\u0000${codes.length - 1}\u0000`; });
  t = t.replace(/!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g, '<img alt="$1" src="$2">')
    .replace(/\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g, '<a href="$2">$1</a>')
    .replace(/(^|[\s(])(https?:\/\/[^\s<)]+)/g, '$1<a href="$2">$2</a>')
    .replace(/\*\*([^*]+)\*\*|__([^_]+)__/g, (_, a, b) => `<strong>${a || b}</strong>`)
    .replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
    .replace(/~~([^~]+)~~/g, '<del>$1</del>');
  return t.replace(/\u0000(\d+)\u0000/g, (_, i) => `<code>${esc(codes[i])}</code>`);
}
function markdown(src) {
  const lines = String(src || '').replace(/\r\n/g, '\n').split('\n');
  let out = '', i = 0;
  while (i < lines.length) {
    let l = lines[i];
    if (/^```/.test(l)) {
      const buf = []; i++;
      while (i < lines.length && !/^```/.test(lines[i])) buf.push(lines[i++]);
      i++; out += `<pre><code>${esc(buf.join('\n'))}</code></pre>`; continue;
    }
    if (/^\s*$/.test(l)) { i++; continue; }
    let m;
    if ((m = l.match(/^(#{1,6})\s+(.*)$/))) { out += `<h${m[1].length}>${mdInline(m[2])}</h${m[1].length}>`; i++; continue; }
    if (/^\s*(-{3,}|\*{3,}|_{3,})\s*$/.test(l)) { out += '<hr>'; i++; continue; }
    if (/^\s*>/.test(l)) {
      const buf = [];
      while (i < lines.length && /^\s*>/.test(lines[i])) buf.push(lines[i++].replace(/^\s*>\s?/, ''));
      out += `<blockquote>${markdown(buf.join('\n'))}</blockquote>`; continue;
    }
    if (/^\s*\|.*\|\s*$/.test(l) && /^\s*\|?\s*:?-{2,}/.test(lines[i + 1] || '')) {
      const row = r => r.trim().replace(/^\||\|$/g, '').split('|').map(c => mdInline(c.trim()));
      const head = row(l); i += 2; const body = [];
      while (i < lines.length && /^\s*\|/.test(lines[i])) body.push(row(lines[i++]));
      out += `<table><thead><tr>${head.map(c => `<th>${c}</th>`).join('')}</tr></thead><tbody>${body.map(r => `<tr>${r.map(c => `<td>${c}</td>`).join('')}</tr>`).join('')}</tbody></table>`; continue;
    }
    if (/^\s*([-*+]|\d+[.)])\s+/.test(l)) {
      const ordered = /^\s*\d/.test(l); const items = [];
      while (i < lines.length && (/^\s*([-*+]|\d+[.)])\s+/.test(lines[i]) || (/^\s{2,}\S/.test(lines[i]) && items.length))) {
        if (/^\s*([-*+]|\d+[.)])\s+/.test(lines[i])) items.push(lines[i].replace(/^\s*([-*+]|\d+[.)])\s+/, ''));
        else items[items.length - 1] += ' ' + lines[i].trim();
        i++;
      }
      out += `<${ordered ? 'ol' : 'ul'}>${items.map(x => `<li>${mdInline(x)}</li>`).join('')}</${ordered ? 'ol' : 'ul'}>`; continue;
    }
    if (/^\s*</.test(l)) { // rohes HTML-Stück
      const buf = [];
      while (i < lines.length && !/^\s*$/.test(lines[i])) buf.push(lines[i++]);
      out += mdInline(buf.join('\n')); continue;
    }
    const buf = [];
    while (i < lines.length && !/^\s*$/.test(lines[i]) && !/^(#{1,6}\s|```|\s*>|\s*([-*+]|\d+[.)])\s)/.test(lines[i])) buf.push(lines[i++]);
    out += `<p>${mdInline(buf.join('\n')).replace(/ {2,}\n|\\\n/g, '<br>')}</p>`;
  }
  return out;
}

// Detailansicht eines Modrinth-Projekts im Panel.
async function addonDetail(box, { hit, info, ver, installedFile, onInstall, onBack }) {
  box.innerHTML = '<div class="row muted"><span class="spin"></span>Lädt ...</div>';
  let p, vs;
  try {
    const q = new URLSearchParams({ loaders: JSON.stringify(info.loaders) });
    [p, vs] = await Promise.all([fetch(`${MR}/project/${hit.project_id}`).then(r => r.json()), fetch(`${MR}/project/${hit.project_id}/version?${q}`).then(r => r.json())]);
  } catch { box.innerHTML = '<div class="empty">Modrinth ist gerade nicht erreichbar.</div>'; return; }
  const fits = v => !ver || v.game_versions.includes(ver);
  const best = vs.find(v => fits(v) && v.version_type === 'release') || vs.find(fits);
  const links = [['source_url', 'Quellcode'], ['wiki_url', 'Wiki'], ['issues_url', 'Fehler melden'], ['discord_url', 'Discord']].filter(([k]) => p[k]);
  box.innerHTML = `
    <button class="btn ghost small" id="ad-back" style="margin-bottom:12px">${I('back')}Zurück zur Liste</button>
    <div class="adhead">${thumb(p.icon_url, p.title, 'sq big')}
      <div class="grow" style="min-width:0"><h2>${esc(p.title)}</h2><div class="muted" style="font-size:14px">${esc(p.description)}</div>
        <div class="row wrap" style="gap:6px;margin-top:10px"><span class="tag">${I('download', 'xs')} ${compact(p.downloads)}</span><span class="tag">von ${esc(hit.author)}</span>
        ${p.categories.filter(c => !info.loaders.includes(c)).slice(0, 6).map(c => `<span class="tag">${esc(c)}</span>`).join('')}</div></div>
      <div class="acol">${best ? `<button class="btn primary" id="ad-inst">${I('download')}${installedFile ? 'Neu installieren' : 'Installieren'}</button><span class="faint" style="font-size:12px">${esc(best.version_number)}${ver ? ' für MC ' + esc(ver) : ''}</span>`
        : `<span class="tag" style="color:var(--danger)">keine Version${ver ? ' für MC ' + esc(ver) : ''}</span>`}</div></div>
    ${p.gallery?.length ? `<div class="gallery">${p.gallery.slice(0, 8).map(g => `<img src="${esc(g.url)}" alt="${esc(g.title || '')}" loading="lazy" referrerpolicy="no-referrer" data-full="${esc(g.raw_url || g.url)}">`).join('')}</div>` : ''}
    <div class="egrid">
      <div class="panel"><div class="pb prose md">${sanitize(markdown(p.body))}</div></div>
      <div class="stack">
        <div class="panel"><div class="ph"><h2>Versionen</h2><span class="faint" style="font-size:12.5px">${vs.length}</span></div><div class="vlist">${vs.slice(0, 25).map((v, i) => `
          <div class="vrow ${fits(v) ? '' : 'nofit'}"><div class="grow" style="min-width:0"><b class="ell" style="display:block;font-size:13.5px">${esc(v.version_number)}</b>
            <span class="faint" style="font-size:12px">${esc(v.game_versions.slice(-3).join(', '))}${v.game_versions.length > 3 ? ' …' : ''} · ${new Date(v.date_published).toLocaleDateString(I18N.lang === 'en' ? 'en-GB' : 'de-DE')}${v.version_type !== 'release' ? ' · ' + esc(v.version_type) : ''}</span></div>
            <button class="ib" data-v="${i}" title="Diese Version installieren">${I('download')}</button></div>`).join('') || '<div class="pb muted">Keine Version für diesen Server.</div>'}</div></div>
        ${links.length ? `<div class="panel"><div class="ph"><h2>Links</h2></div><div class="pb stack" style="gap:6px">${links.map(([k, n]) => `<a href="${esc(p[k])}" target="_blank" rel="noopener">${n}</a>`).join('')}</div></div>` : ''}
        <div class="panel"><dl class="pb kv one"><dt>Lizenz</dt><dd>${esc(p.license?.name || p.license?.id || '–')}</dd><dt>Aktualisiert</dt><dd>${when(p.updated)}</dd><dt>Quelle</dt><dd>modrinth.com</dd></dl></div>
      </div>
    </div>`;
  $('#ad-back').onclick = onBack;
  const doInst = async (v, btn) => {
    const old = btn.innerHTML; btn.disabled = true; btn.innerHTML = '<span class="spin"></span>';
    try { await onInstall(p, v); btn.innerHTML = I('check') + (btn.classList.contains('ib') ? '' : 'Installiert'); }
    catch (e) { fail(e); btn.innerHTML = old; }
    btn.disabled = false;
  };
  const ib = $('#ad-inst'); if (ib) ib.onclick = () => doInst(best, ib);
  $$('[data-v]', box).forEach(b => b.onclick = () => doInst(vs[b.dataset.v], b));
  $$('.gallery img', box).forEach(im => im.onclick = () => modal(`<img src="${esc(im.dataset.full)}" style="width:100%;border-radius:12px" referrerpolicy="no-referrer"><div class="acts"><button class="btn" data-close>Schließen</button></div>`, { wide: true }));
}

/* ---------- Akzentfarbe ---------- */
function hexRgb(h) { const n = parseInt(h.slice(1), 16); return [n >> 16 & 255, n >> 8 & 255, n & 255]; }
function applyAccent(hex) {
  let st = document.getElementById('accent-style');
  if (!hex || !/^#[0-9a-f]{6}$/i.test(hex)) { st?.remove(); return; }
  if (!st) { st = document.createElement('style'); st.id = 'accent-style'; document.head.append(st); }
  const [r, g, b] = hexRgb(hex);
  const lum = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
  const ink = lum > 0.55 ? '#17140f' : '#ffffff';
  // Für helles Design abdunkeln, damit es auf Weiß lesbar bleibt.
  const k = lum > 0.45 ? 0.62 : 0.85;
  const [dr, dg, db] = [r, g, b].map(x => Math.round(x * k));
  const dark = `--accent:${hex};--accent-ink:${ink};--accent-soft:rgba(${r},${g},${b},.14);`;
  const light = `--accent:rgb(${dr},${dg},${db});--accent-ink:#fff;--accent-soft:rgba(${dr},${dg},${db},.12);`;
  st.textContent = `:root{${dark}} :root[data-theme="dark"]{${dark}} @media (prefers-color-scheme: light){:root:not([data-theme="dark"]){${light}}} :root[data-theme="light"]{${light}}`;
}
const ACCENTS = [['#d9c7a7', 'Sand'], ['#8fcf9a', 'Grün'], ['#76c7cf', 'Türkis'], ['#8ab4f8', 'Blau'], ['#b9a3f0', 'Violett'], ['#f0a6ca', 'Rosa'], ['#f2a65a', 'Orange'], ['#e5807a', 'Rot']];

/* ---------- Einstellungen ---------- */
async function pageSettings(page, sec = 'allgemein') {
  page.innerHTML = `<div class="head"><div><h1>Einstellungen</h1><div class="sub term-label">$ nano /etc/serverwerk.conf</div></div></div><div id="setb"><div class="muted">Lädt ...</div></div>`;
  const d = await api('GET', '/settings').catch(fail); if (!d) return;
  const theme = localStorage.getItem('sw-theme') || 'auto';
  let accent = d.accent || '';
  const SECS = [['allgemein', 'Allgemein', 'gear'], ['aussehen', 'Aussehen', 'sliders'], ['server', 'Neue Server', 'server'], ['sicherheit', 'Sicherheit', 'key'], ['cloudflare', 'Cloudflare & Domains', 'plug'], ['import', 'Import', 'download']];
  if (!SECS.find(x => x[0] === sec)) sec = 'allgemein';
  const df = d.defaults || {}, sc = d.security || {};
  $('#setb').innerHTML = `<div class="setlayout">
    <nav class="setnav">${SECS.map(([k, n, i]) => `<a href="#/einstellungen/${k}" class="${k === sec ? 'on' : ''}">${I(i)}<span>${n}</span></a>`).join('')}</nav>
    <div class="setbody">
    <section data-sec="allgemein" class="stack">
      <div class="panel"><div class="ph"><h2>Allgemein</h2></div><div class="pb stack">
        <label class="f">Name des Panels<input class="input" id="s-name" value="${esc(d.panelName)}"><small>Steht im Browser-Tab und oben links.</small></label>
        <label class="f">Sprache<select class="input" id="s-lang"><option value="en" ${d.language === 'en' ? 'selected' : ''}>English</option><option value="de" ${d.language !== 'en' ? 'selected' : ''}>Deutsch</option></select><small>Gilt für alle Benutzer.</small></label>
        <label class="f">Zeitzone für alle Server<input class="input mono" id="s-tz" list="tzlist" value="${esc(d.timezone || '')}" placeholder="${esc(d.hostTimezone || 'Europe/Berlin')} (wie dieser Rechner)">
          <datalist id="tzlist">${['Europe/Berlin', 'Europe/Vienna', 'Europe/Zurich', 'Europe/London', 'Europe/Paris', 'UTC', 'America/New_York', 'Asia/Tokyo'].map(z => `<option value="${z}">`).join('')}</datalist>
          <small>Bestimmt die Uhrzeit in den Server-Logs. Leer = wie dieser Rechner. Gilt ab dem nächsten Start.</small></label>
        <div><button class="btn primary" id="s-gen">${I('check')}Speichern</button></div></div></div>
      <div class="panel"><dl class="pb kv"><dt>Version</dt><dd>Serverwerk ${esc(d.version || '')}</dd><dt>Aktualisieren</dt><dd class="mono" style="font-size:12.5px">Installationsbefehl erneut ausführen</dd><dt>Passwort vergessen</dt><dd class="mono" style="font-size:12.5px">sudo serverwerk passwort &lt;name&gt;</dd></dl></div>
    </section>

    <section data-sec="aussehen" class="stack">
      <div class="panel"><div class="ph"><h2>Aussehen</h2></div><div class="pb stack">
        <div class="f"><span>Akzentfarbe (für alle Benutzer)</span>
          <div class="swatches">${ACCENTS.map(([c, n]) => `<button class="sw" data-c="${c}" style="--c:${c}" title="${n}"></button>`).join('')}
            <label class="sw custom" title="Eigene Farbe"><input type="color" id="s-col" value="${esc(accent || '#d9c7a7')}">${I('plus')}</label></div></div>
        <div class="f"><span>Design (nur für dich, in diesem Browser)</span><div class="chips">${['auto', 'dark', 'light'].map(t => `<button class="chip ${t === theme ? 'on' : ''}" data-t="${t}">${{ auto: 'Wie System', dark: 'Dunkel', light: 'Hell' }[t]}</button>`).join('')}</div></div>
        <div class="preview"><button class="btn primary small">Knopf</button><span class="tag acc">Markierung</span><a href="#/einstellungen" onclick="return false" style="color:var(--accent);font-size:14px">Link</a><span class="switch"><input type="checkbox" checked><i></i></span></div>
        <div><button class="btn primary" id="s-save">${I('check')}Speichern</button></div>
      </div></div>
    </section>

    <section data-sec="server" class="stack">
      <div class="panel"><div class="ph"><h2>Standardwerte für neue Server</h2></div><div class="pb stack">
        <p class="muted" style="margin:0">Damit ist das Formular „Neuer Server“ schon so ausgefüllt, wie du es meistens brauchst. Pro Server lässt sich alles ändern.</p>
        <div class="grid2">
          <label class="f">Arbeitsspeicher (MB)<input class="input" type="number" min="0" id="df-mem" value="${df.memory ?? 2048}"><small>0 = unbegrenzt. Minecraft-Server starten mit mindestens 4096.</small></label>
          <label class="f">CPU (% eines Kerns)<input class="input" type="number" min="0" id="df-cpu" value="${df.cpu ?? 0}"><small>0 = unbegrenzt, 200 = zwei Kerne</small></label>
          <label class="f">Speicherplatz (MB)<input class="input" type="number" min="0" id="df-disk" value="${df.disk ?? 0}"><small>0 = unbegrenzt</small></label>
          <label class="f">Backups behalten<input class="input" type="number" min="1" max="100" id="df-bak" value="${df.maxBackups ?? 5}"><small>Ältere werden automatisch gelöscht.</small></label>
        </div>
        <label class="check"><span class="switch"><input type="checkbox" id="df-auto" ${df.autoRestart !== false ? 'checked' : ''}><i></i></span>Nach Absturz automatisch neu starten</label>
        <label class="check"><span class="switch"><input type="checkbox" id="df-start" ${df.startAfter !== false ? 'checked' : ''}><i></i></span>Nach der Installation gleich starten</label>
        <div><button class="btn primary" id="s-def">${I('check')}Speichern</button></div></div></div>
      <div class="panel"><div class="pb row wrap"><span class="grow muted">Welche Ports neue Server bekommen, stellst du unter Allocations ein.</span><a class="btn" href="#/allocations">${I('plug')}Allocations</a></div></div>
    </section>

    <section data-sec="sicherheit" class="stack">
      <div class="panel"><div class="ph"><h2>Anmeldung</h2></div><div class="pb stack">
        <div class="grid2">
          <label class="f">Angemeldet bleiben (Tage)<input class="input" type="number" min="1" max="365" id="se-days" value="${sc.sessionDays ?? 30}"><small>Danach muss man sich neu anmelden.</small></label>
          <label class="f">Fehlversuche bis zur Sperre<input class="input" type="number" min="3" max="100" id="se-fails" value="${sc.maxFails ?? 8}"><small>Pro Adresse.</small></label>
          <label class="f">Sperre (Minuten)<input class="input" type="number" min="1" max="1440" id="se-lock" value="${sc.lockMinutes ?? 10}"></label>
        </div>
        <div><button class="btn primary" id="s-sec">${I('check')}Speichern</button></div></div></div>
      <div class="panel"><div class="pb row wrap"><span class="grow muted">Benutzer und Admin-Rechte verwaltest du unter Benutzer, dein eigenes Passwort unter Konto (unten links).</span><a class="btn" href="#/benutzer">${I('users')}Benutzer</a></div></div>
    </section>

    <section data-sec="cloudflare" class="stack">
      <div class="panel"><div class="ph"><h2>Cloudflare</h2><span id="cf-tag"></span></div><div class="pb stack" id="cf"></div></div>
      <div class="panel" id="pub-panel" hidden><div class="ph"><h2>Panel öffentlich machen</h2><span id="pub-tag"></span></div><div class="pb stack" id="pub"></div></div>
      <div class="panel" id="auto-panel" hidden><div class="ph"><h2>${tt('Automatische Server-Adressen', 'Automatic server addresses')}</h2><span id="auto-tag"></span></div><div class="pb stack" id="auto"></div></div>
    </section>

    <section data-sec="import" class="stack"><div id="imp"></div></section>
    </div>
  </div>`;
  $$('[data-sec]').forEach(x => x.hidden = x.dataset.sec !== sec);
  const save = (body, msg = 'Gespeichert') => api('PUT', '/settings', body).then(r => { toast(msg, 'good'); return r; }).catch(fail);
  $('#s-gen').onclick = () => save({ panelName: $('#s-name').value, timezone: $('#s-tz').value, language: $('#s-lang').value }).then(r => {
    if (!r) return;
    S.me.panelName = r.panelName; document.title = r.panelName;
    if (r.language !== I18N.lang) location.reload();
  });
  $('#s-def').onclick = () => save({ defaults: { memory: +$('#df-mem').value, cpu: +$('#df-cpu').value, disk: +$('#df-disk').value, maxBackups: +$('#df-bak').value, autoRestart: $('#df-auto').checked, startAfter: $('#df-start').checked } });
  $('#s-sec').onclick = () => save({ security: { sessionDays: +$('#se-days').value, maxFails: +$('#se-fails').value, lockMinutes: +$('#se-lock').value } }).then(r => {
    if (r) { $('#se-days').value = r.security.sessionDays; $('#se-fails').value = r.security.maxFails; $('#se-lock').value = r.security.lockMinutes; }
  });
  if (sec === 'import') importView($('#imp'));
  const markSw = () => $$('.sw[data-c]').forEach(b => b.classList.toggle('on', b.dataset.c.toLowerCase() === (accent || '#d9c7a7').toLowerCase()));
  markSw();
  $$('.sw[data-c]').forEach(b => b.onclick = () => { accent = b.dataset.c === '#d9c7a7' ? '' : b.dataset.c; applyAccent(accent); markSw(); });
  $('#s-col').oninput = e => { accent = e.target.value; applyAccent(accent); markSw(); };
  $$('[data-t]').forEach(b => b.onclick = () => { setTheme(b.dataset.t); applyAccent(accent); $$('[data-t]').forEach(x => x.classList.toggle('on', x === b)); });
  $('#s-save').onclick = () => api('PUT', '/settings', { accent }).then(r => {
    S.me.accent = r.accent; toast('Gespeichert', 'good');
  }).catch(fail);

  const cf = $('#cf');
  let cfOpen = false;
  const drawCF = d => {
    $('#cf-tag').innerHTML = d.cfConnected ? '<span class="tag acc">verbunden</span>' : '';
    drawPub(d);
    drawAuto(d);
    if (!d.cfConnected) {
      cf.innerHTML = `<p class="muted" style="margin:0">${tt('Verbinde dein Cloudflare-Konto, dann bekommt jeder Server auf Wunsch eine eigene Adresse wie <span class="mono">minecraft.deinedomain.de</span>. Bei Minecraft Java brauchen Spieler dann keinen Port mehr.', 'Connect your Cloudflare account and every server can get its own address like <span class="mono">minecraft.example.com</span>. For Minecraft Java, players then don\'t need a port anymore.')}</p>
        ${tt(`<ol class="steps-list">
          <li><a href="https://dash.cloudflare.com/profile/api-tokens" target="_blank" rel="noopener">dash.cloudflare.com &rsaquo; My Profile &rsaquo; API Tokens</a> öffnen und <b>Create Token</b> klicken. Einen Namen eintragen, z. B. <span class="mono">serverwerk</span>.</li>
          <li>Bei <b>Policies</b> auf <b>Start from scratch</b> &rsaquo; <b>Start with a Zone policy</b>. Oben <span class="mono">Include</span> + <span class="mono">All zones</span> lassen (oder nur bestimmte Domains wählen).</li>
          <li>In dieser Policy <b>DNS &amp; Zones</b> aufklappen und anhaken: <span class="mono">DNS</span> mit <b>Edit</b> bzw. <b>Write</b> und <span class="mono">Zone</span> mit <b>Read</b>. Daneben steht dann <span class="mono">2/12</span>. Tipp: Das Suchfeld darüber findet beide schneller.</li>
          <li>Nur fürs öffentliche Panel: unten <b>Add new policy</b> &rsaquo; <b>Account policy</b>, dein Konto wählen, im Suchfeld <span class="mono">Tunnel</span> eingeben und <span class="mono">Cloudflare Tunnel</span> mit <b>Edit</b> bzw. <b>Write</b> anhaken.</li>
          <li>Token expiration auf <b>No expiration</b> lassen, Client IP filtering leer lassen, dann <b>Continue to summary</b> &rsaquo; <b>Create Token</b>.</li>
          <li>Den Token kopieren (Cloudflare zeigt ihn nur einmal) und hier einfügen.</li></ol>`, `<ol class="steps-list">
          <li>Open <a href="https://dash.cloudflare.com/profile/api-tokens" target="_blank" rel="noopener">dash.cloudflare.com &rsaquo; My Profile &rsaquo; API Tokens</a> and click <b>Create Token</b>. Enter a name, e.g. <span class="mono">serverwerk</span>.</li>
          <li>Under <b>Policies</b> click <b>Start from scratch</b> &rsaquo; <b>Start with a Zone policy</b>. Keep <span class="mono">Include</span> + <span class="mono">All zones</span> at the top (or choose only certain domains).</li>
          <li>In this policy, expand <b>DNS &amp; Zones</b> and tick <span class="mono">DNS</span> with <b>Edit</b> (or <b>Write</b>) and <span class="mono">Zone</span> with <b>Read</b>. It then shows <span class="mono">2/12</span>. Tip: the search field above finds both faster.</li>
          <li>Only for the public panel: at the bottom <b>Add new policy</b> &rsaquo; <b>Account policy</b>, choose your account, type <span class="mono">Tunnel</span> into the search field and tick <span class="mono">Cloudflare Tunnel</span> with <b>Edit</b> (or <b>Write</b>).</li>
          <li>Leave token expiration on <b>No expiration</b> and Client IP filtering empty, then <b>Continue to summary</b> &rsaquo; <b>Create Token</b>.</li>
          <li>Copy the token (Cloudflare shows it only once) and paste it here.</li></ol>`)}
        <form class="row wrap" id="cf-f"><input class="input mono grow" id="cf-t" placeholder="API-Token" autocomplete="off" style="min-width:240px"><button class="btn primary">Verbinden</button></form>`;
      $('#cf-f').onsubmit = ev => {
        ev.preventDefault();
        const b = $('button', ev.target); b.disabled = true;
        api('PUT', '/settings', { cfToken: $('#cf-t').value }).then(r => { toast('Cloudflare verbunden', 'good'); cfOpen = false; drawCF(r); }).catch(e => { fail(e); b.disabled = false; });
      };
      return;
    }
    const target = d.cfTarget === 'host' && d.cfHost ? `CNAME auf <span class="mono">${esc(d.cfHost)}</span>` : `öffentliche IP <span class="mono">${esc(d.publicIp || '?')}</span> (automatisch)`;
    if (!cfOpen) {
      cf.innerHTML = `<div class="row wrap" style="gap:6px">${(d.cfZones || []).map(z => `<span class="tag acc">${esc(z.name)}</span>`).join('')}</div>
        <div class="row wrap"><span class="muted grow" style="font-size:13.5px">Adressen zeigen auf ${target}. ${d.domainServers} Server mit Domain.</span><button class="btn" id="cf-open">${I('gear')}Konfigurieren</button></div>`;
      $('#cf-open').onclick = () => { cfOpen = true; drawCF(d); };
      return;
    }
    cf.innerHTML = `
      <div class="f"><span>Domains</span><div class="row wrap" style="gap:6px">${(d.cfZones || []).map(z => `<span class="tag acc">${esc(z.name)}</span>`).join('')}</div></div>
      <div class="f"><span>Wohin sollen die Adressen der Spielserver zeigen?</span>
        <label class="radio"><input type="radio" name="tg" value="ip" ${d.cfTarget !== 'host' ? 'checked' : ''}><span><b>Öffentliche IP automatisch</b><small>${tt(`Jetzt: <span class="mono">${esc(d.publicIp || 'unbekannt')}</span>. Wird alle 5 Minuten geprüft und bei Änderung angepasst.`, `Now: <span class="mono">${esc(d.publicIp || 'unknown')}</span>. Checked every 5 minutes and updated when it changes.`)}</small></span></label>
        <label class="radio"><input type="radio" name="tg" value="host" ${d.cfTarget === 'host' ? 'checked' : ''}><span><b>Eigener Hostname</b><small>Zum Beispiel deine DynDNS-Adresse vom Router. Die Server zeigen dann per CNAME darauf.</small>
          <input class="input mono" id="cf-h" value="${esc(d.cfHost)}" placeholder="meinrouter.myfritz.net" style="margin-top:8px"></span></label></div>
      <div class="banner" style="margin:0"><span class="grow" style="font-size:13px">Die Ports der Server müssen im Router an diesen Rechner weitergeleitet sein. Einträge sind „nur DNS“ (graue Wolke), weil Spiele nicht über den Cloudflare-Proxy laufen. Serverwerk ändert nur Einträge, die es selbst angelegt hat.</span></div>
      <div class="row wrap"><button class="btn primary" id="cf-save">${I('check')}Speichern</button><button class="btn ghost" id="cf-close">Abbrechen</button><button class="btn" id="cf-new">Neuer Token</button><span class="grow"></span><span class="faint" style="font-size:12.5px">Token ${esc(d.cfToken)}</span></div>
      <p class="muted" style="margin:0;font-size:13px">Domains vergibst du pro Server unter Server &rsaquo; Einstellungen &rsaquo; Domain.</p>`;
    $('#cf-save').onclick = () => api('PUT', '/settings', { cfTarget: $('input[name=tg]:checked').value, cfHost: $('#cf-h').value }).then(r => { toast('Gespeichert, Einträge werden angepasst', 'good'); cfOpen = false; drawCF(r); }).catch(fail);
    $('#cf-close').onclick = () => { cfOpen = false; drawCF(d); };
    $('#cf-new').onclick = () => drawCF({ ...d, cfConnected: false });
  };

  // Jeder Server bekommt automatisch name.<basis>, z. B. survival.panel.example.com.
  const drawAuto = d => {
    const box = $('#auto'), card = $('#auto-panel');
    card.hidden = !d.cfConnected;
    if (!d.cfConnected) return;
    const ad = d.autoDomain, zones = d.cfZones || [];
    const p = d.panel && d.panel.prefix !== '@' ? d.panel : null;
    const sub = ad ? ad.sub : (p ? p.prefix : 'play');
    const zoneId = ad ? ad.zoneId : (d.panel?.zoneId || zones[0]?.id);
    $('#auto-tag').innerHTML = ad ? `<span class="tag acc">${tt('an', 'on')}</span>` : '';
    box.innerHTML = `<p class="muted" style="margin:0">${tt('Jeder Server bekommt automatisch eine eigene Adresse aus seinem Namen, z. B. <span class="mono">survival.panel.example.com</span>. Bei Minecraft Java ganz ohne Port, bei anderen Spielen mit Port dahinter. Die Adressen zeigen auf deinen Anschluss und werden angepasst, wenn sich die öffentliche IP ändert.', 'Every server automatically gets its own address from its name, e.g. <span class="mono">survival.panel.example.com</span>. For Minecraft Java with no port at all, for other games with the port appended. The addresses point to your connection and are updated when your public IP changes.')}</p>
      <label class="check"><input type="checkbox" id="au-on" ${ad ? 'checked' : ''}>${tt('Automatische Adressen für alle Server', 'Automatic addresses for all servers')}</label>
      <div class="row wrap" style="gap:8px;align-items:center">
        <span class="mono muted">${tt('name', 'name')}.</span>
        <input class="input mono" id="au-sub" value="${esc(sub)}" placeholder="panel" style="max-width:180px">
        <span class="mono muted">.</span>
        <select class="input" id="au-zone" style="max-width:240px">${zones.map(z => `<option value="${z.id}" ${z.id === zoneId ? 'selected' : ''}>${esc(z.name)}</option>`).join('')}</select></div>
      <div class="mono faint notr" id="au-ex" style="font-size:13px"></div>
      <div class="banner" style="margin:0"><span class="grow" style="font-size:13px">${tt('Die Ports der Server müssen im Router an diesen Rechner weitergeleitet sein. Beim Einschalten bekommen auch alle bestehenden Server ohne Domain eine Adresse. Mittleres Feld leer lassen = direkt unter der Domain.', 'The ports of the servers must be forwarded to this machine in your router. When you turn this on, all existing servers without a domain get an address too. Leave the middle field empty = directly under the domain.')}</span></div>
      <div><button class="btn primary" id="au-save">${I('check')}${tt('Speichern', 'Save')}</button></div>`;
    const ex = () => {
      const z = zones.find(z => z.id === $('#au-zone').value)?.name || '';
      const sb = $('#au-sub').value.trim().replace(/^\.+|\.+$/g, '').toLowerCase();
      $('#au-ex').textContent = (tt('Beispiel: ', 'Example: ')) + ['survival', sb, z].filter(Boolean).join('.');
    };
    ex();
    $('#au-sub').oninput = ex; $('#au-zone').onchange = ex;
    $('#au-save').onclick = () => api('PUT', '/settings', { autoDomain: { enabled: $('#au-on').checked, zoneId: $('#au-zone').value, sub: $('#au-sub').value } })
      .then(r => { toast($('#au-on').checked ? tt('Gespeichert, Adressen werden angelegt', 'Saved, addresses are being created') : tt('Gespeichert', 'Saved'), 'good'); drawCF(r); }).catch(fail);
  };

  // Panel über eine Domain erreichbar machen (Cloudflare Tunnel).
  let pubOpen = false;
  const drawPub = d => {
    const box = $('#pub'), card = $('#pub-panel');
    card.hidden = !d.cfConnected;
    if (!d.cfConnected) return;
    const p = d.panel;
    $('#pub-tag').innerHTML = p ? `<span class="tag ${p.running ? 'acc' : ''}">${p.running ? 'online' : 'gestört'}</span>` : '';
    if (p && !pubOpen) {
      box.innerHTML = `<div class="row wrap"><a class="mono grow" href="${esc(p.url)}" target="_blank" rel="noopener" style="color:var(--accent);font-size:15px">${esc(p.url)}</a>
          <button class="ib" id="pub-copy" title="Kopieren">${I('copy')}</button><button class="btn" id="pub-open">${I('gear')}Konfigurieren</button></div>
        ${p.problem ? `<div class="banner bad" style="margin:0"><span class="grow">${esc(p.problem)}</span></div>` : ''}`;
      $('#pub-copy').onclick = () => copy(p.url);
      $('#pub-open').onclick = () => { pubOpen = true; drawPub(d); };
      return;
    }
    const zones = d.cfZones || [];
    box.innerHTML = `<p class="muted" style="margin:0">${tt(`Das Panel bekommt eine eigene Adresse mit HTTPS, z. B. <span class="mono">panel.${esc(zones[0]?.name || 'deinedomain.de')}</span>. Dafür wird ein Cloudflare Tunnel benutzt: keine Portfreigabe nötig, die Verbindung geht von hier nach außen.`, `The panel gets its own HTTPS address, e.g. <span class="mono">panel.${esc(zones[0]?.name || 'example.com')}</span>. This uses a Cloudflare Tunnel: no port forwarding needed, the connection goes from here to the outside.`)}</p>
      <div class="row wrap" style="gap:8px;align-items:center">
        <input class="input mono" id="pub-pre" value="${esc(p ? (p.prefix === '@' ? '' : p.prefix) : 'panel')}" placeholder="panel" style="max-width:200px">
        <span class="mono muted">.</span>
        <select class="input" id="pub-zone" style="max-width:240px">${zones.map(z => `<option value="${z.id}" ${p && z.id === p.zoneId ? 'selected' : ''}>${esc(z.name)}</option>`).join('')}</select></div>
      <div class="banner" style="margin:0"><span class="grow" style="font-size:13px">Danach ist die Anmeldeseite aus dem ganzen Internet erreichbar. Benutze starke Passwörter. Nach 8 Fehlversuchen wird eine Adresse für 10 Minuten gesperrt.</span></div>
      <div class="row wrap"><button class="btn primary" id="pub-save">${I('check')}${p ? 'Ändern' : 'Öffentlich machen'}</button>
        ${p ? `<button class="btn ghost" id="pub-close">Abbrechen</button><span class="grow"></span><button class="btn danger" id="pub-rm">${I('trash')}Wieder privat machen</button>` : ''}</div>`;
    $('#pub-save').onclick = ev => {
      const b = ev.currentTarget; b.disabled = true; b.innerHTML = '<span class="spin"></span>Richtet Tunnel ein ...';
      api('POST', '/settings/panel', { zoneId: $('#pub-zone').value, prefix: $('#pub-pre').value })
        .then(async r => { toast('Panel ist erreichbar unter ' + r.url, 'good'); pubOpen = false; drawCF(await api('GET', '/settings')); })
        .catch(e => { fail(e); b.disabled = false; b.innerHTML = I('check') + (p ? 'Ändern' : 'Öffentlich machen'); });
    };
    const cl = $('#pub-close'); if (cl) cl.onclick = () => { pubOpen = false; drawPub(d); };
    const rm = $('#pub-rm');
    if (rm) rm.onclick = async () => {
      if (!await confirmBox('Panel wieder privat machen?', 'Tunnel und DNS-Eintrag werden bei Cloudflare gelöscht. Im Heimnetz bleibt das Panel erreichbar.', { ok: 'Privat machen', danger: true })) return;
      api('POST', '/settings/panel', { remove: true }).then(async () => { toast('Panel ist wieder nur im Heimnetz erreichbar'); pubOpen = false; drawCF(await api('GET', '/settings')); }).catch(fail);
    };
  };
  drawCF(d);
}

// Adresse eines Servers für Spieler (Domain falls vorhanden).
function serverAddr(s) {
  if (s.domain?.zoneName) {
    const fq = !s.domain.prefix || s.domain.prefix === '@' ? s.domain.zoneName : s.domain.prefix + '.' + s.domain.zoneName;
    return s.domain.srv ? fq : `${fq}:${s.ports?.[0] ?? ''}`;
  }
  return `${s.alias || host}:${s.ports?.[0] ?? '–'}`;
}

/* ---------- Import aus Pterodactyl ---------- */
function importView(box) {
  const form = (err = '') => {
    box.innerHTML = `<div class="panel"><div class="ph">${thumb('', 'Pterodactyl', 'sq sm')}<h2>Aus Pterodactyl übernehmen</h2><span class="faint" style="font-size:13px">geht auch mit Pelican</span></div><div class="pb stack">
      <p class="muted" style="margin:0">Serverwerk liest deine Server aus Pterodactyl aus und legt sie hier mit gleichen Ports, Grenzen, Variablen und allen Dateien an. In Pterodactyl wird nichts gelöscht.</p>
      <ol class="steps-list">
        ${tt(`<li>In Pterodactyl oben rechts auf dein Konto &rsaquo; <b>API Credentials</b> &rsaquo; Beschreibung eintragen &rsaquo; <b>Create</b>. Der Key beginnt mit <span class="mono">ptlc_</span>.</li>
        <li>Optional, aber genauer: Als Admin unter <b>Admin &rsaquo; Application API &rsaquo; Create New</b> einen Key mit <span class="mono">Servers: Read</span> und <span class="mono">Nests: Read</span> anlegen (beginnt mit <span class="mono">ptla_</span>). Dann wird das exakte Egg übernommen statt eines gleichnamigen aus dem Katalog.</li>
        <li>Die Server in Pterodactyl vorher <b>stoppen</b>, damit die Welt beim Kopieren nicht verändert wird.</li></ol>`, `<li>In Pterodactyl, click your account at the top right &rsaquo; <b>API Credentials</b> &rsaquo; enter a description &rsaquo; <b>Create</b>. The key starts with <span class="mono">ptlc_</span>.</li>
        <li>Optional, but more exact: as admin, create a key under <b>Admin &rsaquo; Application API &rsaquo; Create New</b> with <span class="mono">Servers: Read</span> and <span class="mono">Nests: Read</span> (starts with <span class="mono">ptla_</span>). Then the exact egg is used instead of one with the same name from the catalog.</li>
        <li><b>Stop</b> the servers in Pterodactyl first, so the world doesn't change while copying.</li></ol>`)}
      <form class="stack" id="imp-f">
        <label class="f">Adresse von Pterodactyl<input class="input mono" name="url" placeholder="https://panel.example.com" required></label>
        <div class="grid2">
          <label class="f">Client-API-Key<input class="input mono" name="ck" placeholder="ptlc_..." required autocomplete="off"></label>
          <label class="f">Application-API-Key (optional)<input class="input mono" name="ak" placeholder="ptla_..." autocomplete="off"></label>
        </div>
        ${err ? `<div class="err">${esc(err)}</div>` : ''}
        <div><button class="btn primary">${I('search')}Server suchen</button></div>
      </form>
      <p class="faint" style="margin:0;font-size:12.5px">Läuft Pterodactyl auf diesem Rechner, werden die Dateien direkt kopiert. Sonst packt Pterodactyl sie in ein Archiv, das Serverwerk herunterlädt (bei großen Welten dauert das).</p>
    </div></div>`;
    $('#imp-f').onsubmit = async e => {
      e.preventDefault();
      const f = e.target, b = $('button', f);
      b.disabled = true; b.innerHTML = '<span class="spin"></span>Sucht ...';
      try { list(await api('POST', '/import/pterodactyl/scan', { url: f.url.value, clientKey: f.ck.value, appKey: f.ak.value })); }
      catch (err) { form(err.message); const n = $('#imp-f'); n.url.value = f.url.value; n.ck.value = f.ck.value; n.ak.value = f.ak.value; }
    };
  };
  const list = async r => {
    const [eggsR, users, pinfo] = await Promise.all([api('GET', '/eggs'), api('GET', '/usernames'), api('GET', '/ports/free')]);
    const eggs = (eggsR.eggs || []).filter(e => e.compatible !== false);
    const badge = m => ({ installed: '<span class="tag acc">Egg vorhanden</span>', catalog: '<span class="tag acc">Egg aus Katalog</span>', app: '<span class="tag acc">exaktes Egg</span>' }[m?.source] || '');
    if (!r.servers.length) { box.innerHTML = `<div class="empty">Mit diesem Key sind keine Server sichtbar.<div style="margin-top:14px"><button class="btn" id="imp-back">${I('back')}Zurück</button></div></div>`; $('#imp-back').onclick = () => form(); return; }
    box.innerHTML = `<div class="panel"><div class="ph"><h2 class="grow">${r.servers.length} Server gefunden</h2><button class="btn small ghost" id="imp-back">${I('back')}Andere Zugangsdaten</button></div>
      ${!pinfo.configured ? `<div class="pb" style="padding-bottom:0"><div class="banner bad" style="margin:0"><span class="grow">${tt('Lege zuerst unter Allocations deine Ports fest, sonst kann nichts importiert werden.', 'First set your ports under Allocations, otherwise nothing can be imported.')}</span><a class="btn small" href="#/allocations">Allocations</a></div></div>` : ''}
      <div class="implist">${r.servers.map((s, i) => `<label class="improw ${s.imported ? 'done' : ''}">
        <input type="checkbox" data-i="${i}" ${s.imported ? 'disabled' : s.match?.source !== 'none' ? 'checked' : ''}>
        <div class="grow" style="min-width:0"><b class="ell" style="display:block">${esc(s.name)}</b>
          <div class="row wrap" style="gap:6px;margin-top:4px"><span class="tag">${esc(s.eggName || 'Egg?')}</span>${badge(s.match)}
            <span class="tag mono">${s.ports.join(', ') || 'kein Port'}</span><span class="tag">${s.memory ? s.memory + ' MB' : 'RAM unbegrenzt'}</span>
            ${s.diskBytes ? `<span class="tag">${bytes(s.diskBytes)}</span>` : ''}${s.local ? '<span class="tag acc">Dateien lokal</span>' : ''}
            ${s.state && s.state !== 'offline' ? `<span class="tag" style="color:var(--warn)">läuft noch</span>` : ''}
            ${s.imported ? `<a class="tag acc" href="#/s/${s.imported}">schon importiert</a>` : ''}</div>
          ${s.match?.source === 'none' && !s.imported ? `<select class="input" data-egg="${i}" style="margin-top:8px;max-width:340px;height:36px"><option value="">Egg wählen (kein Treffer für „${esc(s.eggName)}“)</option>${eggs.map(e => `<option value="${e.id}">${esc(e.name)}</option>`).join('')}</select>` : ''}
        </div></label>`).join('')}</div>
      <div class="pb stack" style="border-top:1px solid var(--border)">
        <div class="grid2">
          <label class="f">Besitzer in Serverwerk<select class="input" id="imp-owner">${users.map(u => `<option value="${u.id}" ${u.id === S.me.id ? 'selected' : ''}>${esc(u.name)}</option>`).join('')}</select></label>
        </div>
        <label class="check"><input type="checkbox" id="imp-keep" checked>Ports übernehmen, wenn sie in den Allocations stehen und frei sind (sonst neue vergeben)</label>
        <div class="row wrap"><button class="btn primary" id="imp-go">${I('download')}Ausgewählte importieren</button><span class="faint" id="imp-n" style="font-size:13px"></span></div>
      </div></div>`;
    $('#imp-back').onclick = () => form();
    const count = () => { const n = $$('[data-i]:checked').length; $('#imp-n').textContent = `${n} ausgewählt`; $('#imp-go').disabled = !n || !pinfo.configured; };
    $$('[data-i]').forEach(x => x.onchange = count);
    $$('[data-egg]').forEach(x => x.onchange = () => { const c = $(`[data-i="${x.dataset.egg}"]`); if (x.value) c.checked = true; count(); });
    count();
    $('#imp-go').onclick = async ev => {
      const items = $$('[data-i]:checked').map(c => {
        const s = r.servers[c.dataset.i];
        return { identifier: s.identifier, eggId: $(`[data-egg="${c.dataset.i}"]`)?.value || '', keepPorts: $('#imp-keep').checked };
      });
      const b = ev.currentTarget; b.disabled = true; b.innerHTML = '<span class="spin"></span>Legt an ...';
      try {
        const res = await api('POST', '/import/pterodactyl/run', { scan: r.scan, owner: $('#imp-owner').value, servers: items });
        box.innerHTML = `<div class="panel"><div class="ph"><h2>Import gestartet</h2></div><div class="pb stack">
          <p class="muted" style="margin:0">Die Dateien werden im Hintergrund übertragen. Den Fortschritt siehst du in der Konsole des jeweiligen Servers.</p>
          ${res.results.map(x => `<div class="row">${x.id ? I('check') : I('kill')}<span class="grow">${x.id ? `<a href="#/s/${x.id}/konsole">${esc(x.name)}</a>` : esc(x.name)}</span>${x.error ? `<span class="err">${esc(x.error)}</span>` : '<span class="tag acc">läuft</span>'}</div>`).join('')}
          <div><a class="btn primary" href="#/">${I('server')}Zur Server-Übersicht</a></div></div></div>`;
      } catch (err) { fail(err); b.disabled = false; b.innerHTML = I('download') + 'Ausgewählte importieren'; }
    };
  };
  form();
}

/* ---------- Allocations ---------- */
async function pageAllocations(page) {
  page.innerHTML = `<div class="head"><div><h1>Allocations</h1><div class="sub term-label">$ cat /etc/serverwerk/allocations</div></div></div><div id="alb"><div class="muted">Lädt ...</div></div>`;
  let d = await api('GET', '/allocations').catch(fail); if (!d) return;
  let groups = d.groups.length ? d.groups.map(g => ({ ...g })) : [{ ports: '', alias: '' }];
  const draw = () => {
    const used = d.ports.filter(p => p.server).length, busy = d.ports.filter(p => !p.server && p.busy).length, free = d.ports.length - used - busy;
    $('#alb').innerHTML = `<div class="stack">
      <div class="panel"><div class="ph"><h2>Freigegebene Ports</h2><span class="grow"></span>${d.ports.length ? `<span class="tag acc">${free} frei</span><span class="tag">${used} belegt</span>` : '<span class="tag" style="color:var(--danger)">noch keine</span>'}</div>
        <div class="pb stack">
          <p class="muted" style="margin:0">Trag hier die Ports ein, die in deinem Router an diesen Rechner weitergeleitet sind. Neue Server bekommen automatisch einen freien Port daraus, in der Reihenfolge der Liste. Ohne Einträge lassen sich keine Server anlegen.</p>
          <div class="alrows">
            <div class="alrow head"><span>Ports</span><span>Alias (optional)</span><span></span></div>
            ${groups.map((g, i) => `<div class="alrow">
              <input class="input mono" data-i="${i}" data-k="ports" value="${esc(g.ports)}" placeholder="25565-25575, 19132">
              <input class="input mono" data-i="${i}" data-k="alias" value="${esc(g.alias)}" placeholder="example.com">
              <button class="ib danger" data-rm="${i}" title="Bereich entfernen">${I('trash')}</button></div>`).join('')}
          </div>
          <p class="faint" style="margin:0;font-size:12.5px">Der Alias ist die Adresse, unter der die Server von außen erreichbar sind, z. B. die DynDNS-Adresse deines Routers oder deine öffentliche IP. Er wird dann überall statt der Heimnetz-Adresse angezeigt. Hat ein Server eine Cloudflare-Domain, wird die angezeigt.</p>
          <div class="row wrap"><button class="btn" id="al-add">${I('plus')}Bereich</button><span class="grow"></span><button class="btn primary" id="al-save">${I('check')}Speichern</button></div>
        </div></div>
      ${d.ports.length ? `<div class="panel"><div class="ph"><h2>Übersicht</h2><span class="grow"></span><span class="legend"><i class="lg free"></i>frei <i class="lg used"></i>Server <i class="lg busy"></i>anderes Programm</span></div>
        <div class="pb portgrid">${d.ports.slice(0, 600).map(p => p.server
          ? `<a class="pchip used" href="#/s/${p.server}" title="${esc(p.name)}"><b>${p.port}</b><span class="ell">${esc(p.name)}</span></a>`
          : `<div class="pchip ${p.busy ? 'busy' : 'free'}" title="${p.busy ? 'Port wird von einem anderen Programm auf diesem Rechner benutzt' : 'frei'}"><b>${p.port}</b><span>${p.busy ? 'belegt' : 'frei'}</span></div>`).join('')}
          ${d.ports.length > 600 ? `<div class="muted" style="grid-column:1/-1">… und ${d.ports.length - 600} weitere</div>` : ''}</div></div>` : ''}
      ${d.outside?.length ? `<div class="banner bad" style="margin:0"><span class="grow">${tt('Diese Server nutzen Ports außerhalb deiner Liste:', 'These servers use ports outside your list:')} ${d.outside.map(o => `<a class="notr" href="#/s/${o.server}">${esc(o.name)}</a> (${o.port})`).join(', ')}. ${tt('Sie laufen weiter, aber ändern lassen sich ihre Ports nur auf Ports aus der Liste.', 'They keep running, but their ports can only be changed to ports from the list.')}</span></div>` : ''}
    </div>`;
    $$('[data-k]').forEach(x => x.oninput = () => { groups[x.dataset.i][x.dataset.k] = x.value; });
    $$('[data-rm]').forEach(b => b.onclick = () => { groups.splice(+b.dataset.rm, 1); if (!groups.length) groups.push({ ports: '', alias: '' }); draw(); });
    $('#al-add').onclick = () => { groups.push({ ports: '', alias: '' }); draw(); $$('[data-k=ports]').pop().focus(); };
    $('#al-save').onclick = () => api('PUT', '/allocations', groups).then(r => {
      d = r; groups = r.groups.length ? r.groups.map(g => ({ ...g })) : [{ ports: '', alias: '' }];
      toast(`${r.ports.length} Ports gespeichert`, 'good'); draw();
    }).catch(fail);
  };
  draw();
}

/* ---------- Benutzer ---------- */
async function pageUsers(page) {
  page.innerHTML = `<div class="head"><div><h1>Benutzer</h1><div class="sub term-label">$ cat /etc/passwd</div></div>
    <div class="tools"><button class="btn primary" id="nu">${I('plus')}Benutzer</button></div></div><div class="panel" id="ul"></div>`;
  const load = async () => {
    const list = await api('GET', '/users').catch(fail) || [];
    $('#ul').innerHTML = `<table class="table"><thead><tr><th>Name</th><th>Rolle</th><th>Server</th><th>Angelegt</th><th></th></tr></thead><tbody>${list.map(u => `
      <tr><td><div class="row"><span class="avatar">${esc(u.name[0])}</span><b class="notr">${esc(u.name)}</b>${u.id === S.me.id ? '<span class="tag">du</span>' : ''}</div></td>
      <td>${u.admin ? '<span class="tag acc">Admin</span>' : '<span class="tag">Benutzer</span>'}</td><td>${u.servers}</td><td class="muted">${when(u.created)}</td>
      <td class="r"><button class="ib" data-m="${u.id}">${I('dots')}</button></td></tr>`).join('')}</tbody></table>`;
    $$('[data-m]').forEach(b => b.onclick = () => {
      const u = list.find(x => x.id === b.dataset.m);
      popMenu(b, [
        { label: 'Passwort setzen', icon: 'key', run: async () => { const pw = await promptBox('Neues Passwort', `für ${u.name} (mind. 8 Zeichen)`, '', { type: 'password' }); if (pw) api('PATCH', '/users/' + u.id, { password: pw }).then(() => toast('Passwort gesetzt', 'good')).catch(fail); } },
        u.id !== S.me.id && { label: u.admin ? 'Admin-Rechte entziehen' : 'Zum Admin machen', icon: 'users', run: () => api('PATCH', '/users/' + u.id, { admin: !u.admin }).then(load).catch(fail) },
        u.id !== S.me.id && { label: 'Löschen', icon: 'trash', danger: true, run: async () => { if (await confirmBox('Benutzer löschen?', u.name, { ok: 'Löschen', danger: true })) api('DELETE', '/users/' + u.id).then(load).catch(fail); } },
      ]);
    });
  };
  $('#nu').onclick = () => {
    const m = modal(`<h3>Neuer Benutzer</h3><form class="stack" style="margin-top:14px">
      <label class="f">Name<input class="input" name="name" required autocomplete="off"></label>
      <label class="f">Passwort<input class="input" name="pw" type="password" required minlength="8" autocomplete="new-password"><small>mindestens 8 Zeichen</small></label>
      <label class="check"><input type="checkbox" name="admin">Admin (darf alles, auch Server anlegen)</label>
      <div class="acts"><button type="button" class="btn" data-close>Abbrechen</button><button class="btn primary">Anlegen</button></div></form>`);
    $('form', m.el).onsubmit = e => {
      e.preventDefault(); const f = e.target;
      api('POST', '/users', { name: f.name.value, password: f.pw.value, admin: f.admin.checked }).then(() => { m.close(); load(); toast('Benutzer angelegt', 'good'); }).catch(fail);
    };
  };
  load();
}

/* ---------- System ---------- */
async function pageSystem(page) {
  page.innerHTML = `<div class="head"><div><h1>System</h1><div class="sub term-label">$ uname -a</div></div></div><div id="sys"><div class="muted">Lädt ...</div></div>`;
  const load = async () => {
    const d = await api('GET', '/system').catch(fail); if (!d || !$('#sys')) return;
    const h = d.host || {};
    $('#sys').innerHTML = `<div class="cards" style="margin-bottom:14px">
      <div class="tile"><small>CPU (${d.cpus || '?'} Kerne)</small><b>${(h.cpu || 0).toFixed(1)} %</b><div class="bars" style="grid-template-columns:1fr;margin-top:8px">${barHtml('Last', h.cpu || 0, 100, h.load || '')}</div></div>
      <div class="tile"><small>Arbeitsspeicher</small><b>${bytes(h.memUsed)}<span> / ${bytes(h.memTotal)}</span></b><div class="bars" style="grid-template-columns:1fr;margin-top:8px">${barHtml('belegt', h.memUsed, h.memTotal, Math.round(h.memUsed / h.memTotal * 100) + ' %')}</div></div>
      <div class="tile"><small>Speicherplatz (Datenordner)</small><b>${bytes((h.diskTotal || 0) - (h.diskFree || 0))}<span> / ${bytes(h.diskTotal)}</span></b><div class="bars" style="grid-template-columns:1fr;margin-top:8px">${barHtml('belegt', (h.diskTotal || 0) - (h.diskFree || 0), h.diskTotal, bytes(h.diskFree) + ' frei')}</div></div>
    </div>
    <div class="panel"><dl class="pb kv">
      <dt>Rechner</dt><dd>${esc(d.hostname || '–')}</dd><dt>System</dt><dd>${esc(d.os)} (${esc(d.arch)}), Kernel ${esc(d.kernel || '–')}</dd>
      <dt>Docker</dt><dd>${d.docker ? esc(d.docker) : `<span style="color:var(--danger)">${esc(d.dockerError || 'nicht erreichbar')}</span>`}</dd>
      <dt>Serverwerk</dt><dd>${esc(d.version)}</dd><dt>Datenordner</dt><dd class="mono">${esc(d.data)}</dd><dt>Läuft seit</dt><dd>${dur(h.uptime)}</dd>
      <dt>Aktualisieren</dt><dd class="mono" style="font-size:12.5px">Installationsbefehl einfach erneut ausführen</dd>
      <dt>Passwort vergessen</dt><dd class="mono" style="font-size:12.5px">sudo serverwerk passwort &lt;name&gt;</dd></dl></div>`;
  };
  load();
  const t = setInterval(load, 4000);
  onLeave(() => clearInterval(t));
}

/* ---------- Konto ---------- */
function pageAccount(page) {
  page.innerHTML = `<div class="head"><div><h1>Konto</h1><div class="sub term-label">$ passwd ${esc(S.me.name)}</div></div></div>
    <div class="panel" style="max-width:520px"><form class="pb stack">
    <label class="f">Altes Passwort<input class="input" type="password" name="old" required autocomplete="current-password"></label>
    <label class="f">Neues Passwort<input class="input" type="password" name="neu" required minlength="8" autocomplete="new-password"></label>
    <div><button class="btn primary">Passwort ändern</button></div></form></div>
    <div class="panel" style="max-width:520px;margin-top:14px"><div class="pb row"><span class="grow">Erscheinungsbild</span>
      <div class="chips">${['auto', 'dark', 'light'].map(t => `<button class="chip" data-t="${t}">${{ auto: 'System', dark: 'Dunkel', light: 'Hell' }[t]}</button>`).join('')}</div></div></div>`;
  $('form', page).onsubmit = e => {
    e.preventDefault(); const f = e.target;
    api('POST', '/me/password', { old: f.old.value, new: f.neu.value }).then(() => { toast('Passwort geändert', 'good'); f.reset(); }).catch(fail);
  };
  const cur = localStorage.getItem('sw-theme') || 'auto';
  $$('[data-t]').forEach(b => { b.classList.toggle('on', b.dataset.t === cur); b.onclick = () => { setTheme(b.dataset.t); pageAccount(page); }; });
}
function setTheme(t) {
  try { localStorage.setItem('sw-theme', t); } catch { }
  if (t === 'auto') document.documentElement.removeAttribute('data-theme');
  else document.documentElement.dataset.theme = t;
}
try { setTheme(localStorage.getItem('sw-theme') || 'auto'); } catch { }

boot();
