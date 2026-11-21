/* Creata News — редакционная админка. Данные общие с сайтом (store.js). */
(function () {
  var CN = window.CN, esc = CN.esc, S = function () { return CN.state; };
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };

  var COLS = [
    ["new", "Новые"], ["writing", "AI пишет"], ["review", "На проверке"], ["rework", "На доработке"],
    ["approved", "Одобрено"], ["published", "Опубликовано"], ["rejected", "Отклонено"]
  ];
  var STATUS = {}; COLS.forEach(function (c) { STATUS[c[0]] = c[1]; });
  var STCLS = { published: "ok", approved: "ok", rework: "warn", rejected: "bad", review: "", writing: "out", new: "out" };
  var ROLES = [
    { id: "amina", label: "Амина · главный редактор", chief: true },
    { id: "merey", label: "Мерей · выпускающий редактор" },
    { id: "tamiris", label: "Тамирис · редактор вертикали" },
    { id: "elnura", label: "Эльнура · модератор" },
    { id: "alisher", label: "Алишер · SMM" }
  ];
  var ui = { role: "amina", vert: "all", layer: "all", q: "", tab: "telegram" };
  try { var saved = JSON.parse(localStorage.getItem("creata-news-admin") || "{}"); for (var k in saved) ui[k] = saved[k]; } catch (e) {}
  function saveUi() { try { localStorage.setItem("creata-news-admin", JSON.stringify({ role: ui.role, vert: ui.vert, layer: ui.layer })); } catch (e) {} }
  function role() { return ROLES.find(function (r) { return r.id === ui.role; }) || ROLES[0]; }
  function me() { return CN.author(ui.role); }

  function toast(m) { var o = $(".toast"); if (o) o.remove(); var el = document.createElement("div"); el.className = "toast"; el.setAttribute("role", "status"); el.textContent = m; document.body.appendChild(el); setTimeout(function () { el.remove(); }, 2600); }
  function commit(what) { if (what) CN.log(me().name, what); CN.save(); }
  function chip(st) { return '<span class="chip ' + (STCLS[st] || "") + '">' + esc(STATUS[st] || st) + "</span>"; }
  function filtered(list) {
    return list.filter(function (a) {
      if (ui.vert !== "all" && a.vertical !== ui.vert) return false;
      if (ui.layer !== "all" && a.layer !== ui.layer) return false;
      if (ui.q && (a.title + " " + (a.lead || "")).toLowerCase().indexOf(ui.q.toLowerCase()) < 0) return false;
      return true;
    });
  }
  function canPublish(a) {
    if (a.layer === "eco" && !role().chief) return "«Экосистему Creata» публикует только главный редактор";
    if (a.minors && !a.consent) return "Нет письменного согласия родителей на фото несовершеннолетних";
    if (!a.title || !a.lead) return "Нужны заголовок и лид";
    return "";
  }

  /* ---------- Каркас ---------- */
  var NAV = [
    ["", "Редакция"], ["today", "Сегодня"], ["bank", "Банк новостей"], ["mine", "Материалы"],
    ["", "Проверка"], ["opps", "Возможности"], ["community", "Сообщество"],
    ["", "Выпуск"], ["social", "Соцсети"], ["paper", "Газета"],
    ["", "Команда и данные"], ["team", "Задачи стажёров"], ["analytics", "Аналитика"], ["sources", "Источники и настройки"], ["log", "Журнал"]
  ];
  function counts() {
    return {
      today: S().articles.filter(function (a) { return ["new", "review", "rework"].indexOf(a.status) > -1; }).length,
      bank: S().bank.filter(function (b) { return !b.taken; }).length,
      opps: S().opportunities.filter(function (o) { return o.status === "pending"; }).length,
      community: S().comments.filter(function (c) { return c.status === "pending"; }).length + S().reports.filter(function (r) { return r.status === "new"; }).length
    };
  }
  function shell() {
    var c = counts();
    $("#side").innerHTML = '<div class="brand"><span class="logo">CR<span class="e" aria-hidden="true"></span>ATA</span><small>News · редакция</small></div>' +
      NAV.map(function (n) {
        if (!n[0]) return '<div class="grp">' + n[1] + "</div>";
        return '<a class="item" href="#' + n[0] + '" data-nav="' + n[0] + '"><span>' + n[1] + "</span>" + (c[n[0]] ? '<span class="cnt num">' + c[n[0]] + "</span>" : "") + "</a>";
      }).join("") +
      '<div class="foot"><a class="btn sm" href="index.html">Открыть сайт ↗</a><span>Демо: данные хранятся в этом браузере и сразу видны на сайте.</span></div>';
    var verts = S().verticals;
    var now = new Date();
    $("#top").innerHTML = '<button class="btn sm burger" data-burger aria-label="Меню">Меню</button>' +
      '<form class="search" data-search><label class="sr" for="q">Поиск по материалам</label><input id="q" placeholder="Поиск по материалам" value="' + esc(ui.q) + '"></form>' +
      '<label class="sr" for="fv">Вертикаль</label><select id="fv" data-filter="vert"><option value="all">Все вертикали</option>' + verts.map(function (v) { return '<option value="' + v.id + '"' + (ui.vert === v.id ? " selected" : "") + ">" + esc(v.name) + "</option>"; }).join("") + "</select>" +
      '<label class="sr" for="fl">Слой</label><select id="fl" data-filter="layer"><option value="all">Оба слоя</option><option value="world"' + (ui.layer === "world" ? " selected" : "") + '>Мир</option><option value="eco"' + (ui.layer === "eco" ? " selected" : "") + ">Экосистема</option></select>" +
      '<span class="date">' + CN.fmtDate(now.toISOString()) + "</span>" +
      '<span class="who"><span class="ava">' + CN.initials(me().name) + '</span><label class="sr" for="role">Роль</label><select id="role" data-role>' + ROLES.map(function (r) { return '<option value="' + r.id + '"' + (r.id === ui.role ? " selected" : "") + ">" + esc(r.label) + "</option>"; }).join("") + "</select></span>";
  }

  /* ---------- Сегодня: сводка + канбан дня (PRD 8.2) ---------- */
  function today() {
    var arts = filtered(S().articles);
    var pubToday = arts.filter(function (a) { return a.status === "published" && new Date(a.published).toDateString() === new Date().toDateString(); }).length;
    var c = counts();
    var alerts = [];
    if (c.community) alerts.push(['<b>' + c.community + "</b> в сообществе: комментарии и сообщения об ошибках ждут модерации", "community", "warn"]);
    if (c.opps) alerts.push(['<b>' + c.opps + "</b> возможности ждут проверки модератором", "opps", "warn"]);
    S().articles.filter(function (a) { return a.flags && a.flags.length && ["review", "rework"].indexOf(a.status) > -1; }).forEach(function (a) { alerts.push(["Проверка фактов: «" + esc(a.title) + "» — " + esc(a.flags[0]), "story-" + a.id, "bad"]); });
    var h = '<div class="ph"><div><h1>Сегодня</h1><p>08:00 AI собрал ночные материалы · 09:30 планёрка · 10:00–13:00 проверка · 13:00 Утро Creata · 17:00 Postiz · 18:00 вечерний дайджест</p></div>' +
      '<div class="row"><button class="btn" data-new>Новый материал</button><a class="btn primary" href="#bank">Взять из банка</a></div></div>';
    h += '<div class="summary"><div class="kpi"><b class="num">' + S().bank.length + '</b><span>найдено в банке</span></div><div class="kpi"><b class="num">' + arts.filter(function (a) { return a.status === "approved"; }).length + '</b><span>одобрено, ждёт публикации</span></div><div class="kpi"><b class="num">' + arts.filter(function (a) { return a.status === "rejected"; }).length + '</b><span>отклонено</span></div><div class="kpi"><b class="num">' + pubToday + "</b><span>опубликовано сегодня</span></div></div>";
    if (alerts.length) h += '<div class="alerts">' + alerts.map(function (a) { return '<div class="alert ' + a[2] + '"><span class="row" style="flex-wrap:nowrap"><span class="dotm"></span><span>' + a[0] + '</span></span><a class="btn sm" href="#' + a[1] + '">Открыть</a></div>'; }).join("") + "</div>";
    h += '<div class="board" aria-label="Канбан дня">' + COLS.map(function (col) {
      var list = arts.filter(function (a) { return a.status === col[0]; });
      return '<section class="col" data-col="' + col[0] + '"><div class="col-h"><span>' + col[1] + '</span><span class="n num">' + list.length + "</span></div>" + list.map(kcard).join("") + "</section>";
    }).join("") + "</div>";
    h += '<p class="small muted" style="margin-top:10px">Перетащи карточку в другую колонку или выбери статус в карточке. «Опубликовано» для «Экосистемы Creata» доступно только главному редактору.</p>';
    return h;
  }
  function kcard(a) {
    var au = CN.author(a.author), v = CN.vert(a.vertical);
    var score = a.score || (S().bank.find(function (b) { return b.taken === a.id; }) || {}).score;
    var since = a.status === "published" ? "в колонке с " + CN.fmtDate(a.published) : "";
    return '<article class="kc" draggable="true" data-id="' + a.id + '"><div class="meta"><span class="chip ' + (a.layer === "eco" ? "ink" : "out") + '">' + (a.layer === "eco" ? "Экосистема" : "Мир") + "</span><span>" + esc(v.name) + "</span>" + (score ? '<span class="score num">' + score + "</span>" : "") + "</div>" +
      '<a href="#story-' + a.id + '"><h4>' + esc(a.title || "Без заголовка") + "</h4></a>" +
      '<div class="meta"><span>Источников: ' + (a.sources || 0) + "</span>" + (a.demo ? '<span class="chip out">демо</span>' : "") + "</div>" +
      (a.flags && a.flags.length ? '<div class="flags">' + a.flags.map(function (f) { return "<span>⚑ " + esc(f) + "</span>"; }).join("") + "</div>" : "") +
      (a.reworkNote && a.status === "rework" ? '<div class="small muted">Комментарий: ' + esc(a.reworkNote) + "</div>" : "") +
      '<div class="foot"><span class="row" style="gap:6px"><span class="ava" style="width:22px;height:22px;font-size:10px">' + CN.initials(au.name) + '</span><span class="small muted">' + esc(au.name) + '</span></span><label class="sr" for="mv-' + a.id + '">Статус</label><select class="move" id="mv-' + a.id + '" data-move="' + a.id + '">' + COLS.map(function (c) { return '<option value="' + c[0] + '"' + (c[0] === a.status ? " selected" : "") + ">" + c[1] + "</option>"; }).join("") + "</select></div>" +
      (since ? '<span class="small muted">' + since + "</span>" : "") + "</article>";
  }
  function moveTo(id, st, note) {
    var a = CN.article(id); if (!a || a.status === st) return false;
    if (st === "published") {
      var why = canPublish(a);
      if (why) { toast(why); return false; }
      a.published = a.published || new Date().toISOString();
      if (!a.published || a.status !== "published") a.published = new Date().toISOString();
    }
    var from = a.status; a.status = st;
    if (note) a.reworkNote = note;
    commit("«" + a.title + "»: " + STATUS[from] + " → " + STATUS[st]);
    return true;
  }

  /* ---------- Банк новостей ---------- */
  function bank() {
    var list = S().bank.filter(function (b) { return (ui.vert === "all" || b.vertical === ui.vert) && (!ui.q || b.title.toLowerCase().indexOf(ui.q.toLowerCase()) > -1); })
      .sort(function (a, b) { return b.score - a.score; });
    return '<div class="ph"><div><h1>Банк новостей</h1><p>Всё, что собрал парсер: RSS, сайты, Telegram-каналы, пресс-релизы на любых языках. В канбан автоматически идёт только то, что выше порога скоринга.</p></div><span class="chip">Порог: 70</span></div>' +
      '<div class="tbl-wrap"><table><thead><tr><th>Скор</th><th>Материал</th><th>Источник</th><th>Язык</th><th>Вертикаль</th><th>Найдено</th><th></th></tr></thead><tbody>' +
      list.map(function (b) {
        return "<tr><td><span class=\"score num\">" + b.score + "</span></td><td><b>" + esc(b.title) + "</b>" + (b.demo ? ' <span class="chip out">демо</span>' : "") + "</td><td>" + esc(b.source) + "</td><td>" + b.lang.toUpperCase() + "</td><td>" + esc(CN.vert(b.vertical).name) + '</td><td class="num">' + b.found + "</td><td>" +
          (b.taken ? '<a class="btn sm" href="#story-' + b.taken + '">В работе →</a>' : '<button class="btn sm primary" data-take="' + b.id + '">Взять в работу</button>') + "</td></tr>";
      }).join("") + "</tbody></table></div>";
  }

  /* ---------- Все материалы ---------- */
  function mine() {
    var list = filtered(S().articles).slice().sort(function (a, b) { return new Date(b.published || 0) - new Date(a.published || 0); });
    return '<div class="ph"><div><h1>Материалы</h1><p>Собственные статьи, интервью и репортажи редакции. Архив с датой, автором и статусом согласования.</p></div><button class="btn primary" data-new>Новый материал</button></div>' +
      '<div class="tbl-wrap"><table><thead><tr><th>Материал</th><th>Слой</th><th>Вертикаль</th><th>Автор</th><th>Статус</th><th>Опубликовано</th><th></th></tr></thead><tbody>' +
      list.map(function (a) {
        return '<tr><td><a href="#story-' + a.id + '"><b>' + esc(a.title || "Без заголовка") + "</b></a>" + (a.demo ? ' <span class="chip out">демо</span>' : "") + "</td><td>" + (a.layer === "eco" ? "Экосистема" : "Мир") + "</td><td>" + esc(CN.vert(a.vertical).name) + "</td><td>" + esc(CN.author(a.author).name) + "</td><td>" + chip(a.status) + '</td><td class="num">' + (a.published ? CN.fmtDate(a.published) : "—") + '</td><td><a class="btn sm" href="#story-' + a.id + '">Открыть</a></td></tr>';
      }).join("") + "</tbody></table></div>";
  }

  /* ---------- Карточка материала: источники · статья · медиа и соцсети ---------- */
  function story(id) {
    var a = CN.article(id); if (!a) return '<div class="panel">Материал не найден. <a href="#today">К канбану</a></div>';
    var block = function (b, i) {
      var body;
      if (b.t === "p" || b.t === "h") body = '<textarea class="inp" data-blk="' + i + '" data-key="text">' + esc(b.text) + "</textarea>";
      else if (b.t === "quote") body = '<textarea class="inp" data-blk="' + i + '" data-key="text">' + esc(b.text) + '</textarea><input class="inp" data-blk="' + i + '" data-key="who" value="' + esc(b.who) + '" placeholder="Кто сказал">';
      else body = '<textarea class="inp" data-blk="' + i + '" data-key="json" spellcheck="false" style="font-family:ui-monospace,monospace;font-size:12.5px">' + esc(JSON.stringify(b, null, 1)) + "</textarea>";
      var names = { p: "Текст", h: "Подзаголовок", quote: "Цитата", stats: "Цифры", facts: "Факты", timeline: "Таймлайн", quiz: "Квиз", poll: "Опрос" };
      return '<div class="blk"><div class="bh"><span>' + (names[b.t] || b.t) + (b.lvl === "deep" ? " · только «Глубже»" : "") + '</span><span class="row" style="gap:4px"><button class="btn sm" data-bup="' + i + '" aria-label="Выше">↑</button><button class="btn sm" data-bdel="' + i + '" aria-label="Удалить блок">✕</button></span></div>' + body + "</div>";
    };
    var facts = [];
    a.blocks.forEach(function (b) {
      if (b.t === "stats") b.items.forEach(function (s) { facts.push([s.n + " — " + s.l, true]); });
      if (b.t === "quote") facts.push(["Цитата: " + b.who, true]);
    });
    (a.flags || []).forEach(function (f) { facts.push([f, false]); });
    var why = canPublish(a);
    var verts = S().verticals, authors = S().authors;
    var social = socialText(a);

    var left = '<div class="ed-left stack"><div class="panel"><h3>Источники</h3>' +
      (a.source ? '<div class="stack small"><div><b>' + esc(a.source.name) + "</b> · " + esc(a.source.author || "") + '<br><span class="muted">' + CN.fmtDate(a.source.date) + '</span></div><a class="btn sm" href="' + esc(a.source.url) + '" target="_blank" rel="noopener">Открыть оригинал ↗</a></div>' : '<p class="small muted">Свой материал редакции.</p>') +
      '<div class="field" style="margin-top:10px"><label for="src-url">Ссылка на первоисточник</label><input class="inp" id="src-url" data-src="url" value="' + esc(a.source ? a.source.url : "") + '" placeholder="https://"></div>' +
      '<div class="field"><label for="src-name">Название источника</label><input class="inp" id="src-name" data-src="name" value="' + esc(a.source ? a.source.name : "") + '"></div>' +
      '<div class="field"><label for="src-n">Источников всего</label><input class="inp" id="src-n" type="number" min="0" data-f="sources" value="' + (a.sources || 0) + '"></div></div>' +
      '<div class="panel"><h3>Факты и проверка</h3>' + (facts.length ? facts.map(function (f) { return '<div class="fact"><span class="mk ' + (f[1] ? "ok" : "warn") + '">' + (f[1] ? "✓" : "!") + "</span><span>" + esc(f[0]) + "</span></div>"; }).join("") : '<p class="small muted">AI ещё не извлёк факты.</p>') +
      ((a.sources || 0) < 2 ? '<p class="small" style="color:var(--warn);margin-top:8px">Для важных фактов нужно ≥ 2 источника. Сейчас: ' + (a.sources || 0) + ".</p>" : "") + "</div></div>";

    var center = '<div class="stack"><div class="panel stack"><div class="row" style="justify-content:space-between"><span class="row">' + chip(a.status) + (a.demo ? '<span class="chip out">демо</span>' : "") + '</span><span class="small muted">' + (a.published ? "Опубликовано " + CN.fmtDate(a.published, true) : "Не опубликовано") + "</span></div>" +
      '<label class="sr" for="ti">Заголовок</label><textarea class="inp title-inp" id="ti" rows="2" data-f="title" placeholder="Заголовок до 8 слов">' + esc(a.title) + "</textarea>" +
      '<div class="field"><label for="le">Лид</label><textarea class="inp" id="le" data-f="lead">' + esc(a.lead) + '</textarea></div>' +
      '<div class="field"><label for="tl">Коротко (уровень «Проще»)</label><textarea class="inp" id="tl" data-f="tldr">' + esc(a.tldr || "") + "</textarea></div>" +
      '<details class="field en-fields"' + (a.title_en ? "" : " open") + '><summary>English version (EN)' + (a.title_en ? " ✓" : " — не заполнено, на английском сайте материал не покажется") + '</summary>' +
      '<div class="field"><label for="ti-en">Title (EN)</label><textarea class="inp" id="ti-en" rows="2" data-f="title_en">' + esc(a.title_en || "") + "</textarea></div>" +
      '<div class="field"><label for="le-en">Lead (EN)</label><textarea class="inp" id="le-en" data-f="lead_en">' + esc(a.lead_en || "") + "</textarea></div>" +
      '<div class="field"><label for="tl-en">In brief (EN)</label><textarea class="inp" id="tl-en" data-f="tldr_en">' + esc(a.tldr_en || "") + "</textarea></div></details>" +
      '<div class="row"><div class="field" style="flex:1 1 140px"><label for="la">Метка</label><select class="sel" id="la" data-f="label">' + ["Редакция", "Экосистема Creata", "Партнёрский материал", "AI-подборка"].map(function (l) { return "<option" + (a.label === l ? " selected" : "") + ">" + l + "</option>"; }).join("") + "</select></div>" +
      '<div class="field" style="flex:1 1 140px"><label for="ly">Слой</label><select class="sel" id="ly" data-f="layer"><option value="world"' + (a.layer === "world" ? " selected" : "") + '>Мир</option><option value="eco"' + (a.layer === "eco" ? " selected" : "") + ">Экосистема Creata</option></select></div>" +
      '<div class="field" style="flex:1 1 160px"><label for="ve">Вертикаль</label><select class="sel" id="ve" data-f="vertical">' + verts.map(function (v) { return '<option value="' + v.id + '"' + (a.vertical === v.id ? " selected" : "") + ">" + esc(v.name) + "</option>"; }).join("") + "</select></div>" +
      '<div class="field" style="flex:1 1 140px"><label for="au">Автор</label><select class="sel" id="au" data-f="author">' + authors.map(function (x) { return '<option value="' + x.id + '"' + (a.author === x.id ? " selected" : "") + ">" + esc(x.name) + "</option>"; }).join("") + "</select></div></div></div>" +
      '<div class="panel stack"><div class="row" style="justify-content:space-between"><h3 style="margin:0">Блоки статьи</h3><span class="row" style="gap:4px">' +
      [["p", "Текст"], ["h", "Подзаголовок"], ["quote", "Цитата"], ["facts", "Факты"], ["timeline", "Таймлайн"], ["quiz", "Квиз"]].map(function (b) { return '<button class="btn sm" data-badd="' + b[0] + '">+ ' + b[1] + "</button>"; }).join("") + "</span></div>" +
      a.blocks.map(block).join("") + "</div></div>";

    var right = '<div class="stack"><div class="panel stack"><h3 style="margin:0">Изображение</h3><div class="preview">' + CN.cover(a) + "</div>" +
      '<p class="small muted" style="margin:0">Фото из библиотеки сайта (источники и creata.news):</p><div class="photos-pick" role="group" aria-label="Фото">' + CN.photoLib().map(function (p) { return '<button data-pick="' + esc(p.src) + '" aria-pressed="' + (!!a.photo && a.photo.url === p.src) + '" title="' + esc(p.cap) + '"><img src="' + esc(p.src) + '" alt="" loading="lazy"></button>'; }).join("") + "</div>" +
      '<div class="field"><label for="ph-url">Фото из источника (URL)</label><input class="inp" id="ph-url" data-photo="url" value="' + esc(a.photo && a.photo.url && a.photo.url.indexOf("data:") !== 0 ? a.photo.url : "") + '" placeholder="https://"></div>' +
      '<div class="field"><label for="ph-up">Или загрузить файл</label><input class="inp" id="ph-up" type="file" accept="image/*" data-upload></div>' +
      '<div class="field"><label for="ph-cr">Подпись к фото</label><input class="inp" id="ph-cr" data-photo="credit" value="' + esc(a.photo ? a.photo.credit : "") + '" placeholder="Фото: автор / издание"></div>' +
      sw("live", a.live, "Живое фото (3–5 с, без звука, в цикле)") + sw("minors", a.minors, "В кадре несовершеннолетние") +
      (a.minors ? sw("consent", a.consent, "Письменное согласие родителей получено") : "") +
      '<p class="small muted">AI не генерирует лица реальных людей. Сгенерированное всегда помечается как AI.</p></div>' +
      '<div class="panel"><h3>Версии для соцсетей</h3><div class="tabs" role="group">' + [["telegram", "Telegram"], ["instagram", "Instagram"], ["linkedin", "LinkedIn"]].map(function (t) { return '<button data-tab="' + t[0] + '" aria-pressed="' + (ui.tab === t[0]) + '">' + t[1] + "</button>"; }).join("") + "</div>" +
      '<div class="tg" data-social>' + esc(social[ui.tab]) + '</div><div class="row" style="margin-top:8px"><button class="btn sm" data-copy-social>Скопировать</button><button class="btn sm" data-postiz' + (a.status !== "published" ? " disabled" : "") + '>В Postiz</button></div></div></div>';

    var decide = '<div class="decide"><span class="row">' + (why ? '<span class="lock">' + esc(why) + "</span>" : '<span class="small muted">Все проверки для публикации пройдены</span>') + "</span>" +
      '<span class="row"><button class="btn bad" data-act="rejected">Отклонить</button><button class="btn warn" data-rework>Доработать AI</button><button class="btn ok" data-act="approved">Одобрить</button>' +
      '<button class="btn primary" data-act="published"' + (why ? " disabled" : "") + ">Опубликовать</button>" + (a.status === "published" ? '<a class="btn dark" href="index.html#a-' + a.slug + '" target="_blank">На сайте ↗</a>' : "") + "</span></div>";
    return '<div class="ph"><div><a class="small muted" href="#today">← Канбан дня</a><h1 style="margin-top:4px">Карточка материала</h1></div><span class="small muted">Изменения сохраняются автоматически</span></div><div class="editor">' + left + center + right + "</div>" + decide;
  }
  function sw(key, on, label) { return '<button class="sw" role="switch" aria-checked="' + !!on + '" data-sw="' + key + '"><i></i><span>' + esc(label) + "</span></button>"; }
  function socialText(a) {
    var link = "creata.news/ru/" + a.vertical + "/" + a.slug;
    return {
      telegram: (a.title || "") + "\n\n" + (a.tldr || a.lead || "") + "\n\n→ " + link + "\n\n◼ Creata News",
      instagram: "Слайд 1: " + (a.title || "") + "\nСлайд 2: " + (a.lead || "") + "\nСлайд 3: Что с этим сделать → ссылка в био\n\n#CreataNews #" + CN.vert(a.vertical).name.split(" ")[0].replace(/\W/g, ""),
      linkedin: (a.title || "") + "\n\n" + (a.lead || "") + "\n\nИсточник: " + (a.source ? a.source.name : "редакция Creata News") + "\nСсылка в первом комментарии."
    };
  }

  /* ---------- Возможности: очередь модерации ---------- */
  function opps() {
    var pend = S().opportunities.filter(function (o) { return o.status === "pending"; });
    var rest = S().opportunities.filter(function (o) { return o.status !== "pending"; });
    var CHK = [["link", "Официальная ссылка работает"], ["org", "Организатор существует"], ["deadline", "Дедлайн указан и в будущем"], ["terms", "Условия понятны"], ["fraud", "Нет признаков мошенничества"]];
    return '<div class="ph"><div><h1>Возможности</h1><p>Модератор проверяет каждую заявку за 24 часа по чек-листу. После дедлайна возможность снимается автоматически. Поле «продвигается» заложено, но выключено.</p></div></div>' +
      '<h3 style="margin-bottom:10px">Очередь · ' + pend.length + "</h3>" + (pend.length ? '<div class="team">' + pend.map(function (o) {
        var c = o.checklist || {}, all = CHK.every(function (k) { return c[k[0]]; });
        return '<div class="panel stack"><div class="row"><span class="chip">' + esc(o.type) + '</span><span class="small muted">до ' + CN.fmtDate(o.deadline) + "</span></div><h3 style=\"margin:0\">" + esc(o.title) + '</h3><p class="small muted">' + esc(o.org) + " · " + esc(o.conditions || "") + "</p>" +
          CHK.map(function (k) { return '<label class="task"><input type="checkbox" data-chk="' + o.id + '" data-k="' + k[0] + '"' + (c[k[0]] ? " checked" : "") + "><span>" + k[1] + "</span></label>"; }).join("") +
          '<div class="row"><button class="btn sm bad" data-opp="' + o.id + '" data-st="rejected">Отклонить</button><button class="btn sm primary" data-opp="' + o.id + '" data-st="approved"' + (all ? "" : " disabled") + ">Одобрить</button></div></div>";
      }).join("") + "</div>" : '<div class="panel muted">Очередь пуста.</div>') +
      '<h3 style="margin:22px 0 10px">Опубликованные и отклонённые</h3><div class="tbl-wrap"><table><thead><tr><th>Возможность</th><th>Тип</th><th>Организатор</th><th>Дедлайн</th><th>Статус</th></tr></thead><tbody>' +
      rest.map(function (o) { return "<tr><td><b>" + esc(o.title) + "</b>" + (o.demo ? ' <span class="chip out">демо</span>' : "") + "</td><td>" + esc(o.type) + "</td><td>" + esc(o.org) + '</td><td class="num">' + CN.fmtDate(o.deadline) + "</td><td>" + (o.status === "approved" ? (new Date(o.deadline) < Date.now() ? '<span class="chip out">Снято по дедлайну</span>' : '<span class="chip ok">На сайте</span>') : '<span class="chip bad">Отклонено</span>') + "</td></tr>"; }).join("") + "</tbody></table></div>";
  }

  /* ---------- Сообщество ---------- */
  function community() {
    var cm = S().comments.slice().sort(function (a, b) { return (a.status === "pending" ? -1 : 1) - (b.status === "pending" ? -1 : 1); });
    return '<div class="ph"><div><h1>Сообщество</h1><p>Комментарии проходят модерацию. Сообщения «Здесь ошибка» идут редактору материала, исправления публикуются в истории статьи.</p></div></div>' +
      '<h3 style="margin-bottom:10px">Сообщения об ошибках</h3><div class="tbl-wrap"><table><thead><tr><th>Материал</th><th>Что не так</th><th>Когда</th><th>Статус</th><th></th></tr></thead><tbody>' +
      (S().reports.length ? S().reports.map(function (r) { var a = CN.article(r.article) || {}; return '<tr><td><a href="#story-' + a.id + '"><b>' + esc(a.title) + "</b></a></td><td>" + esc(r.text) + '</td><td class="num">' + CN.fmtDate(r.date, true) + "</td><td>" + (r.status === "fixed" ? '<span class="chip ok">Исправлено</span>' : r.status === "dismissed" ? '<span class="chip out">Не ошибка</span>' : '<span class="chip warn">Новое</span>') + "</td><td>" + (r.status === "new" ? '<span class="row"><button class="btn sm ok" data-rep="' + r.id + '" data-st="fixed">Исправлено</button><button class="btn sm" data-rep="' + r.id + '" data-st="dismissed">Не ошибка</button></span>' : "") + "</td></tr>"; }).join("") : '<tr><td colspan="5" class="muted">Сообщений нет.</td></tr>') + "</tbody></table></div>" +
      '<h3 style="margin:22px 0 10px">Комментарии</h3><div class="tbl-wrap"><table><thead><tr><th>Автор</th><th>Комментарий</th><th>Материал</th><th>Статус</th><th></th></tr></thead><tbody>' +
      cm.map(function (c) { var a = CN.article(c.article) || {}; return "<tr><td><b>" + esc(c.name) + "</b></td><td>" + esc(c.text) + '</td><td class="small">' + esc(a.title) + "</td><td>" + (c.status === "published" ? '<span class="chip ok">Опубликован</span>' : c.status === "blocked" ? '<span class="chip bad">Скрыт</span>' : '<span class="chip warn">На модерации</span>') + "</td><td>" + (c.status === "pending" ? '<span class="row"><button class="btn sm ok" data-cm="' + c.id + '" data-st="published">Опубликовать</button><button class="btn sm bad" data-cm="' + c.id + '" data-st="blocked">Скрыть</button></span>' : "") + "</td></tr>"; }).join("") + "</tbody></table></div>";
  }

  /* ---------- Соцсети ---------- */
  function social() {
    var pub = CN.published();
    var CH = ["Telegram", "Instagram", "LinkedIn"];
    var st = S().socialStatus || (S().socialStatus = {});
    return '<div class="ph"><div><h1>Соцсети</h1><p>AI делает версии каждого одобренного материала, SMM ставит их в Postiz по медиаплану. На запуске: Telegram, Instagram, LinkedIn. Подключение Postiz API — вместе с бэкендом.</p></div></div>' +
      '<div class="tbl-wrap"><table><thead><tr><th>Материал</th>' + CH.map(function (c) { return "<th>" + c + "</th>"; }).join("") + "</tr></thead><tbody>" +
      pub.map(function (a) {
        return '<tr><td><a href="#story-' + a.id + '"><b>' + esc(a.title) + "</b></a></td>" + CH.map(function (c) {
          var k = a.id + ":" + c, v = st[k] || "draft";
          var lab = { draft: ["out", "Черновик"], queued: ["", "В Postiz · 17:00"], live: ["ok", "Вышло"] }[v];
          return '<td><button class="chip ' + lab[0] + '" style="border:0" data-soc="' + k + '">' + lab[1] + "</button></td>";
        }).join("") + "</tr>";
      }).join("") + '</tbody></table></div><p class="small muted" style="margin-top:8px">Нажми на статус, чтобы перевести: черновик → в Postiz → вышло.</p>';
  }

  /* ---------- Газета ---------- */
  function paper() {
    var iss = S().paper || (S().paper = { number: S().issue.number, items: CN.published().map(function (a) { return a.id; }) });
    var pub = CN.published();
    return '<div class="ph"><div><h1>Газета · №' + iss.number + " · " + esc(S().issue.month) + '</h1><p>Сборка номера из материалов месяца: отметь, что идёт в номер, и расставь полосы. Онлайн-версия обновляется сразу, PDF для печати соберёт сервер после подключения бэкенда.</p></div><button class="btn primary" data-pdf>Экспорт PDF</button></div>' +
      '<div class="editor" style="grid-template-columns:minmax(0,1fr) 320px"><div class="tbl-wrap"><table><thead><tr><th>В номер</th><th>Полоса</th><th>Материал</th><th></th></tr></thead><tbody>' +
      pub.map(function (a) {
        var i = iss.items.indexOf(a.id);
        return '<tr><td><input type="checkbox" aria-label="В номер" data-inissue="' + a.id + '"' + (i > -1 ? " checked" : "") + '></td><td class="num">' + (i > -1 ? i + 1 : "—") + "</td><td><b>" + esc(a.title) + '</b><br><span class="small muted">' + (a.layer === "eco" ? "Экосистема Creata" : "Мир") + " · " + esc(CN.vert(a.vertical).name) + "</span></td><td>" + (i > 0 ? '<button class="btn sm" data-up="' + a.id + '" aria-label="Выше">↑</button>' : "") + "</td></tr>";
      }).join("") + '</tbody></table></div><div class="panel stack"><h3 style="margin:0">Полосы</h3>' +
      iss.items.map(function (id, i) { var a = CN.article(id); return a ? '<div class="fact"><span class="mk ok">' + (i + 1) + '</span><span>' + esc(a.title) + "</span></div>" : ""; }).join("") + "</div></div>";
  }

  /* ---------- Задачи стажёров ---------- */
  function team() {
    return '<div class="ph"><div><h1>Задачи стажёров</h1><p>Каждый стажёр — редактор своих вертикалей. 8–11 часов в неделю.</p></div></div>' +
      '<form class="panel row" data-addtask style="margin-bottom:14px"><label class="sr" for="tw">Кому</label><select class="sel" id="tw" style="width:auto">' + S().authors.map(function (a) { return '<option value="' + a.id + '">' + esc(a.name) + "</option>"; }).join("") + '</select><label class="sr" for="tt">Задача</label><input class="inp" id="tt" required placeholder="Что сделать" style="flex:1 1 240px;width:auto"><label class="sr" for="td">Срок</label><input class="inp" id="td" type="date" style="width:auto"><button class="btn primary">Добавить</button></form>' +
      '<div class="team">' + S().authors.map(function (au) {
        var ts = S().tasks.filter(function (t) { return t.who === au.id; });
        return '<div class="panel"><div class="row" style="justify-content:space-between;margin-bottom:6px"><span class="row"><span class="ava">' + CN.initials(au.name) + "</span><span><b>" + esc(au.name) + '</b><br><span class="small muted">' + esc(au.role) + '</span></span></span><span class="chip">' + ts.filter(function (t) { return !t.done; }).length + "</span></div>" +
          (ts.length ? ts.map(function (t) { return '<label class="task' + (t.done ? " done" : "") + '"><input type="checkbox" data-task="' + t.id + '"' + (t.done ? " checked" : "") + "><span>" + esc(t.text) + '<br><span class="small muted">' + esc(CN.vert(t.vertical).name) + " · до " + CN.fmtDate(t.due) + "</span></span></label>"; }).join("") : '<p class="small muted">Задач нет.</p>') + "</div>";
      }).join("") + "</div>";
  }

  /* ---------- Аналитика ---------- */
  function analytics() {
    var arts = S().articles, pub = CN.published();
    var by = COLS.map(function (c) { return [c[1], arts.filter(function (a) { return a.status === c[0]; }).length]; });
    var maxS = Math.max.apply(null, by.map(function (b) { return b[1]; }).concat([1]));
    var top = pub.slice().sort(function (a, b) { return b.views - a.views; });
    var maxV = Math.max.apply(null, top.map(function (a) { return a.views; }).concat([1]));
    var vc = {}; pub.forEach(function (a) { var n = CN.vert(a.vertical).name; vc[n] = (vc[n] || 0) + 1; });
    var authorsC = S().authors.map(function (au) { return [au.name, pub.filter(function (a) { return a.author === au.id; }).length]; }).filter(function (x) { return x[1]; });
    var bars = function (rows, max, cls) { return '<div class="bars">' + rows.map(function (r) { return '<div class="bar ' + (cls || "") + '"><span>' + esc(r[0]) + '</span><span class="track"><span class="fill" style="width:' + Math.round(r[1] / max * 100) + '%"></span></span><b>' + r[1].toLocaleString("ru") + "</b></div>"; }).join("") + "</div>"; };
    return '<div class="ph"><div><h1>Аналитика</h1><p>Editorial и AI считаются по живым данным редакции. Просмотры и сохранения — демо-цифры: реальные придут из событий сайта после запуска.</p></div></div>' +
      '<div class="summary"><div class="kpi"><b class="num">' + pub.length + '</b><span>опубликовано</span></div><div class="kpi"><b class="num">' + S().bank.length + '</b><span>в банке новостей</span></div><div class="kpi"><b class="num">' + (S().bank.length ? Math.round(S().bank.filter(function (b) { return b.taken; }).length / S().bank.length * 100) : 0) + '%</b><span>банка взято в работу</span></div><div class="kpi"><b class="num">' + S().reports.length + "</b><span>сообщений об ошибках</span></div></div>" +
      '<div class="charts"><div class="panel"><h3>Editorial: материалы по этапам</h3>' + bars(by, maxS, "ink") + '</div><div class="panel"><h3>Content: просмотры <span class="chip out">демо</span></h3>' + bars(top.map(function (a) { return [a.title.slice(0, 28) + (a.title.length > 28 ? "…" : ""), a.views]; }), maxV) + "</div>" +
      '<div class="panel"><h3>Публикации по вертикалям</h3>' + bars(Object.keys(vc).map(function (k) { return [k, vc[k]]; }), Math.max.apply(null, Object.keys(vc).map(function (k) { return vc[k]; }).concat([1])), "ink") + '</div><div class="panel"><h3>Creator: материалы авторов</h3>' + bars(authorsC, Math.max.apply(null, authorsC.map(function (x) { return x[1]; }).concat([1])), "ink") + "</div></div>" +
      '<div class="panel" style="margin-top:14px"><h3>Главная метрика</h3><p class="small muted">Число людей, которые за неделю сделали целевое действие: сохранили или открыли возможность, написали комментарий, прошли интерактив. Появится после подключения событий сайта.</p></div>';
  }

  /* ---------- Источники и настройки ---------- */
  function sources() {
    var st = S().settings, w = st.weights;
    var sum = Object.keys(w).reduce(function (s, k) { return s + w[k]; }, 0);
    var FL = [["autopublish", "Автопубликация коротких новостей (после 4 недель < 10% правок)"], ["community", "Сообщество: доска команд и круги (Ф2)"], ["startups", "Карточки стартапов (Ф2)"], ["investors", "Кабинет инвестора (Ф3)"], ["whatsapp", "Дайджест в WhatsApp (Ф2)"], ["monetization", "Монетизация: продвижение и спонсоры (Ф3)"]];
    var WN = { I: "Интересы и стадия", B: "Поведение", Q: "Качество и источник", R: "Свежесть / дедлайн", D: "Разнообразие" };
    return '<div class="ph"><div><h1>Источники и настройки</h1><p>Справочник вертикалей, веса рекомендательной формулы, лимит AI-расходов, языки и флаги модулей. Всё меняется без разработки.</p></div></div>' +
      '<div class="charts"><div class="panel stack"><h3 style="margin:0">Формула ленты</h3><p class="small muted">Score = ' + Object.keys(w).map(function (k) { return w[k].toFixed(2) + "·" + k; }).join(" + ") + ' · сумма <b class="num">' + sum.toFixed(2) + "</b></p>" +
      Object.keys(w).map(function (k) { return '<div class="field"><label for="w-' + k + '">' + k + " — " + WN[k] + ' · <span class="num">' + w[k].toFixed(2) + '</span></label><input id="w-' + k + '" type="range" min="0" max="0.6" step="0.05" value="' + w[k] + '" data-w="' + k + '"></div>'; }).join("") + "</div>" +
      '<div class="panel stack"><h3 style="margin:0">AI и модули</h3><div class="field"><label for="lim">Дневной лимит AI-расходов, $</label><input class="inp" id="lim" type="number" min="0" value="' + st.aiDailyLimit + '" data-limit></div>' +
      FL.map(function (f) { return '<button class="sw" role="switch" aria-checked="' + !!st.flags[f[0]] + '" data-flag="' + f[0] + '"><i></i><span>' + f[1] + "</span></button>"; }).join("") + "</div>" +
      '<div class="panel"><h3>Языки</h3><div class="tbl-wrap" style="border:0"><table><tbody>' + Object.keys(st.languages).map(function (l) { var v = st.languages[l]; return "<tr><td><b>" + l.toUpperCase() + "</b></td><td>" + (v === "проверенный" ? '<span class="chip ok">' : v === "скоро" ? '<span class="chip out">' : '<span class="chip">') + esc(v) + "</span></td></tr>"; }).join("") + "</tbody></table></div><p class=\"small muted\">Язык становится «проверенным», когда появляется редактор-носитель.</p></div>" +
      '<div class="panel stack"><h3 style="margin:0">Вертикали</h3>' + ["world", "eco"].map(function (L) { return '<p class="small muted">' + (L === "world" ? "Мир" : "Экосистема Creata") + '</p><div class="row">' + S().verticals.filter(function (v) { return v.layer === L; }).map(function (v) { return '<span class="chip ' + (L === "eco" ? "ink" : "out") + '">' + esc(v.name) + "</span>"; }).join("") + "</div>"; }).join("") +
      '<form class="row" data-addvert><label class="sr" for="nv">Новая вертикаль</label><input class="inp" id="nv" placeholder="Новая вертикаль" required style="flex:1;width:auto"><select class="sel" id="nvl" style="width:auto" aria-label="Слой"><option value="world">Мир</option><option value="eco">Экосистема</option></select><button class="btn">Добавить</button></form></div></div>' +
      '<div class="panel row" style="margin-top:14px;justify-content:space-between"><span><b>Демо-данные</b><br><span class="small muted">Вернуть сайт и админку к исходному наполнению. Твои изменения в этом браузере удалятся.</span></span><span class="row" data-reset-area><button class="btn bad" data-reset>Сбросить демо-данные</button></span></div>';
  }

  function log() {
    return '<div class="ph"><div><h1>Журнал</h1><p>Все действия в админке: кто, что, когда.</p></div></div><div class="tbl-wrap"><table><thead><tr><th>Когда</th><th>Кто</th><th>Что</th></tr></thead><tbody>' +
      S().log.map(function (l) { return '<tr><td class="num" style="white-space:nowrap">' + CN.fmtDate(l.t, true) + "</td><td><b>" + esc(l.who) + "</b></td><td>" + esc(l.what) + "</td></tr>"; }).join("") + "</tbody></table></div>";
  }

  /* ---------- Роутер ---------- */
  function route() {
    var h = (location.hash || "#today").slice(1) || "today", v, nav = h;
    var R = { today: today, bank: bank, mine: mine, opps: opps, community: community, social: social, paper: paper, team: team, analytics: analytics, sources: sources, log: log };
    if (R[h]) v = R[h]();
    else if (h.indexOf("story-") === 0) { v = story(h.slice(6)); nav = "mine"; }
    else v = today();
    shell();
    $("#view").innerHTML = v;
    $("#view").classList.remove("view-enter"); void $("#view").offsetWidth; $("#view").classList.add("view-enter");
    $$("[data-nav]").forEach(function (a) { if (a.dataset.nav === nav) a.setAttribute("aria-current", "page"); else a.removeAttribute("aria-current"); });
    $("#side").classList.remove("open");
    bindDnD();
  }
  function rerender() { var y = window.scrollY; route(); window.scrollTo({ top: y }); }
  window.addEventListener("hashchange", function () { route(); window.scrollTo({ top: 0 }); });

  function bindDnD() {
    $$(".kc").forEach(function (c) {
      c.addEventListener("dragstart", function (e) { c.classList.add("dragging"); e.dataTransfer.setData("text/plain", c.dataset.id); e.dataTransfer.effectAllowed = "move"; });
      c.addEventListener("dragend", function () { c.classList.remove("dragging"); });
    });
    $$(".col").forEach(function (col) {
      col.addEventListener("dragover", function (e) { e.preventDefault(); col.classList.add("over"); });
      col.addEventListener("dragleave", function () { col.classList.remove("over"); });
      col.addEventListener("drop", function (e) {
        e.preventDefault(); col.classList.remove("over");
        var id = e.dataTransfer.getData("text/plain");
        if (col.dataset.col === "rework") { reworkModal(id); return; }
        if (moveTo(id, col.dataset.col)) { toast("Перемещено: " + STATUS[col.dataset.col]); rerender(); }
      });
    });
  }
  function reworkModal(id) {
    var m = document.createElement("div"); m.className = "modal"; m.setAttribute("role", "dialog"); m.setAttribute("aria-modal", "true");
    m.innerHTML = '<form class="box" data-rework-form="' + id + '"><h2 style="font-size:20px">Доработать AI</h2><p class="small muted">Напиши, что исправить. AI перепишет черновик, карточка вернётся в «На проверке».</p><label class="sr" for="rw">Комментарий</label><textarea class="inp" id="rw" required placeholder="Например: найди второй источник для цифры 1,3 млрд"></textarea><div class="row" style="justify-content:flex-end"><button type="button" class="btn" data-close>Отмена</button><button class="btn primary">Отправить AI</button></div></form>';
    document.body.appendChild(m); $("#rw").focus();
  }

  /* ---------- События ---------- */
  var storyId = function () { var h = location.hash.slice(1); return h.indexOf("story-") === 0 ? h.slice(6) : null; };
  document.addEventListener("input", function (e) {
    var el = e.target, a = storyId() && CN.article(storyId());
    if (a && el.dataset.f) { a[el.dataset.f] = el.type === "number" ? +el.value : el.value; CN.save(); var s = socialText(a); var box = $("[data-social]"); if (box) box.textContent = s[ui.tab]; return; }
    if (a && el.dataset.src) { a.source = a.source || { name: "", url: "", date: new Date().toISOString().slice(0, 10), author: "" }; a.source[el.dataset.src] = el.value; CN.save(); return; }
    if (a && el.dataset.photo) { a.photo = a.photo || { url: "", credit: "" }; a.photo[el.dataset.photo] = el.value; CN.save(); return; }
    if (a && el.dataset.blk != null) {
      var b = a.blocks[+el.dataset.blk];
      if (el.dataset.key === "json") { try { a.blocks[+el.dataset.blk] = JSON.parse(el.value); el.style.borderColor = ""; } catch (err) { el.style.borderColor = "var(--bad)"; return; } }
      else b[el.dataset.key] = el.value;
      CN.save(); return;
    }
    if (el.dataset.w) { S().settings.weights[el.dataset.w] = +el.value; CN.save(); var lab = el.previousElementSibling.querySelector(".num"); if (lab) lab.textContent = (+el.value).toFixed(2); return; }
    if ("limit" in el.dataset) { S().settings.aiDailyLimit = +el.value; CN.save(); return; }
  });
  document.addEventListener("change", function (e) {
    var el = e.target, d = el.dataset;
    if (d.filter) { ui[d.filter] = el.value; saveUi(); route(); return; }
    if ("role" in d) { ui.role = el.value; saveUi(); toast("Роль: " + role().label); route(); return; }
    if (d.move) { if (d.move && el.value === "rework") { reworkModal(d.move); return; } if (moveTo(d.move, el.value)) toast("Статус: " + STATUS[el.value]); rerender(); return; }
    if (d.chk) { var o = CN.opp(d.chk); o.checklist = o.checklist || {}; o.checklist[d.k] = el.checked; commit(); rerender(); return; }
    if (d.task) { var t = S().tasks.find(function (x) { return x.id === d.task; }); t.done = el.checked; commit((t.done ? "Выполнено: " : "Вернули в работу: ") + t.text); rerender(); return; }
    if (d.inissue) { var it = S().paper.items, i = it.indexOf(d.inissue); if (el.checked && i < 0) it.push(d.inissue); if (!el.checked && i > -1) it.splice(i, 1); commit(); rerender(); return; }
    if ("upload" in d) {
      var f = el.files[0]; if (!f) return;
      if (f.size > 1.5e6) { toast("Файл больше 1,5 МБ: в демо он не поместится в память браузера"); return; }
      var r = new FileReader(); r.onload = function () { var a = CN.article(storyId()); a.photo = { url: r.result, credit: (a.photo && a.photo.credit) || "Фото: редакция Creata News" }; commit("Загружено фото к «" + a.title + "»"); rerender(); toast("Фото загружено"); };
      r.readAsDataURL(f); return;
    }
  });
  document.addEventListener("click", function (e) {
    var el = e.target.closest("button, a"); if (!el) return;
    var d = el.dataset, a = storyId() && CN.article(storyId());
    if ("burger" in d) { $("#side").classList.toggle("open"); return; }
    if ("new" in d) {
      var id = CN.uid("m");
      S().articles.push({ id: id, slug: id, layer: "world", vertical: ui.vert !== "all" ? ui.vert : "ai", tags: [], label: "Редакция", author: ui.role, status: "new", published: null, title: "", lead: "", tldr: "", live: false, sources: 0, readMin: 3, views: 0, saves: 0, blocks: [{ t: "p", text: "" }], actions: [], why: "", corrections: [] });
      commit("Создан новый материал"); location.hash = "story-" + id; return;
    }
    if (d.take) {
      var b = S().bank.find(function (x) { return x.id === d.take; });
      var nid = CN.uid("m");
      S().articles.push({ id: nid, slug: nid, layer: "world", vertical: b.vertical, tags: [], label: "AI-подборка", author: ui.role, status: "writing", published: null, title: b.title.replace(/^Демо-находка: /, ""), lead: "AI пишет черновик по источнику «" + b.source + "»: заголовок до 8 слов, лид, три уровня подачи и блок «что с этим сделать».", tldr: "", live: false, sources: 1, readMin: 3, views: 0, saves: 0, blocks: [{ t: "p", text: "Черновик появится, когда AI закончит исследование источников." }], actions: [], why: "", corrections: [], demo: b.demo, score: b.score, flags: ["Один источник"] });
      b.taken = nid; commit("Взято в работу из банка: «" + b.title + "»"); toast("Карточка в колонке «AI пишет»"); rerender(); return;
    }
    if (d.act && a) {
      if (moveTo(a.id, d.act)) toast(d.act === "published" ? "Опубликовано на сайте" : "Статус: " + STATUS[d.act]);
      rerender(); return;
    }
    if ("rework" in d && a) { reworkModal(a.id); return; }
    if ("close" in d) { el.closest(".modal").remove(); return; }
    if (d.pick && a) { var lib = CN.photoLib().find(function (p) { return p.src === d.pick; }); var cr = d.pick.indexOf("/c") > 0 ? "Фото: Creata News" : (a.photo && a.photo.credit) || ""; a.photo = { url: d.pick, credit: cr, credit_en: cr.replace("Фото", "Photo") }; commit("Выбрано фото для «" + a.title + "»"); rerender(); return; }
    if (d.sw && a) { a[d.sw] = !a[d.sw]; commit(); rerender(); return; }
    if (d.tab) { ui.tab = d.tab; rerender(); return; }
    if ("copySocial" in d) { var txt = $("[data-social]").textContent; try { navigator.clipboard.writeText(txt).then(function () { toast("Скопировано"); }, function () { toast("Выдели текст и скопируй вручную"); }); } catch (err) { toast("Выдели текст и скопируй вручную"); } return; }
    if ("postiz" in d && a) { S().socialStatus = S().socialStatus || {}; S().socialStatus[a.id + ":" + ui.tab.charAt(0).toUpperCase() + ui.tab.slice(1).replace("edin", "edIn")] = "queued"; commit("Версия «" + a.title + "» отправлена в Postiz"); toast("Поставлено в Postiz на 17:00 (демо)"); return; }
    if (d.badd && a) { var nb = { p: { t: "p", text: "" }, h: { t: "h", text: "" }, quote: { t: "quote", text: "", who: "" }, facts: { t: "facts", title: "Коротко", items: ["Факт 1"] }, timeline: { t: "timeline", title: "Хронология", items: [{ y: "2026", text: "Событие" }] }, quiz: { t: "quiz", q: "Вопрос?", options: ["Ответ 1", "Ответ 2"], answer: 0, explain: "Пояснение" } }[d.badd]; a.blocks.push(nb); commit(); rerender(); return; }
    if (d.bdel != null && a) { a.blocks.splice(+d.bdel, 1); commit(); rerender(); return; }
    if (d.bup != null && a) { var k = +d.bup; if (k > 0) { var tmp = a.blocks[k - 1]; a.blocks[k - 1] = a.blocks[k]; a.blocks[k] = tmp; commit(); rerender(); } return; }
    if (d.opp) { var o = CN.opp(d.opp); o.status = d.st; commit((d.st === "approved" ? "Одобрена возможность «" : "Отклонена возможность «") + o.title + "»"); toast(d.st === "approved" ? "Возможность на сайте" : "Отклонено"); rerender(); return; }
    if (d.rep) {
      var r = S().reports.find(function (x) { return x.id === d.rep; }); r.status = d.st;
      if (d.st === "fixed") { var ar = CN.article(r.article); if (ar) ar.corrections.push({ date: new Date().toISOString().slice(0, 10), text: r.text }); }
      commit("Сообщение об ошибке: " + (d.st === "fixed" ? "исправлено" : "не ошибка")); rerender(); return;
    }
    if (d.cm) { var c = S().comments.find(function (x) { return x.id === d.cm; }); c.status = d.st; commit("Комментарий " + (d.st === "published" ? "опубликован" : "скрыт")); rerender(); return; }
    if (d.soc) { var st = S().socialStatus; st[d.soc] = { draft: "queued", queued: "live", live: "draft" }[st[d.soc] || "draft"]; commit(); rerender(); return; }
    if (d.up) { var it = S().paper.items, j = it.indexOf(d.up); if (j > 0) { it.splice(j, 1); it.splice(j - 1, 0, d.up); commit(); rerender(); } return; }
    if ("pdf" in d) { toast("PDF для печати соберёт сервер после подключения бэкенда. Порядок полос сохранён."); return; }
    if (d.flag) { var fl = S().settings.flags; fl[d.flag] = !fl[d.flag]; commit("Флаг модуля «" + d.flag + "»: " + (fl[d.flag] ? "вкл" : "выкл")); rerender(); return; }
    if ("reset" in d) { el.parentElement.innerHTML = '<span class="small">Точно сбросить?</span><button class="btn sm" data-reset-no>Нет</button><button class="btn sm bad" data-reset-yes>Да, сбросить</button>'; return; }
    if ("resetNo" in d) { rerender(); return; }
    if ("resetYes" in d) { CN.reset(); toast("Демо-данные восстановлены"); rerender(); return; }
  });
  document.addEventListener("submit", function (e) {
    var f = e.target; e.preventDefault();
    if (f.matches("[data-search]")) { ui.q = $("#q").value.trim(); if (["today", "mine", "bank"].indexOf(location.hash.slice(1)) < 0) location.hash = "mine"; else route(); return; }
    if (f.matches("[data-rework-form]")) {
      var id = f.dataset.reworkForm, note = $("#rw").value.trim();
      moveTo(id, "rework", note); f.closest(".modal").remove(); toast("Отправлено AI на доработку"); rerender(); return;
    }
    if (f.matches("[data-addtask]")) { S().tasks.push({ id: CN.uid("t"), who: $("#tw").value, text: $("#tt").value.trim(), vertical: (CN.author($("#tw").value).verticals || ["eco-group"])[0], due: $("#td").value || new Date().toISOString().slice(0, 10), done: false }); commit("Новая задача: " + $("#tt").value.trim()); rerender(); return; }
    if (f.matches("[data-addvert]")) { var name = $("#nv").value.trim(); S().verticals.push({ id: CN.uid("v"), name: name, layer: $("#nvl").value }); commit("Добавлена вертикаль «" + name + "»"); rerender(); return; }
  });
  document.addEventListener("keydown", function (e) { if (e.key === "Escape") { var m = $(".modal"); if (m) m.remove(); } });
  window.addEventListener("storage", function (e) { if (e.key === "creata-news-state" && e.newValue) { CN.state = JSON.parse(e.newValue); rerender(); } });
  route();
})();
