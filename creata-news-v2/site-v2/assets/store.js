/* Creata News — общее хранилище сайта и админки.
   Сейчас это localStorage браузера (демо без бэкенда).
   При подключении API заменить load/save на запросы к серверу — интерфейс CN.* останется тем же. */
(function () {
  var KEY = "creata-news-state";
  var PKEY = "creata-news-prefs";

  function clone(o) { return JSON.parse(JSON.stringify(o)); }
  function load() {
    try {
      var raw = localStorage.getItem(KEY);
      if (raw) {
        var d = JSON.parse(raw);
        if (d && d.version === window.CN_SEED.version) return d;
      }
    } catch (e) {}
    return clone(window.CN_SEED);
  }
  function loadPrefs() {
    var def = { lang: "ru", level: "full", motion: true, character: true, theme: "auto", saved: [], liked: [], reminders: [], follows: [], hidden: [], profile: null, digest: null, polls: {}, game: null };
    try {
      var raw = localStorage.getItem(PKEY);
      if (raw) { var p = JSON.parse(raw); for (var k in def) if (!(k in p)) p[k] = def[k]; return p; }
    } catch (e) {}
    return def;
  }

  var CN = window.CN = {
    state: load(),
    prefs: loadPrefs(),
    save: function () { try { localStorage.setItem(KEY, JSON.stringify(CN.state)); } catch (e) {} },
    savePrefs: function () { try { localStorage.setItem(PKEY, JSON.stringify(CN.prefs)); } catch (e) {} },
    reset: function () { CN.state = clone(window.CN_SEED); CN.save(); },
    log: function (who, what) {
      CN.state.log.unshift({ t: new Date().toISOString(), who: who, what: what });
      CN.state.log = CN.state.log.slice(0, 200);
    },
    esc: function (s) {
      return String(s == null ? "" : s).replace(/[&<>"']/g, function (c) {
        return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
      });
    },
    vert: function (id) { return CN.state.verticals.find(function (v) { return v.id === id; }) || { id: id, name: id, layer: "world" }; },
    author: function (id) { return CN.state.authors.find(function (a) { return a.id === id; }) || { id: id, name: id, role: "" }; },
    article: function (key) { return CN.state.articles.find(function (a) { return a.id === key || a.slug === key; }); },
    opp: function (id) { return CN.state.opportunities.find(function (o) { return o.id === id; }); },
    published: function () {
      return CN.state.articles.filter(function (a) { return a.status === "published"; })
        .sort(function (a, b) { return new Date(b.published) - new Date(a.published); });
    },
    liveOpps: function () {
      var now = Date.now();
      return CN.state.opportunities.filter(function (o) { return o.status === "approved" && new Date(o.deadline) > now; })
        .sort(function (a, b) { return new Date(a.deadline) - new Date(b.deadline); });
    },
    months: ["января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"],
    fmtDate: function (iso, withTime) {
      if (!iso) return "—";
      var d = new Date(iso);
      if (isNaN(d)) return iso;
      var s = d.getDate() + " " + CN.months[d.getMonth()] + " " + d.getFullYear();
      if (withTime) s += ", " + String(d.getHours()).padStart(2, "0") + ":" + String(d.getMinutes()).padStart(2, "0");
      return s;
    },
    timeLeft: function (iso) {
      var ms = new Date(iso) - Date.now();
      if (ms <= 0) return { done: true, d: 0, h: 0, m: 0 };
      return { done: false, d: Math.floor(ms / 864e5), h: Math.floor(ms / 36e5) % 24, m: Math.floor(ms / 6e4) % 60 };
    },
    initials: function (name) { return (name || "?").trim().charAt(0).toUpperCase(); },
    uid: function (p) { return (p || "id") + "-" + Math.random().toString(36).slice(2, 8); }
  };

  /* Фото-библиотека: все снимки, вшитые в сайт (из статей-источников и creata.news). */
  CN.photoLib = function () {
    var seen = {}, out = [];
    CN.state.articles.forEach(function (a) {
      [a.photo && { src: a.photo.url, cap: a.photo.credit }].concat(a.photos || []).forEach(function (p) {
        if (p && p.src && p.src.indexOf("assets/photos/") === 0 && !seen[p.src]) { seen[p.src] = 1; out.push({ src: p.src, cap: p.cap || "" }); }
      });
    });
    return out;
  };
  CN.illoTypes = [];
  CN.illo = function () { return ""; };
  /* Обложка материала — реальное фото с подписью */
  CN.cover = function (a, cls) {
    var url = a.photo && a.photo.url;
    if (!url) return '<figure class="cover cover-empty ' + (cls || "") + '"><figcaption>Нет фото</figcaption></figure>';
    return '<figure class="cover ' + (cls || "") + '"><img class="cover-photo" src="' + CN.esc(url) + '" alt="" loading="lazy">' +
      (a.photo.credit ? '<figcaption>' + CN.esc(a.photo.credit) + "</figcaption>" : "") + "</figure>";
  };
})();
