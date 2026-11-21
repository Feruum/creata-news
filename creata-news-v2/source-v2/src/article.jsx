import React, { useState } from "react";
import { motion, useScroll, useSpring } from "motion/react";
import { ExternalLink, Link2, Share2, Clock, MapPin, Check, X as XIcon } from "lucide-react";
import { CN, useLang, tr, published, vertName, fmtDate, setPref, bump } from "./lib.js";
import { useT } from "./i18n.js";
import { Kicker, Photo, SaveBtn, StoryCard, SectionHead, Avatar, LabelTag, Rise, ease } from "./ui.jsx";
import { OppCard } from "./home.jsx";
import { Skiper89 } from "./skiper/skiper89.jsx";
import { AnimatedNumber_004 } from "./skiper/skiper37.jsx";

export function Article({ a }) {
  const lang = useLang(), t = useT();
  const { scrollYProgress } = useScroll();
  const prog = useSpring(scrollYProgress, { stiffness: 200, damping: 30 });
  const deep = CN.prefs.level !== "short";
  const blocks = (lang === "en" ? a.blocks_en : a.blocks) || [];
  const photos = a.photos || [];
  const au = CN.author(a.author);
  const related = published(lang).filter((x) => x.id !== a.id && (x.vertical === a.vertical || x.layer === a.layer)).slice(0, 4);
  const credit = lang === "en" ? a.photo?.credit_en || a.photo?.credit : a.photo?.credit;
  const src = a.source;
  const corrections = (lang === "en" ? a.corrections_en : a.corrections) || [];

  /* фото вставляем между абзацами: после 2-го, 4-го блока и т.д. */
  const items = [];
  let pi = 0;
  blocks.filter((b) => deep || b.lvl !== "deep").forEach((b, i) => {
    items.push({ b, i });
    if ((i === 1 || i === 3 || i === 5) && pi < photos.length) {
      const n = i === 1 && photos.length - pi >= 2 ? 2 : 1;
      items.push({ photos: photos.slice(pi, pi + n) });
      pi += n;
    }
  });
  const rest = photos.slice(pi);

  return (
    <article>
      <motion.div className="fixed inset-x-0 top-0 z-[60] h-[3px] origin-left bg-accent" style={{ scaleX: prog }} />
      <Skiper89 className="bottom-20 sm:bottom-4" />
      <header className="wrap pt-8 sm:pt-12">
        <div className="mx-auto max-w-4xl">
          <div className="flex flex-wrap items-center gap-2"><Kicker a={a} className="text-[12px]" /><LabelTag a={a} /></div>
          <motion.h1 initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.6, ease }} className="h-title mt-3 text-[34px] sm:text-[52px] lg:text-[60px] leading-[1.04]">{tr(a, "title", lang)}</motion.h1>
          <motion.p initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.6, delay: 0.1, ease }} className="mt-4 font-serif text-[20px] leading-snug text-muted sm:text-[23px]">{tr(a, "lead", lang)}</motion.p>
          <div className="mt-6 flex flex-wrap items-center justify-between gap-4 border-y border-rule py-4">
            <a href={"#author-" + a.author} className="group flex items-center gap-3">
              <Avatar a={au} size={44} />
              <span><span className="block font-semibold group-hover:text-accent">{t.by} {tr(au, "name", lang)}</span><span className="meta">{tr(au, "role", lang)}</span></span>
            </a>
            <div className="meta flex flex-wrap items-center gap-x-4 gap-y-1">
              {tr(a, "place", lang) && <span className="inline-flex items-center gap-1"><MapPin size={12} />{tr(a, "place", lang)}</span>}
              <time dateTime={a.published}>{fmtDate(a.published, lang)}</time>
              <span className="inline-flex items-center gap-1"><Clock size={12} />{a.readMin} {t.minRead}</span>
            </div>
            <div className="flex gap-2"><ShareBtn /><SaveBtn id={a.id} /></div>
          </div>
        </div>
      </header>

      {a.photo && (
        <figure className="wrap mt-8">
          <Photo src={a.photo.url} ratio="aspect-[16/9]" live={a.live} eager alt={tr(a, "title", lang)} />
          {credit && <figcaption className="meta mt-2 text-right normal-case tracking-normal">{a.live && <span className="mr-2 rounded-sm bg-soft px-1.5 py-0.5 font-mono text-[10px] uppercase text-accent-ink">{t.liveLabel}</span>}{credit}</figcaption>}
        </figure>
      )}

      <div className="wrap mt-10">
        <div className="mx-auto max-w-[680px]">
          {/* Коротко */}
          <Rise>
            <aside className="rounded-xl border-l-4 border-accent bg-soft/70 p-5">
              <div className="flex items-center justify-between gap-3">
                <p className="kicker">{t.tldr}</p>
                <div className="flex rounded-full bg-card p-0.5 text-[12px] font-semibold ring-1 ring-rule" role="group">
                  {[["short", t.deeperOff], ["full", t.deeperOn]].map(([id, l]) => (
                    <button key={id} onClick={() => setPref("level", id)} aria-pressed={(CN.prefs.level || "full") === id}
                      className={"relative rounded-full px-3 py-1 " + ((CN.prefs.level || "full") === id ? "text-onaccent" : "text-muted")}>
                      {(CN.prefs.level || "full") === id && <motion.span layoutId="lvl" className="absolute inset-0 rounded-full bg-accent" />}
                      <span className="relative">{l}</span>
                    </button>
                  ))}
                </div>
              </div>
              <p className="mt-2 font-serif text-[18px] leading-relaxed">{tr(a, "tldr", lang)}</p>
            </aside>
          </Rise>

          <div className="prose-cn mt-6">
            {items.map((it, k) => it.photos ? <Gallery key={"g" + k} photos={it.photos} lang={lang} credit={credit} /> : <Block key={k} b={it.b} lang={lang} t={t} id={a.id + "-" + it.i} />)}
          </div>

          {!deep && <button onClick={() => setPref("level", "full")} className="btn-line mt-2">{t.deeper} ↓</button>}
        </div>

        {deep && rest.length > 0 && <div className="mx-auto mt-6 max-w-5xl"><Gallery photos={rest} lang={lang} credit={credit} wide /></div>}

        <div className="mx-auto mt-10 max-w-[680px] space-y-8">
          {/* Источник */}
          {src && (
            <div className="rounded-xl border border-ink p-5">
              <p className="kicker">{t.source}</p>
              <p className="mt-2 font-serif text-[17px]">{a.layer === "world" ? t.retold + " " : ""}<b>{src.name}</b>{src.author && <> · {tr(src, "author", lang)}</>} · {fmtDate(src.date, lang)}</p>
              <a href={src.url} target="_blank" rel="noopener" className="btn-accent mt-4">{t.openOriginal} <ExternalLink size={15} /></a>
            </div>
          )}
          {/* Что сделать */}
          {a.actions?.length > 0 && (
            <div>
              <h2 className="h-title border-t-2 border-ink pt-3 text-[24px]">{t.whatToDo}</h2>
              <div className="mt-4 grid gap-4 sm:grid-cols-2">{a.actions.map((id, i) => CN.opp(id)).filter(Boolean).map((o, i) => <OppCard key={o.id} o={o} i={i} />)}</div>
            </div>
          )}
          {corrections.length > 0 && (
            <div className="border-t border-rule pt-4">
              <p className="kicker">{t.corrections}</p>
              {corrections.map((c, i) => <p key={i} className="mt-2 text-[15px] text-muted"><span className="font-mono text-[12px]">{fmtDate(c.date, lang)}</span> — {c.text}</p>)}
            </div>
          )}
          <div className="flex flex-wrap gap-2">{(tr(a, "tags", lang) || []).map((tg) => <span key={tg} className="rounded-full border border-rule px-3 py-1 text-[13px] text-muted">#{tg}</span>)}</div>
          <Comments a={a} />
        </div>
      </div>

      {related.length > 0 && (
        <section className="wrap mt-16">
          <SectionHead title={t.related} href={"#v-" + a.vertical} />
          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">{related.map((x) => <StoryCard key={x.id} a={x} size="sm" dek={false} />)}</div>
        </section>
      )}
    </article>
  );
}

function ShareBtn() {
  const t = useT();
  const [ok, setOk] = useState(false);
  return (
    <button onClick={() => { try { navigator.clipboard.writeText(location.href); } catch (_) {} setOk(true); setTimeout(() => setOk(false), 1800); }}
      aria-label={t.share} className="flex h-9 items-center gap-1.5 rounded-full border border-rule px-3 text-[13px] font-semibold hover:border-accent hover:text-accent">
      {ok ? <><Check size={14} /> {t.copied}</> : <><Link2 size={14} /> {t.share}</>}
    </button>
  );
}

function Gallery({ photos, lang, credit, wide }) {
  const cap = (p) => (lang === "en" ? p.cap_en : p.cap);
  if (photos.length === 1 && !wide) {
    const p = photos[0];
    return (
      <figure className="my-8 sm:-mx-16">
        <Photo src={p.src} ratio="aspect-[3/2]" alt={cap(p)} />
        <figcaption className="mt-2 font-sans text-[13px] text-muted">{cap(p)}{cap(p) && " · "}<span className="opacity-70">{credit}</span></figcaption>
      </figure>
    );
  }
  return (
    <div className={"my-8 grid gap-3 " + (wide ? "sm:grid-cols-3" : "sm:-mx-16 sm:grid-cols-2")}>
      {photos.map((p, i) => (
        <figure key={p.src} className={wide && i === 0 && photos.length % 3 === 1 ? "sm:col-span-3" : ""}>
          <Photo src={p.src} ratio={wide && i === 0 && photos.length % 3 === 1 ? "aspect-[21/9]" : "aspect-[4/3]"} alt={cap(p)} />
          <figcaption className="mt-2 text-[13px] leading-snug text-muted">{cap(p)}{cap(p) && " · "}<span className="opacity-70">{credit}</span></figcaption>
        </figure>
      ))}
    </div>
  );
}

function Block({ b, lang, t, id }) {
  if (b.t === "p") return <p>{b.text}</p>;
  if (b.t === "h") return <h2 className="h-title mb-2 mt-10 text-[26px]">{b.text}</h2>;
  if (b.t === "quote") return (
    <blockquote className="my-10 border-l-4 border-accent pl-5">
      <div className="h-title text-[26px] leading-[1.2] sm:text-[30px]">{lang === "en" ? "“" + b.text + "”" : "«" + b.text + "»"}</div>
      <footer className="meta mt-3">— {b.who}</footer>
    </blockquote>
  );
  if (b.t === "stats") return (
    <div className="my-8 grid grid-cols-1 gap-px overflow-hidden rounded-xl bg-rule sm:grid-cols-3">
      {b.items.map((s, i) => (
        <motion.div key={i} initial={{ opacity: 0, y: 12 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true }} transition={{ delay: i * 0.08 }} className="bg-card p-5">
          <p className="font-logo text-[30px] font-bold leading-none text-accent"><StatNum v={s.n} lang={lang} /></p><p className="mt-2 text-[14px] text-muted">{s.l}</p>
        </motion.div>
      ))}
    </div>
  );
  if (b.t === "facts") return (
    <div className="my-8 rounded-xl bg-band p-5"><p className="kicker">{b.title}</p>
      <ul className="mt-3 space-y-2">{b.items.map((x, i) => <li key={i} className="flex gap-2 text-[16px]"><span className="text-accent">■</span>{x}</li>)}</ul></div>
  );
  if (b.t === "timeline") return (
    <div className="my-10"><p className="kicker">{b.title}</p>
      <ol className="relative mt-4 border-l-2 border-accent/30 pl-6">
        {b.items.map((x, i) => (
          <motion.li key={i} initial={{ opacity: 0, x: -10 }} whileInView={{ opacity: 1, x: 0 }} viewport={{ once: true }} transition={{ delay: i * 0.05 }} className="relative pb-5 last:pb-0">
            <span className="absolute -left-[31px] top-1.5 h-3 w-3 rounded-full border-2 border-accent bg-paper" />
            <span className="font-mono text-[13px] font-semibold text-accent">{x.y}</span><span className="block text-[16px] leading-snug">{x.text}</span>
          </motion.li>
        ))}
      </ol></div>
  );
  if (b.t === "quiz") return <Quiz b={b} t={t} />;
  if (b.t === "poll") return <Poll b={b} t={t} id={id} />;
  return null;
}

function Quiz({ b, t }) {
  const [pick, setPick] = useState(null);
  return (
    <div className="my-10 rounded-xl border-2 border-ink p-5">
      <p className="kicker">{t.quiz}</p><p className="h-title mt-2 text-[20px]">{b.q}</p>
      <div className="mt-4 grid gap-2">
        {b.options.map((o, i) => {
          const st = pick == null ? "" : i === b.answer ? "border-accent bg-soft" : i === pick ? "border-ink/40 opacity-60" : "opacity-60";
          return <button key={i} disabled={pick != null} onClick={() => setPick(i)} className={"flex items-center justify-between rounded-lg border border-rule px-4 py-3 text-left font-semibold transition " + st}>{o}{pick != null && i === b.answer && <Check size={16} className="text-accent" />}{pick === i && i !== b.answer && <XIcon size={16} />}</button>;
        })}
      </div>
      {pick != null && <motion.p initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} className="mt-3 text-[15px]"><b>{pick === b.answer ? t.quizRight : t.quizWrong}</b> {b.explain}</motion.p>}
    </div>
  );
}
function Poll({ b, t, id }) {
  const polls = CN.prefs.polls || {};
  const mine = polls[id];
  const base = [41, 33, 26];
  const votes = base.map((v, i) => v + (mine === i ? 1 : 0));
  const sum = votes.reduce((x, y) => x + y, 0);
  return (
    <div className="my-10 rounded-xl bg-band p-5">
      <p className="h-title text-[20px]">{b.q}</p>
      <div className="mt-4 grid gap-2">
        {b.options.map((o, i) => (
          <button key={i} disabled={mine != null} onClick={() => { setPref("polls", { ...polls, [id]: i }); }} className="relative overflow-hidden rounded-lg border border-rule bg-card px-4 py-3 text-left font-semibold">
            {mine != null && <motion.span initial={{ width: 0 }} animate={{ width: `${(votes[i] / sum) * 100}%` }} transition={{ duration: 0.8, ease }} className={"absolute inset-y-0 left-0 " + (mine === i ? "bg-accent/25" : "bg-ink/[.06]")} />}
            <span className="relative flex justify-between">{o}{mine != null && <span className="font-mono text-[13px]">{Math.round((votes[i] / sum) * 100)}%</span>}</span>
          </button>
        ))}
      </div>
      {mine != null && <p className="meta mt-3">{t.pollThanks} · {sum} {t.votes}</p>}
    </div>
  );
}

function Comments({ a }) {
  const t = useT(), lang = useLang();
  const [txt, setTxt] = useState(""), [sent, setSent] = useState(false);
  const list = CN.state.comments.filter((c) => c.article === a.id && c.status === "published" && (c.lang || "ru") === lang);
  function send(e) {
    e.preventDefault();
    if (!txt.trim()) return;
    CN.state.comments.push({ id: CN.uid("c"), article: a.id, name: lang === "en" ? "Guest" : "Гость", text: txt.trim(), status: "pending", date: new Date().toISOString(), lang });
    CN.save(); bump(); setTxt(""); setSent(true);
  }
  return (
    <section>
      <h2 className="h-title border-t-2 border-ink pt-3 text-[24px]">{t.comments} <span className="font-mono text-[14px] text-muted">{list.length}</span></h2>
      {list.length ? list.map((c) => (
        <div key={c.id} className="border-b border-rule py-4"><p className="font-semibold">{c.name} <span className="meta ml-2">{fmtDate(c.date, lang, { day: "numeric", month: "short" })}</span></p><p className="mt-1 font-serif text-[17px]">{c.text}</p></div>
      )) : <p className="mt-3 text-muted">{t.noComments}</p>}
      <form onSubmit={send} className="mt-4">
        <textarea value={txt} onChange={(e) => setTxt(e.target.value)} rows={3} placeholder={t.commentPh} aria-label={t.comments} className="w-full rounded-lg border border-rule bg-card p-3 font-serif text-[16px] outline-none focus:border-accent" />
        <div className="mt-2 flex items-center gap-3"><button className="btn-accent">{t.send}</button>{sent && <span className="text-[14px] text-accent">{t.sentMod}</span>}</div>
      </form>
    </section>
  );
}

/* Анимированное число (Skiper UI skiper37), если значение — целое с префиксом/суффиксом */
function StatNum({ v, lang }) {
  const m = /^([^\d]*)(\d{1,3}(?:[ ,\u00a0]\d{3})*|\d+)([^\d]*)$/.exec(v);
  if (!m) return v;
  const n = parseInt(m[2].replace(/[ ,\u00a0]/g, ""), 10);
  return <AnimatedNumber_004 to={n} prefix={m[1]} suffix={m[3]} locales={lang === "en" ? "en-US" : "ru-RU"} />;
}
