import React, { useEffect, useMemo, useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { X, ChevronLeft, ChevronRight, Trophy, Award, Eye, Bookmark, FileText, Users, Download, Newspaper } from "lucide-react";
import { CN, useLang, tr, published, vertName, fmtDate, togglePref, plural, useMedia } from "./lib.js";
import { byCountry, ccOf, ccName, flag } from "./geo.js";
import { StoryGlobe } from "./globe.jsx";
import { useT } from "./i18n.js";
import { StoryCard, SectionHead, Chips, Photo, Avatar, Kicker, Byline, Rise, Logo, HoverMembers, DragStrip, ease } from "./ui.jsx";
import { OppCard, Newsletter } from "./home.jsx";
import { AnimatedNumber_004 } from "./skiper/skiper37.jsx";
import { HoverExpand_001 } from "./skiper/skiper52.jsx";
import { Carousel_006 } from "./skiper/skiper54.jsx";

/* ---------- Сетка материалов с фильтрами и пагинацией (Figma) ---------- */
export function Grid({ layer, vertical, country }) {
  const lang = useLang(), t = useT();
  const v = vertical ? CN.vert(vertical) : null;
  const L = layer || v?.layer || "world";
  const W = L === "world" && !vertical;
  const verts = CN.state.verticals.filter((x) => x.layer === L);
  const base = published(lang).filter((a) => a.layer === L);
  const [f, setF] = useState(vertical || country || "all");
  const [sort, setSort] = useState("new");
  const [page, setPage] = useState(1);
  useEffect(() => { setF(vertical || country || "all"); setPage(1); }, [vertical, layer, country]);
  const groups = useMemo(() => byCountry(base), [lang, base.length]);
  const counts = useMemo(() => Object.fromEntries(groups.map(([c, l]) => [c, l.length])), [groups]);
  const match = (a) => f === "all" || (W ? ccOf(a).includes(f) : a.vertical === f);
  const list = useMemo(() => {
    let l = base.filter(match);
    if (sort === "pop") l = [...l].sort((a, b) => b.views - a.views);
    return l;
  }, [f, sort, lang, base.length]);
  const per = 9, pages = Math.max(1, Math.ceil(list.length / per));
  const shown = list.slice((page - 1) * per, page * per);
  const chips = W
    ? [{ id: "all", label: t.allCountries, count: base.length }, ...groups.map(([c, l]) => ({ id: c, label: flag(c) + " " + ccName(c, lang), count: l.length }))]
    : [{ id: "all", label: t.filterAll, count: base.length }, ...verts.map((x) => ({ id: x.id, label: tr(x, "name", lang), count: base.filter((a) => a.vertical === x.id).length })).filter((c) => c.count > 0 || c.id === vertical)];
  const title = f !== "all" ? (W ? ccName(f, lang) : vertName(f, lang)) : L === "eco" ? t.eco : t.world;
  const lead = L === "eco" ? t.ecoLead : t.worldLead;
  const top = shown[0];
  const mob = useMedia("(max-width: 1023px)"); const globeH = mob ? 460 : 640;
  const pick = (x) => { setF(x || "all"); setPage(1); };
  return (
    <div className={W ? "" : "wrap pt-8"}>
      {W ? (
        <header className="relative overflow-hidden border-b border-rule bg-band text-ink">
          <div className="wrap grid grid-cols-1 items-center gap-6 pt-10 lg:grid-cols-12 lg:pt-6">
            <div className="min-w-0 lg:col-span-4">
              <p className="kicker">{t.world} · {groups.length} {lang === "en" ? "countries" : plural(groups.length, "страна", "страны", "стран")}</p>
              <AnimatePresence mode="wait"><motion.h1 key={title} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -12 }} className="h-title mt-2 text-[44px] sm:text-[72px]">{f !== "all" && <span className="mr-3">{flag(f)}</span>}{title}</motion.h1></AnimatePresence>
              <p className="mt-3 max-w-xl font-serif text-[18px] text-muted">{f !== "all" ? (lang === "en" ? `${counts[f] || 0} stories from ${ccName(f, lang)}. Click another flag on the globe to switch.` : `${counts[f] || 0} ${plural(counts[f] || 0, "материал", "материала", "материалов")} о стране. Нажмите на другой флажок на глобусе, чтобы переключиться.`) : lead}</p>
              {f !== "all" ? (
                <button onClick={() => pick(null)} className="mt-6 inline-flex h-10 items-center gap-2 rounded-full border border-rule bg-card px-4 text-sm font-semibold hover:border-accent hover:text-accent"><X size={15} /> {t.allCountries}</button>
              ) : (
                <p className="mt-6 flex items-center gap-2 font-mono text-[11px] uppercase tracking-wider text-muted"><span className="relative flex h-2 w-2"><span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent opacity-75" /><span className="relative inline-flex h-2 w-2 rounded-full bg-accent" /></span>{lang === "en" ? "Hover a flag and the country's news float around the globe" : "Наведите на флажок: новости страны всплывут вокруг глобуса"}</p>
              )}
            </div>
            <div className="-mx-4 min-w-0 overflow-hidden lg:col-span-8 lg:mx-0"><StoryGlobe groups={groups} selected={f === "all" ? null : f} onSelect={pick} lang={lang} height={globeH} /></div>
          </div>
        </header>
      ) : (
      <header className="border-b border-ink pb-6">
        <p className="kicker">{L === "eco" ? t.eco : t.world}</p>
        <AnimatePresence mode="wait"><motion.h1 key={title} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -12 }} className="h-title mt-2 text-[44px] sm:text-[72px]">{title}</motion.h1></AnimatePresence>
        <p className="mt-2 max-w-2xl font-serif text-[18px] text-muted">{lead}</p>
        {f === "all" && base.length > 3 && (
          <div className="mt-8 hidden md:block">
            <HoverExpand_001 className="mx-auto" active="28rem" idle="4.5rem" height="20rem"
              images={base.slice(0, 8).map((a) => ({ src: a.photo.url, alt: "", code: tr(a, "title", lang), href: "#" + a.id }))} />
          </div>
        )}
      </header>
      )}
      <div className={W ? "wrap" : ""}>
      <div className="sticky top-14 z-30 -mx-4 flex flex-col gap-3 bg-paper/95 px-4 py-4 backdrop-blur sm:mx-0 sm:px-0 lg:flex-row lg:items-center lg:justify-between">
        {W ? <p className="meta">{f === "all" ? (lang === "en" ? `All stories · ${base.length}` : `Все материалы · ${base.length}`) : `${flag(f)} ${ccName(f, lang)} · ${list.length}`}</p> : <Chips id={"g" + L} items={chips} value={f} onChange={pick} />}
        <select value={sort} onChange={(e) => setSort(e.target.value)} aria-label={t.sort} className="h-9 shrink-0 self-start rounded-full border border-rule bg-card px-3 text-sm font-semibold">
          <option value="new">{t.sortNew}</option><option value="pop">{t.sortPopular}</option>
        </select>
      </div>
      <AnimatePresence mode="wait">
        <motion.div key={f + sort + page + lang} initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} transition={{ duration: 0.25 }}>
          {!shown.length && <p className="py-20 text-center font-serif text-lg text-muted">{t.nothing}</p>}
          {top && page === 1 && (
            <article className="group mt-4 grid items-center gap-6 border-b border-rule pb-10 lg:grid-cols-12">
              <a href={"#" + top.id} tabIndex={-1} aria-hidden="true" className="lg:col-span-7"><Photo src={top.photo.url} ratio="aspect-[16/10]" live={top.live} /></a>
              <div className="lg:col-span-5"><Kicker a={top} /><h2 className="h-title mt-2 text-[30px] sm:text-[38px]"><a href={"#" + top.id} className="hover-u">{tr(top, "title", lang)}</a></h2>
                <p className="mt-3 font-serif text-[18px] text-muted">{tr(top, "lead", lang)}</p><Byline a={top} date className="mt-4" /></div>
            </article>
          )}
          <div className="mt-10 grid gap-x-8 gap-y-12 sm:grid-cols-2 lg:grid-cols-3">
            {(page === 1 ? shown.slice(1) : shown).map((a, i) => <motion.div key={a.id} initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.04 }}><StoryCard a={a} size="md" /></motion.div>)}
          </div>
        </motion.div>
      </AnimatePresence>
      {pages > 1 && (
        <nav aria-label={t.page} className="mt-14 flex items-center justify-center gap-2">
          <button disabled={page === 1} onClick={() => setPage(page - 1)} className="btn-line h-10 w-10 !px-0 disabled:opacity-30" aria-label={t.prev}><ChevronLeft size={18} /></button>
          {Array.from({ length: pages }, (_, i) => i + 1).map((n) => (
            <button key={n} onClick={() => setPage(n)} aria-current={n === page ? "page" : undefined} className="relative h-10 w-10 rounded-full font-mono text-sm font-semibold">
              {n === page && <motion.span layoutId="pg" className="absolute inset-0 rounded-full bg-ink" />}<span className={"relative " + (n === page ? "text-paper" : "")}>{n}</span>
            </button>
          ))}
          <button disabled={page === pages} onClick={() => setPage(page + 1)} className="btn-line h-10 w-10 !px-0 disabled:opacity-30" aria-label={t.next}><ChevronRight size={18} /></button>
        </nav>
      )}
      </div>
    </div>
  );
}

/* ---------- Профиль автора (Figma: обложка, аватар, рейтинг, достижения) ---------- */
export function score(au) {
  const mine = CN.state.articles.filter((a) => a.author === au.id && a.status === "published");
  const views = mine.reduce((s, a) => s + a.views, 0), saves = mine.reduce((s, a) => s + a.saves, 0);
  return { mine, views, saves, pts: Math.round(views / 10 + saves * 2 + mine.length * 50) };
}
export function Author({ id }) {
  const lang = useLang(), t = useT();
  const au = CN.author(id);
  const [tab, setTab] = useState("stories");
  const s = score(au);
  const mine = s.mine.filter((a) => lang !== "en" || a.title_en).sort((a, b) => new Date(b.published) - new Date(a.published));
  const ranking = CN.state.authors.map((x) => ({ a: x, ...score(x) })).sort((x, y) => y.pts - x.pts);
  const max = ranking[0].pts || 1;
  const following = (CN.prefs.follows || []).includes("author:" + id);
  return (
    <div>
      <div className="relative h-[220px] overflow-hidden bg-band sm:h-[340px]">
        {au.cover && <motion.img initial={{ scale: 1.15 }} animate={{ scale: 1 }} transition={{ duration: 1.4, ease }} src={au.cover} alt="" className="h-full w-full object-cover" />}
        <div className="absolute inset-0 bg-gradient-to-t from-black/40 to-transparent" />
      </div>
      <div className="wrap">
        <div className="relative z-10 -mt-14 flex flex-col gap-5 border-b border-rule pb-6 sm:-mt-16 sm:flex-row sm:items-end sm:justify-between">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <motion.div initial={{ scale: 0.6, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} transition={{ type: "spring", delay: 0.2 }} className="w-fit rounded-full bg-paper p-1.5"><Avatar a={au} size={120} /></motion.div>
            <div className="pb-1">
              <h1 className="h-title text-[34px] sm:text-[44px]">{tr(au, "name", lang)}</h1>
              <p className="meta mt-1">{tr(au, "role", lang)} · Creata News</p>
            </div>
          </div>
          <button onClick={() => togglePref("follows", "author:" + id)} aria-pressed={following} className={following ? "btn-line" : "btn-accent"}>{following ? "✓ " + t.following : t.follow}</button>
        </div>
        <p className="mt-5 max-w-2xl font-serif text-[18px]">{tr(au, "bio", lang)}</p>
        <div className="mt-6 grid max-w-2xl grid-cols-3 gap-px overflow-hidden rounded-xl bg-rule">
          {[[FileText, mine.length, t.stories], [Users, au.followers || 0, t.followers], [Eye, s.views, t.views]].map(([I, n, l], i) => (
            <div key={i} className="bg-card p-4"><I size={16} className="text-accent" /><p className="mt-2 font-logo text-[24px] font-bold"><AnimatedNumber_004 to={n} locales={lang === "en" ? "en-US" : "ru-RU"} /></p><p className="meta">{l}</p></div>
          ))}
        </div>
        <div className="mt-10"><Chips id="atab" items={[{ id: "stories", label: t.stories }, { id: "ranking", label: t.ranking }, { id: "ach", label: t.achievements }]} value={tab} onChange={setTab} /></div>
        <AnimatePresence mode="wait">
          <motion.div key={tab} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }} transition={{ duration: 0.25 }} className="mt-8">
            {tab === "stories" && (mine.length ? <div className="grid gap-x-8 gap-y-10 sm:grid-cols-2 lg:grid-cols-3">{mine.map((a) => <StoryCard key={a.id} a={a} size="md" />)}</div> : <p className="text-muted">{t.nothing}</p>)}
            {tab === "ranking" && (
              <div className="max-w-3xl">
                <p className="mb-4 font-serif text-muted">{t.rankingLead}</p>
                <ol className="space-y-2">
                  {ranking.map((r, i) => (
                    <li key={r.a.id}>
                      <a href={"#author-" + r.a.id} className={"grid grid-cols-[32px_40px_1fr_auto] items-center gap-3 rounded-lg p-2.5 " + (r.a.id === id ? "bg-soft ring-1 ring-accent" : "hover:bg-band")}>
                        <span className="font-mono text-sm font-semibold">{i < 3 ? <Trophy size={18} className={["text-[#C9A227]", "text-[#9AA0A6]", "text-[#B87333]"][i]} /> : i + 1}</span>
                        <Avatar a={r.a} size={40} />
                        <span className="min-w-0"><span className="block truncate font-semibold">{tr(r.a, "name", lang)}</span>
                          <span className="mt-1 block h-1.5 overflow-hidden rounded bg-band"><motion.span initial={{ width: 0 }} animate={{ width: `${(r.pts / max) * 100}%` }} transition={{ duration: 0.9, delay: i * 0.05, ease }} className="block h-full bg-accent" /></span></span>
                        <span className="font-mono text-sm tabular-nums">{r.pts}</span>
                      </a>
                    </li>
                  ))}
                </ol>
              </div>
            )}
            {tab === "ach" && (
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                {Object.entries(t.ach).map(([k, [name, desc]], i) => {
                  const got = (au.achievements || []).includes(k);
                  return (
                    <motion.div key={k} initial={{ opacity: 0, scale: 0.9 }} animate={{ opacity: 1, scale: 1 }} transition={{ delay: i * 0.04 }} whileHover={got ? { y: -4, rotate: -1 } : {}}
                      className={"rounded-xl p-5 ring-1 " + (got ? "bg-card ring-accent" : "bg-band opacity-50 ring-rule grayscale")}>
                      <span className={"grid h-12 w-12 place-items-center rounded-full " + (got ? "bg-accent text-onaccent" : "bg-rule")}><Award size={22} /></span>
                      <p className="h-title mt-3 text-[18px]">{name}</p><p className="mt-1 text-[14px] text-muted">{desc}</p>
                    </motion.div>
                  );
                })}
              </div>
            )}
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  );
}

export function Authors() {
  const lang = useLang(), t = useT();
  const ranking = CN.state.authors.map((x) => ({ a: x, ...score(x) })).sort((x, y) => y.pts - x.pts);
  return (
    <div className="wrap pt-8">
      <header className="border-b border-ink pb-6"><p className="kicker">Creata News</p><h1 className="h-title mt-2 text-[44px] sm:text-[72px]">{t.team}</h1><p className="mt-2 max-w-2xl font-serif text-[18px] text-muted">{t.teamLead}</p></header>
      <div className="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {ranking.map(({ a, pts, mine }, i) => (
          <Rise key={a.id} delay={i * 0.04}>
            <a href={"#author-" + a.id} className="group block overflow-hidden rounded-xl bg-card ring-1 ring-rule transition hover:ring-accent">
              <div className="relative h-28 overflow-hidden bg-band">{a.cover && <img src={a.cover} alt="" loading="lazy" className="h-full w-full object-cover transition duration-700 group-hover:scale-105" />}</div>
              <div className="-mt-8 px-5 pb-5"><span className="inline-block rounded-full bg-card p-1"><Avatar a={a} size={56} /></span>
                <p className="h-title mt-2 text-[20px] group-hover:text-accent">{tr(a, "name", lang)}</p><p className="meta">{tr(a, "role", lang)}</p>
                <p className="mt-3 line-clamp-2 text-[14px] text-muted">{tr(a, "bio", lang)}</p>
                <p className="meta mt-3">{mine.length} · {t.stories} · {pts} {t.pts}</p></div>
            </a>
          </Rise>
        ))}
      </div>
    </div>
  );
}

/* ---------- Возможности ---------- */
export function Opps() {
  const lang = useLang(), t = useT();
  const all = CN.liveOpps();
  const types = [...new Set(all.map((o) => o.type))];
  const [f, setF] = useState("all");
  const list = all.filter((o) => f === "all" || o.type === f);
  return (
    <div className="wrap pt-8">
      <header className="border-b border-ink pb-6"><p className="kicker">{t.opps}</p><h1 className="h-title mt-2 text-[44px] sm:text-[72px]">{t.oppsTitle}</h1><p className="mt-2 max-w-2xl font-serif text-[18px] text-muted">{t.oppsLead}</p></header>
      <div className="py-5"><Chips id="opp" items={[{ id: "all", label: t.filterAll, count: all.length }, ...types.map((ty) => ({ id: ty, label: tr(all.find((o) => o.type === ty), "type", lang), count: all.filter((o) => o.type === ty).length }))]} value={f} onChange={setF} /></div>
      <motion.div layout className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <AnimatePresence>{list.map((o, i) => <motion.div key={o.id} layout initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0, scale: 0.95 }}><OppCard o={o} i={i} /></motion.div>)}</AnimatePresence>
      </motion.div>
      <p className="meta mt-8">{lang === "en" ? "Demo opportunities: links lead to the organisers' sites, deadlines are illustrative." : "Демо-возможности: ссылки ведут на сайты организаторов, дедлайны условные."}</p>
    </div>
  );
}

/* ---------- Газета №1 ---------- */
export function Issue() {
  const lang = useLang(), t = useT();
  const all = published(lang);
  const iss = CN.state.issue;
  const pics = all.filter((a) => a.layer === "eco").flatMap((a) => (a.photos || []).slice(0, 2).map((p) => ({ src: p.src, alt: lang === "en" ? p.cap_en : p.cap, title: lang === "en" ? p.cap_en : p.cap })));
  return (
    <div className="wrap pt-8">
      <div className="grid gap-10 lg:grid-cols-[minmax(0,460px)_1fr]">
        <motion.div initial={{ rotate: -4, y: 30, opacity: 0 }} animate={{ rotate: -2, y: 0, opacity: 1 }} transition={{ duration: 0.9, ease }} className="relative mx-auto w-full max-w-[460px]">
          <div className="bg-[#F3F1EA] p-4 text-[#0E0E10] shadow-[0_30px_60px_-20px_rgba(0,0,0,.45)]">
            <div className="flex items-center justify-between border-b-2 border-[#0E0E10] pb-2"><Logo className="text-[22px]" /><span className="font-mono text-[10px] uppercase">№01 · {tr(iss, "month", lang)}</span></div>
            <img src={iss.cover} alt="" className="mt-3 aspect-[4/5] w-full object-cover grayscale-[.15]" />
            <p className="mt-3 font-head text-[24px] font-bold leading-tight">{tr(all[0], "title", lang)}</p>
            <div className="mt-3 grid grid-cols-3 gap-2 border-t border-[#0E0E10] pt-2">{all.slice(1, 4).map((a) => <p key={a.id} className="font-serif text-[11px] leading-tight">{tr(a, "title", lang)}</p>)}</div>
          </div>
        </motion.div>
        <div>
          <p className="kicker">{tr(iss, "month", lang)} · {tr(iss, "city", lang)}</p>
          <h1 className="h-title mt-2 text-[48px] sm:text-[80px]">{t.issueTitle}</h1>
          <p className="mt-3 max-w-xl font-serif text-[19px] text-muted">{t.issueLead}</p>
          <div className="mt-6 flex flex-wrap gap-3"><a href="https://creata.news/issues/01" target="_blank" rel="noopener" className="btn-accent"><Newspaper size={16} /> creata.news/issues/01</a><span className="btn-line pointer-events-none opacity-60"><Download size={16} /> {t.pdfSoon}</span></div>
          <h2 className="h-title mt-10 border-t-2 border-ink pt-3 text-[24px]">{t.inIssue}</h2>
          <ol className="mt-2">
            {all.map((a, i) => (
              <li key={a.id}><a href={"#" + a.id} className="group grid grid-cols-[40px_1fr_auto] items-baseline gap-3 border-b border-rule py-3">
                <span className="font-mono text-sm text-muted">{String(i + 2).padStart(2, "0")}</span>
                <span><span className="kicker">{vertName(a.vertical, lang)}</span><span className="block text-[17px] font-semibold leading-snug group-hover:text-accent">{tr(a, "title", lang)}</span></span>
                <span className="meta hidden sm:block">{tr(CN.author(a.author), "name", lang)}</span>
              </a></li>
            ))}
          </ol>
        </div>
      </div>
      <h2 className="h-title mt-16 border-t-2 border-ink pt-3 text-[24px]">{lang === "en" ? "Photos from the issue" : "Фото номера"}</h2>
      <div className="mt-6 pb-10"><Carousel_006 images={pics} /></div>
    </div>
  );
}

/* ---------- Утро Creata (страница) ---------- */
export function Morning({ onSubscribe }) {
  const t = useT(), lang = useLang();
  const items = lang === "en"
    ? [["08:00", "Creator's Forecast", "A tip for the day by your sign, your profession or interest, based on a real trend or deadline."], ["5", "News items", "What changed in the world while you slept, each with a link to the original."], ["3", "Opportunities", "Grants, contests and internships for your stage."], ["1", "Outside your bubble", "One story from a topic you don't follow yet."], ["5×5", "Game of the Day", "A five-letter word puzzle you can share."]]
    : [["08:00", "Прогноз креатора", "Совет на день по твоему знаку — профессии или интересу, на основе реального тренда или дедлайна."], ["5", "Новостей", "Что изменилось в мире, пока ты спал, — каждая со ссылкой на оригинал."], ["3", "Возможности", "Гранты, конкурсы и стажировки для твоей стадии."], ["1", "Вне пузыря", "Одна история из темы, на которую ты ещё не подписан."], ["5×5", "Игра дня", "Слово из пяти букв, результатом можно поделиться."]];
  return (
    <div>
      <Newsletter onSubscribe={onSubscribe} />
      <div className="wrap mt-14 grid gap-px overflow-hidden rounded-xl bg-rule sm:grid-cols-2 lg:grid-cols-5">
        {items.map(([n, h, p], i) => <Rise key={i} delay={i * 0.05} className="bg-card p-6"><p className="font-logo text-[28px] font-bold text-accent">{n}</p><p className="h-title mt-3 text-[19px]">{h}</p><p className="mt-2 text-[14px] text-muted">{p}</p></Rise>)}
      </div>
    </div>
  );
}

/* ---------- О нас ---------- */
export function About() {
  const t = useT(), lang = useLang();
  const labels = [[t.labelEditorial, lang === "en" ? "Our own reporting or a retelling with a link to the original." : "Наш материал или пересказ со ссылкой на оригинал."], [t.labelEco, lang === "en" ? "Stories about Creata Group. Final approval by the editor-in-chief." : "Материалы о Creata Group. Финальное одобрение — главный редактор."], [t.labelPartner, lang === "en" ? "Paid or partner content, always labelled." : "Платный или партнёрский материал, всегда с меткой."], [t.labelAi, lang === "en" ? "Found and drafted by AI, checked by an editor." : "Найдено и написано AI, проверено редактором."]];
  return (
    <div className="wrap pt-8">
      <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1.1fr_1fr] [&>*]:min-w-0">
        <div>
          <p className="kicker">Creata News</p>
          <h1 className="h-title mt-2 text-[44px] sm:text-[72px]">{t.aboutTitle}</h1>
          {t.aboutText.map((p, i) => <p key={i} className="mt-5 font-serif text-[20px] leading-relaxed">{p}</p>)}
          <div className="mt-8 flex flex-wrap items-center gap-5"><HoverMembers people={CN.state.authors} max={6} /><a href="#authors" className="btn-line">{t.authors}</a></div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          {["c03", "c39", "c112", "c26"].map((n, i) => <Photo key={n} src={"assets/photos/" + n + ".jpg"} ratio={i % 3 === 0 ? "aspect-[3/4]" : "aspect-[4/3]"} className={i === 1 ? "mt-10" : ""} />)}
        </div>
      </div>
      <h2 className="h-title mt-16 border-t-2 border-ink pt-3 text-[26px]">{t.labels}</h2>
      <div className="mt-6 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">{labels.map(([h, p]) => <div key={h}><span className="inline-block rounded-sm bg-soft px-2 py-1 font-mono text-[11px] uppercase tracking-wider text-accent-ink">{h}</span><p className="mt-3 text-[15px] text-muted">{p}</p></div>)}</div>
    </div>
  );
}

export function Saved() {
  const t = useT(), lang = useLang();
  const list = published(lang).filter((a) => (CN.prefs.saved || []).includes(a.id));
  return (
    <div className="wrap pt-8">
      <header className="border-b border-ink pb-6"><h1 className="h-title text-[44px] sm:text-[64px]">{t.saved}</h1></header>
      {list.length ? <div className="mt-10 grid gap-x-8 gap-y-10 sm:grid-cols-2 lg:grid-cols-3">{list.map((a) => <StoryCard key={a.id} a={a} size="md" />)}</div> : <p className="mt-10 font-serif text-lg text-muted">{t.nothing}</p>}
    </div>
  );
}

export function NotFound() {
  const t = useT();
  return <div className="wrap py-24 text-center"><p className="font-logo text-[80px] font-bold text-accent">404</p><h1 className="h-title text-[28px]">{t.notFound}</h1><a href="#" className="btn-accent mt-6">{t.toHome}</a></div>;
}
