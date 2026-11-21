import React, { useEffect, useMemo, useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { ChevronLeft, ChevronRight, Printer, Shuffle, Check, X } from "lucide-react";
import { CN, useLang, tr, published, vertName, fmtDate, plural } from "./lib.js";
import { Logo, ease } from "./ui.jsx";
import { byCountry, ccName, flag } from "./geo.js";

/* ---------------- Газета, которую сайт собирает сам ----------------
   Номер не берётся готовым: полосы компонуются из опубликованных материалов
   по выбранной формуле. Номер выпуска и дата считаются от текущей даты. */

const INK = "#141414", PAPER = "#F4F1E8";
const paras = (a, lang) => ((lang === "en" && a.blocks_en) || a.blocks || []).filter((b) => b.t === "p").map((b) => b.text);
const sentences = (s) => (s || "").match(/[^.!?…]+[.!?…]+["»”]?/g) || [s || ""];
const first = (a, lang) => sentences(tr(a, "tldr", lang))[0].trim();
/* псевдослучайность от seed, чтобы «Пересобрать» давало другой, но повторяемый номер */
const rnd = (seed) => { let x = seed * 9301 + 49297; return () => ((x = (x * 9301 + 49297) % 233280) / 233280); };

function edition(now) {
  const n = (now.getFullYear() - 2026) * 12 + now.getMonth() - 9; // ноябрь 2026 = №01
  return Math.max(1, n);
}

export function compose(all, mode, seed, lang) {
  const follows = CN.prefs.follows || [];
  const score = (a) => mode === "top" ? a.views + a.saves * 20 : mode === "fresh" ? +new Date(a.published) : (follows.includes(a.vertical) ? 1e7 : 0) + a.views;
  const sorted = [...all].filter((a) => a.photo).sort((a, b) => score(b) - score(a));
  const r = rnd(seed + 1);
  const pool = sorted.slice(0, 4);
  const lead = pool[Math.floor(r() * pool.length)] || sorted[0];
  const rest = sorted.filter((a) => a !== lead);
  const second = rest.slice(0, 3);
  const briefs = rest.slice(3, 9);
  const world = byCountry(all.filter((a) => a.layer === "world"));
  const eco = all.filter((a) => a.layer === "eco").sort((a, b) => score(b) - score(a));
  const opps = (CN.state.opportunities || []).filter((o) => o.status === "approved" && new Date(o.deadline) > new Date()).sort((a, b) => new Date(a.deadline) - new Date(b.deadline)).slice(0, 6);
  /* игра номера: цифры из материалов, угадать, к какой истории относится */
  const statsPool = all.flatMap((a) => ((lang === "en" && a.blocks_en) || a.blocks || []).filter((b) => b.t === "stats").flatMap((b) => b.items.filter((i) => /\d/.test(i.n)).map((i) => ({ a, ...i }))));
  const game = [];
  const used = new Set();
  for (let k = 0; k < statsPool.length && game.length < 4; k++) {
    const it = statsPool[Math.floor(r() * statsPool.length)];
    if (used.has(it.a.id)) continue; used.add(it.a.id);
    const others = all.filter((x) => x !== it.a && x.layer === it.a.layer).sort(() => r() - 0.5).slice(0, 2);
    const opts = [it.a, ...others].sort(() => r() - 0.5);
    game.push({ n: it.n, l: it.l, opts, answer: opts.indexOf(it.a) });
  }
  const quote = all.flatMap((a) => ((lang === "en" && a.blocks_en) || a.blocks || []).filter((b) => b.t === "quote").map((b) => ({ a, ...b })))[Math.floor(r() * 5)];
  return { lead, second, briefs, world, eco, opps, game, quote, count: all.length };
}

/* фото в газетной обработке: ч/б + растр */
function PPhoto({ a, className = "", ratio = "aspect-[3/2]" }) {
  const lang = useLang();
  const map = a.photo?.map;
  return (
    <figure className={className}>
      <div className={"relative overflow-hidden " + ratio}>
        <img src={a.photo.url} alt="" loading="lazy" className={"absolute inset-0 h-full w-full object-cover " + (map ? "grayscale contrast-[1.1]" : "grayscale contrast-[1.15] brightness-[1.04]")} />
        <span aria-hidden="true" className="absolute inset-0 mix-blend-multiply opacity-[.22]" style={{ backgroundImage: "radial-gradient(#000 0.7px, transparent 0.8px)", backgroundSize: "3px 3px" }} />
      </div>
      {a.photo.credit && <figcaption className="mt-1 text-right font-mono text-[9px] uppercase tracking-wider opacity-60">{(lang === "en" && a.photo.credit_en) || a.photo.credit}</figcaption>}
    </figure>
  );
}
const H = ({ children, className = "" }) => <h3 className={"font-head font-bold leading-[1.05] tracking-[-0.01em] " + className}>{children}</h3>;
const Kick = ({ children }) => <p className="font-mono text-[10px] font-semibold uppercase tracking-[0.12em]">{children}</p>;
const Rule = ({ double }) => <div className={double ? "my-3 border-t-[3px] border-double" : "my-3 border-t"} style={{ borderColor: INK }} />;
const Cont = ({ a, lang, page }) => <a href={"#" + a.id} className="font-mono text-[10px] uppercase tracking-wider underline-offset-2 hover:underline">{lang === "en" ? `Continued online →` : `Полностью на сайте →`}{page ? ` · ${lang === "en" ? "p." : "стр."} ${page}` : ""}</a>;

function Masthead({ ed, now, lang, page, total }) {
  return (
    <header>
      <div className="flex items-center justify-between font-mono text-[10px] uppercase tracking-wider">
        <span>{lang === "en" ? "Issue" : "Номер"} {String(ed).padStart(2, "0")}</span>
        <span className="hidden sm:inline">{lang === "en" ? "Innovation news from around the world — and what to do with it" : "Новости об инновациях со всего мира — и что с ними делать"}</span>
        <span>{lang === "en" ? "Page" : "Полоса"} {page}/{total}</span>
      </div>
      <Rule />
      {page === 1 ? (
        <div className="py-2 text-center"><Logo className="text-[32px] sm:text-[80px] lg:text-[112px]" /></div>
      ) : (
        <div className="flex items-baseline justify-between"><Logo className="text-[22px] sm:text-[28px]" /></div>
      )}
      <div className="border-y-[3px] border-double py-1.5 font-mono text-[10px] uppercase tracking-wider sm:flex sm:justify-between" style={{ borderColor: INK }}>
        <span>{fmtDate(now.toISOString(), lang, { weekday: "long", day: "numeric", month: "long", year: "numeric" })}</span>
        <span className="block sm:inline">{lang === "en" ? "Almaty · Assembled automatically" : "Алматы · Собрано автоматически"}</span>
      </div>
    </header>
  );
}

function Front({ c, lang }) {
  const { lead, second, briefs, quote } = c;
  const lp = paras(lead, lang);
  return (
    <>
      <div className="mt-5 grid gap-6 lg:grid-cols-[1fr_260px]">
        <article>
          <Kick>{vertName(lead.vertical, lang)}{lead.country ? " · " + ccName(lead.country, lang) : ""}</Kick>
          <H className="mt-2 text-[34px] sm:text-[52px] lg:text-[62px]"><a href={"#" + lead.id} className="hover:underline">{tr(lead, "title", lang)}</a></H>
          <p className="mt-3 font-serif text-[18px] italic leading-snug sm:text-[20px]">{tr(lead, "lead", lang)}</p>
          <PPhoto a={lead} className="mt-4" ratio="aspect-[16/9]" />
          <div className="mt-4 gap-6 font-serif text-[14.5px] leading-[1.55] sm:columns-2 lg:columns-3 [&>p]:mb-3 [&>p:first-child]:first-letter:float-left [&>p:first-child]:first-letter:mr-1.5 [&>p:first-child]:first-letter:font-head [&>p:first-child]:first-letter:text-[52px] [&>p:first-child]:first-letter:font-bold [&>p:first-child]:first-letter:leading-[0.85]">
            {(lp.length ? lp : [tr(lead, "tldr", lang)]).slice(0, 4).map((p, i) => <p key={i} className="text-justify [hyphens:auto]">{p}</p>)}
            <p><Cont a={lead} lang={lang} /></p>
          </div>
        </article>
        <aside className="border-t pt-3 lg:border-l lg:border-t-0 lg:pl-5 lg:pt-0" style={{ borderColor: INK }}>
          <p className="font-head text-[20px] font-bold uppercase tracking-tight">{lang === "en" ? "In brief" : "Коротко"}</p>
          <Rule />
          <ol>
            {briefs.map((a, i) => (
              <li key={a.id} className="border-b py-2.5 last:border-0" style={{ borderColor: INK + "40" }}>
                <Kick>{vertName(a.vertical, lang)}</Kick>
                <a href={"#" + a.id} className="mt-0.5 block font-head text-[16px] font-bold leading-tight hover:underline">{tr(a, "title", lang)}</a>
                <p className="mt-1 font-serif text-[13px] leading-snug opacity-80">{first(a, lang)}</p>
              </li>
            ))}
          </ol>
          {quote && (
            <blockquote className="mt-4 border-y-[3px] border-double py-4 text-center" style={{ borderColor: INK }}>
              <p className="font-serif text-[18px] italic leading-snug">{lang === "en" ? `“${quote.text}”` : `«${quote.text}»`}</p>
              <footer className="mt-2 font-mono text-[10px] uppercase tracking-wider">— {quote.who}</footer>
            </blockquote>
          )}
        </aside>
      </div>
      <Rule double />
      <div className="grid gap-6 sm:grid-cols-3">
        {second.map((a, i) => (
          <article key={a.id} className={i ? "sm:border-l sm:pl-6" : ""} style={{ borderColor: INK }}>
            <PPhoto a={a} ratio="aspect-[4/3]" />
            <Kick><span className="mt-2 block">{vertName(a.vertical, lang)}</span></Kick>
            <H className="mt-1 text-[21px]"><a href={"#" + a.id} className="hover:underline">{tr(a, "title", lang)}</a></H>
            <p className="mt-2 font-serif text-[14px] leading-snug">{tr(a, "lead", lang)}</p>
            <div className="mt-2"><Cont a={a} lang={lang} /></div>
          </article>
        ))}
      </div>
    </>
  );
}

function WorldPage({ c, lang }) {
  return (
    <>
      <div className="mt-5 flex items-end justify-between gap-4">
        <H className="text-[40px] sm:text-[64px]">{lang === "en" ? "World" : "Мир"}</H>
        <p className="pb-2 text-right font-mono text-[10px] uppercase tracking-wider">{c.world.length} {lang === "en" ? "countries" : plural(c.world.length, "страна", "страны", "стран")}</p>
      </div>
      <Rule double />
      <div className="grid gap-x-6 sm:grid-cols-2 lg:grid-cols-3">
        {c.world.map(([cc, list]) => (
          <section key={cc} className="mb-6 break-inside-avoid border-t-2 pt-2" style={{ borderColor: INK }}>
            <p className="font-head text-[18px] font-bold uppercase tracking-tight">{flag(cc)} {ccName(cc, lang)} <span className="font-mono text-[11px] font-normal">· {list.length}</span></p>
            {list.slice(0, 2).map((a, i) => (
              <article key={a.id} className={"mt-3 " + (i ? "border-t pt-3" : "")} style={{ borderColor: INK + "40" }}>
                {i === 0 && <PPhoto a={a} ratio="aspect-[16/10]" />}
                <H className="mt-2 text-[19px]"><a href={"#" + a.id} className="hover:underline">{tr(a, "title", lang)}</a></H>
                <p className="mt-1.5 font-serif text-[13.5px] leading-snug">{tr(a, "tldr", lang)}</p>
                <p className="mt-1 font-mono text-[9.5px] uppercase tracking-wider opacity-70">{lang === "en" ? "Source" : "Источник"}: {a.source?.name}</p>
              </article>
            ))}
          </section>
        ))}
      </div>
    </>
  );
}

function EcoPage({ c, lang }) {
  const [a, ...rest] = c.eco;
  if (!a) return null;
  return (
    <>
      <div className="mt-5 flex items-end justify-between gap-4">
        <H className="text-[40px] sm:text-[64px]">{lang === "en" ? "Creata Ecosystem" : "Экосистема Creata"}</H>
      </div>
      <Rule double />
      <div className="grid gap-6 lg:grid-cols-[1.3fr_1fr]">
        <article>
          <PPhoto a={a} ratio="aspect-[4/3]" />
          <Kick><span className="mt-3 block">{vertName(a.vertical, lang)}</span></Kick>
          <H className="mt-1 text-[30px] sm:text-[38px]"><a href={"#" + a.id} className="hover:underline">{tr(a, "title", lang)}</a></H>
          <div className="mt-3 gap-5 font-serif text-[14.5px] leading-[1.55] sm:columns-2 [&>p]:mb-3">
            {[tr(a, "lead", lang), ...paras(a, lang)].slice(0, 4).map((p, i) => <p key={i} className="text-justify [hyphens:auto]">{p}</p>)}
            <p><Cont a={a} lang={lang} /></p>
          </div>
        </article>
        <div className="divide-y" style={{ borderColor: INK }}>
          {rest.slice(0, 5).map((x) => (
            <article key={x.id} className="grid grid-cols-[1fr_96px] gap-3 py-3 first:pt-0" style={{ borderColor: INK + "40" }}>
              <div><Kick>{vertName(x.vertical, lang)}</Kick><H className="mt-1 text-[17px]"><a href={"#" + x.id} className="hover:underline">{tr(x, "title", lang)}</a></H><p className="mt-1 font-serif text-[13px] leading-snug">{first(x, lang)}</p></div>
              <PPhoto a={{ ...x, photo: { ...x.photo, credit: "" } }} ratio="aspect-square" />
            </article>
          ))}
        </div>
      </div>
    </>
  );
}

function Game({ c, lang, seed }) {
  const [ans, setAns] = useState({});
  useEffect(() => setAns({}), [seed, lang]);
  const done = Object.keys(ans).length, right = c.game.filter((g, i) => ans[i] === g.answer).length;
  return (
    <section>
      <p className="font-head text-[22px] font-bold uppercase tracking-tight">{lang === "en" ? "Puzzle: numbers of the issue" : "Игра номера: цифры"}</p>
      <p className="font-serif text-[13px] italic">{lang === "en" ? "Which story is each number from?" : "Из какого материала эта цифра?"}</p>
      <Rule />
      <ol className="space-y-4">
        {c.game.map((g, i) => (
          <li key={i}>
            <p><span className="font-head text-[26px] font-bold">{g.n}</span> <span className="font-serif text-[13px]">— {g.l}</span></p>
            <div className="mt-1.5 grid gap-1">
              {g.opts.map((o, k) => {
                const picked = ans[i] === k, show = ans[i] != null, ok = k === g.answer;
                return (
                  <button key={o.id} disabled={show} onClick={() => setAns({ ...ans, [i]: k })}
                    className={"flex items-start gap-2 border px-2 py-1.5 text-left font-serif text-[13px] leading-snug transition-colors " + (show && ok ? "bg-[#141414] text-[#F4F1E8]" : picked ? "line-through opacity-60" : "hover:bg-black/5")} style={{ borderColor: INK }}>
                    <span className="font-mono text-[11px]">{"ABC"[k]}</span><span className="flex-1">{tr(o, "title", lang)}</span>
                    {show && ok && <Check size={14} className="mt-0.5 shrink-0" />}{picked && !ok && <X size={14} className="mt-0.5 shrink-0" />}
                  </button>
                );
              })}
            </div>
          </li>
        ))}
      </ol>
      {done === c.game.length && c.game.length > 0 && <p className="mt-4 border-y-[3px] border-double py-2 text-center font-head text-[18px] font-bold" style={{ borderColor: INK }}>{right}/{c.game.length} · {right === c.game.length ? (lang === "en" ? "Perfect reader!" : "Идеальный читатель!") : lang === "en" ? "Read the issue again 🙂" : "Перечитайте номер 🙂"}</p>}
      <p className="mt-4 rotate-180 text-center font-mono text-[9px] uppercase tracking-wider opacity-60 print:block">{lang === "en" ? "Answers" : "Ответы"}: {c.game.map((g, i) => `${i + 1}${"ABC"[g.answer]}`).join(" · ")}</p>
    </section>
  );
}

function OppsPage({ c, lang, seed }) {
  return (
    <>
      <div className="mt-5"><H className="text-[40px] sm:text-[64px]">{lang === "en" ? "Opportunities" : "Возможности"}</H></div>
      <Rule double />
      <div className="grid gap-8 lg:grid-cols-[1.4fr_1fr]">
        <div>
          <p className="font-serif text-[14px] italic">{lang === "en" ? "Grants, contests and programmes with open deadlines, nearest first." : "Гранты, конкурсы и программы с открытым дедлайном. Сначала ближайшие."}</p>
          <table className="mt-3 w-full border-collapse text-left">
            <thead><tr className="border-b-2 font-mono text-[10px] uppercase tracking-wider" style={{ borderColor: INK }}><th className="py-1.5 pr-3">{lang === "en" ? "Deadline" : "Дедлайн"}</th><th className="py-1.5">{lang === "en" ? "What and for whom" : "Что и для кого"}</th></tr></thead>
            <tbody>
              {c.opps.map((o) => {
                const d = Math.ceil((new Date(o.deadline) - Date.now()) / 864e5);
                return (
                  <tr key={o.id} className="border-b align-top" style={{ borderColor: INK + "40" }}>
                    <td className="whitespace-nowrap py-2.5 pr-3"><p className="font-head text-[22px] font-bold leading-none">{fmtDate(o.deadline, lang, { day: "numeric", month: "short" })}</p><p className="mt-1 font-mono text-[10px] uppercase">{lang === "en" ? `${d} days left` : `осталось ${d} ${plural(d, "день", "дня", "дней")}`}</p></td>
                    <td className="py-2.5"><p className="font-mono text-[10px] uppercase tracking-wider">{tr(o, "type", lang)} · {tr(o, "org", lang)}</p><p className="font-head text-[17px] font-bold leading-tight">{tr(o, "title", lang)}</p><p className="mt-0.5 font-serif text-[13px] leading-snug">{tr(o, "forWhom", lang)}. {tr(o, "conditions", lang)}</p></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <div className="border-t pt-3 lg:border-l lg:border-t-0 lg:pl-6 lg:pt-0" style={{ borderColor: INK }}><Game c={c} lang={lang} seed={seed} /></div>
      </div>
    </>
  );
}

export function Issue() {
  const lang = useLang();
  const now = useMemo(() => new Date(), []);
  const ed = edition(now);
  const all = published(lang);
  const [mode, setMode] = useState("top");
  const [seed, setSeed] = useState(0);
  const [pg, setPg] = useState(0);
  const [dir, setDir] = useState(1);
  const c = useMemo(() => compose(all, mode, seed, lang), [mode, seed, lang, all.length]);
  const pages = [
    { k: "front", label: lang === "en" ? "Front page" : "Главное", el: <Front c={c} lang={lang} /> },
    { k: "world", label: lang === "en" ? "World" : "Мир", el: <WorldPage c={c} lang={lang} /> },
    { k: "eco", label: lang === "en" ? "Ecosystem" : "Экосистема", el: <EcoPage c={c} lang={lang} /> },
    { k: "opps", label: lang === "en" ? "Opportunities & puzzle" : "Возможности и игра", el: <OppsPage c={c} lang={lang} seed={seed} /> },
  ];
  const to = (n) => { if (n < 0 || n >= pages.length) return; setDir(n > pg ? 1 : -1); setPg(n); };
  useEffect(() => {
    const f = (e) => { if (e.target.closest("input,textarea,select")) return; if (e.key === "ArrowRight") to(pg + 1); if (e.key === "ArrowLeft") to(pg - 1); };
    window.addEventListener("keydown", f); return () => window.removeEventListener("keydown", f);
  });
  const modes = [["top", lang === "en" ? "Most read" : "Самое читаемое"], ["fresh", lang === "en" ? "Latest" : "Самое свежее"], ["you", lang === "en" ? "For you" : "Для вас"]];
  const sheet = (p, i) => (
    <div className="paper-sheet mx-auto max-w-[1180px] px-4 py-5 shadow-[0_30px_80px_-30px_rgba(0,0,0,.45)] sm:px-8 sm:py-7" style={{ background: PAPER, color: INK }}>
      <Masthead ed={ed} now={now} lang={lang} page={i + 1} total={pages.length} />
      {p.el}
      <div className="mt-6 flex justify-between border-t pt-2 font-mono text-[9.5px] uppercase tracking-wider" style={{ borderColor: INK }}>
        <span>Creata News · {lang === "en" ? "Issue" : "№"} {String(ed).padStart(2, "0")}</span>
        <span>{lang === "en" ? `Built from ${c.count} published stories` : `Собрано из ${c.count} опубликованных материалов`}</span>
      </div>
    </div>
  );
  return (
    <div className="pb-16">
      <div className="wrap pt-8 no-print">
        <p className="kicker">{lang === "en" ? "Print edition" : "Газета"} · №{String(ed).padStart(2, "0")}</p>
        <h1 className="h-title mt-2 text-[44px] sm:text-[72px]">{lang === "en" ? "The paper builds itself" : "Газета собирается сама"}</h1>
        <p className="mt-3 max-w-2xl font-serif text-[18px] text-muted">
          {lang === "en" ? "Nothing here is a ready-made file. The site lays out the issue from published stories: the lead, briefs, a World page by country, the ecosystem, open deadlines and a puzzle built from the numbers in the stories. Pick a formula or rebuild it." : "Здесь нет готового файла. Сайт сам верстает номер из опубликованных материалов: главная тема, «Коротко», полоса «Мир» по странам, экосистема, открытые дедлайны и игра из цифр в материалах. Выберите формулу или пересоберите номер."}
        </p>
        <div className="mt-6 flex flex-wrap items-center gap-2">
          {modes.map(([k, l]) => (
            <button key={k} onClick={() => { setMode(k); setPg(0); }} className={"relative h-10 rounded-full px-4 text-sm font-semibold " + (mode === k ? "text-paper" : "border border-rule bg-card")}>
              {mode === k && <motion.span layoutId="pmode" className="absolute inset-0 rounded-full bg-ink" />}<span className="relative">{l}</span>
            </button>
          ))}
          <button onClick={() => { setSeed(seed + 1); setPg(0); }} className="btn-line h-10"><Shuffle size={15} /> {lang === "en" ? "Rebuild" : "Пересобрать"}</button>
          <button onClick={() => window.print()} className="btn-accent h-10"><Printer size={15} /> {lang === "en" ? "Print / PDF" : "Печать / PDF"}</button>
        </div>
        <nav aria-label={lang === "en" ? "Pages" : "Полосы"} className="mt-8 flex items-center gap-2 overflow-x-auto pb-1">
          <button onClick={() => to(pg - 1)} disabled={pg === 0} className="btn-line h-10 w-10 shrink-0 !px-0 disabled:opacity-30" aria-label={lang === "en" ? "Previous page" : "Предыдущая полоса"}><ChevronLeft size={18} /></button>
          {pages.map((p, i) => (
            <button key={p.k} onClick={() => to(i)} aria-current={i === pg ? "page" : undefined} className={"h-10 shrink-0 rounded-full px-3.5 text-sm font-semibold " + (i === pg ? "bg-accent text-white" : "hover:text-accent")}>
              <span className="font-mono text-xs opacity-70">{i + 1}</span> {p.label}
            </button>
          ))}
          <button onClick={() => to(pg + 1)} disabled={pg === pages.length - 1} className="btn-line h-10 w-10 shrink-0 !px-0 disabled:opacity-30" aria-label={lang === "en" ? "Next page" : "Следующая полоса"}><ChevronRight size={18} /></button>
        </nav>
      </div>
      <div className="no-print mt-6 px-2 sm:px-4 [perspective:2200px]">
        <AnimatePresence mode="wait" custom={dir}>
          <motion.div key={pg + mode + seed + lang} custom={dir}
            initial={{ opacity: 0, rotateY: dir * 18, x: dir * 60, transformOrigin: dir > 0 ? "left center" : "right center" }}
            animate={{ opacity: 1, rotateY: 0, x: 0 }} exit={{ opacity: 0, rotateY: -dir * 14, x: -dir * 40 }} transition={{ duration: 0.45, ease }}>
            {sheet(pages[pg], pg)}
          </motion.div>
        </AnimatePresence>
      </div>
      {/* для печати: все полосы подряд */}
      <div className="paper-print hidden">{pages.map((p, i) => <div key={p.k} className="paper-page">{sheet(p, i)}</div>)}</div>
    </div>
  );
}
