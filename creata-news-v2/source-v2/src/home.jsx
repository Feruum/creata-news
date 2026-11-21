import React, { useState } from "react";
import { motion } from "motion/react";
import { ArrowRight, Bell, BellRing, Send, Newspaper } from "lucide-react";
import { CN, useLang, tr, published, vertName, fmtDate, togglePref, ago } from "./lib.js";
import { useT } from "./i18n.js";
import { ShortsStrip } from "./shorts.jsx";
import { byCountry, ccName, flag } from "./geo.js";
import { Skiper16 } from "./skiper/skiper16.jsx";
import { Carousel_006 } from "./skiper/skiper54.jsx";
import { SpringMouseFollow } from "./skiper/skiper61.jsx";
import { StoryCard, SectionHead, Photo, Kicker, Byline, Countdown, ImageTrail, DragStrip, HoverMembers, Rise, Logo, LabelTag, Magnetic, ease } from "./ui.jsx";

export function Home({ onSubscribe }) {
  const lang = useLang(), t = useT();
  const all = published(lang);
  const world = all.filter((a) => a.layer === "world");
  const eco = all.filter((a) => a.layer === "eco");
  const hero = all[0], side = all.slice(1, 3);
  const latest = all.slice(0, 9);
  const row = all.slice(3, 7);
  const picks = [...all].sort((a, b) => b.saves - a.saves).slice(0, 4);
  const feature = CN.article("a-mosques");
  const follows = CN.prefs.follows || [];
  const forYou = follows.length ? all.filter((a) => follows.includes(a.vertical)).slice(0, 4) : [];

  return (
    <div>
      {/* ---------- Верх: Последнее · Главное · Ещё ---------- */}
      <section className="wrap pt-6 sm:pt-8">
        <div className="grid gap-8 lg:grid-cols-12">
          <div className="order-2 lg:order-2 lg:col-span-6">
            <Rise><SpringMouseFollow label={lang === "en" ? "Read →" : "Читать →"}><StoryCard a={hero} size="xl" /></SpringMouseFollow></Rise>
          </div>
          <aside className="order-3 lg:order-1 lg:col-span-3">
            <div className="border border-ink p-4">
              <h2 className="flex items-center justify-between border-b border-ink pb-2">
                <span className="h-title text-[20px]">{t.latest}</span>
                <span className="relative flex h-2.5 w-2.5"><span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent opacity-60" /><span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-accent" /></span>
              </h2>
              <ol className="divide-y divide-rule">
                {latest.map((a, i) => (
                  <motion.li key={a.id} initial={{ opacity: 0, x: -10 }} whileInView={{ opacity: 1, x: 0 }} viewport={{ once: true }} transition={{ delay: i * 0.04 }} className="py-3">
                    <a href={"#" + a.id} className="group block">
                      <span className="meta">{ago(a.published, lang)} · <span className="text-accent">{vertName(a.vertical, lang)}</span></span>
                      <span className="mt-1 block text-[16px] font-semibold leading-snug group-hover:text-accent">{tr(a, "title", lang)}</span>
                    </a>
                  </motion.li>
                ))}
              </ol>
              <a href="#world" className="meta mt-2 inline-flex items-center gap-1 hover:text-accent">{t.allStories} <ArrowRight size={12} /></a>
            </div>
          </aside>
          <div className="order-4 grid content-start gap-8 sm:grid-cols-2 lg:col-span-3 lg:grid-cols-1">
            {side.map((a, i) => <Rise key={a.id} delay={0.1 + i * 0.08}><StoryCard a={a} size="md" dek={false} /></Rise>)}
          </div>
        </div>
      </section>

      {/* ---------- Ряд из четырёх ---------- */}
      <section className="wrap mt-12">
        <div className="grid gap-8 border-t border-rule pt-8 sm:grid-cols-2 lg:grid-cols-4">
          {row.map((a, i) => <Rise key={a.id} delay={i * 0.06}><StoryCard a={a} size="sm" dek={false} /></Rise>)}
        </div>
      </section>

      <ShortsStrip />

      {/* ---------- Утро Creata ---------- */}
      <Newsletter onSubscribe={onSubscribe} />

      {forYou.length > 0 && (
        <section className="wrap mt-16">
          <SectionHead title={t.forYou} lead={t.forYouLead} />
          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">{forYou.map((a) => <StoryCard key={a.id} a={a} size="sm" />)}</div>
        </section>
      )}

      {/* ---------- Мир ---------- */}
      <section className="wrap mt-16">
        <SectionHead title={t.world} href="#world" lead={t.worldLead} />
        <nav aria-label={t.allCountries} className="-mt-2 mb-8 flex gap-2 overflow-x-auto pb-1">
          <a href="#world" className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-full bg-[#0b0b14] px-3.5 text-sm font-semibold text-white">🌐 {lang === "en" ? `Open the globe · ${byCountry(world).length} countries` : `Открыть глобус · ${byCountry(world).length} стран`}</a>
        </nav>
        <div className="grid gap-8 lg:grid-cols-12">
          <div className="lg:col-span-7"><StoryCard a={world[0]} size="lg" /></div>
          <div className="grid gap-6 lg:col-span-5">
            {world.slice(1, 4).map((a) => (
              <article key={a.id} className="group grid grid-cols-[1fr_120px] gap-4 border-b border-rule pb-6 last:border-0 sm:grid-cols-[1fr_180px]">
                <div><Kicker a={a} /><h3 className="h-title mt-1 text-[19px]"><a href={"#" + a.id} className="hover-u">{tr(a, "title", lang)}</a></h3><Byline a={a} className="mt-2" /></div>
                <a href={"#" + a.id} tabIndex={-1} aria-hidden="true"><Photo src={a.photo.url} ratio="aspect-[4/3]" /></a>
              </article>
            ))}
          </div>
        </div>
      </section>

      {/* ---------- Экосистема Creata: след из фото за курсором ---------- */}
      <section className="mt-16 bg-band">
        <ImageTrail images={CN.photoLib().filter((p) => p.src.includes("/c")).map((p) => p.src).slice(0, 30)} className="border-b border-rule">
          <div className="wrap flex min-h-[260px] flex-col justify-center py-14 sm:min-h-[340px]">
            <p className="kicker">{t.ecoCursor}</p>
            <h2 className="h-title mt-3 max-w-5xl text-[40px] sm:text-[64px] lg:text-[88px]">
              <a href="#ecosystem" className="hover:text-accent">{t.eco}<span className="whitespace-nowrap">&nbsp;→</span></a>
            </h2>
            <p className="mt-4 max-w-xl font-serif text-[18px] text-muted">{t.ecoLead}</p>
          </div>
        </ImageTrail>
        <Skiper16 items={eco.slice(0, 5).map((a) => ({ title: tr(a, "title", lang), kicker: vertName(a.vertical, lang), src: a.photo.url, href: "#" + a.id }))} />
        <div className="wrap pb-12 text-center"><a href="#ecosystem" className="btn-line">{t.seeAll} · {eco.length}</a></div>
      </section>

      {/* ---------- Возможности ---------- */}
      <OppsBand />

      {/* ---------- Фича: большой материал ---------- */}
      {feature && hasEn(feature, lang) && (
        <section className="wrap mt-16">
          <SectionHead title={lang === "en" ? "Features" : "Большие истории"} />
          <article className="group grid items-center gap-8 lg:grid-cols-12">
            <a href={"#" + feature.id} tabIndex={-1} aria-hidden="true" className="lg:col-span-8"><Photo src={feature.photo.url} ratio="aspect-[16/9]" live /></a>
            <div className="lg:col-span-4">
              <Kicker a={feature} />
              <h3 className="h-title mt-2 text-[30px] sm:text-[40px]"><a href={"#" + feature.id} className="hover-u">{tr(feature, "title", lang)}</a></h3>
              <p className="mt-3 font-serif text-[19px] leading-snug text-muted">{tr(feature, "lead", lang)}</p>
              <Byline a={feature} className="mt-4" />
            </div>
          </article>
        </section>
      )}

      {/* ---------- Выбор редакции ---------- */}
      <section className="mt-16 bg-ink py-14 text-paper">
        <div className="wrap">
          <SectionHead title={t.editorsPicks} invert />
          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
            {picks.map((a, i) => (
              <Rise key={a.id} delay={i * 0.06}>
                <article className="group">
                  <a href={"#" + a.id} tabIndex={-1} aria-hidden="true"><Photo src={a.photo.url} ratio="aspect-[4/5]" /></a>
                  <span className="mt-3 block font-mono text-[11px] uppercase tracking-wider text-[#8C87FF]">{String(i + 1).padStart(2, "0")} · {vertName(a.vertical, lang)}</span>
                  <h3 className="h-title mt-1 text-[19px]"><a href={"#" + a.id} className="hover-u">{tr(a, "title", lang)}</a></h3>
                </article>
              </Rise>
            ))}
          </div>
        </div>
      </section>

      {/* ---------- Газета №1 ---------- */}
      <IssueBand />

      {/* ---------- Разделы ---------- */}
      <section className="wrap mt-16">
        <div className="grid gap-10 md:grid-cols-3">
          {[["eco-games", "#v-eco-games"], ["eco-transfornation", "#v-eco-transfornation"], ["world", "#world"]].map(([v, href]) => {
            const list = all.filter((a) => (v === "world" ? a.layer === "world" : a.vertical === v));
            if (!list.length) return null;
            return (
              <div key={v}>
                <SectionHead title={v === "world" ? t.world : vertName(v, lang)} href={href} className="!mb-4" compact />
                <StoryCard a={list[v === "world" ? 1 : 0]} size="sm" dek={false} />
                <ul className="mt-4 divide-y divide-rule border-t border-rule">
                  {list.filter((x, i) => i !== (v === "world" ? 1 : 0)).slice(0, 3).map((a) => <li key={a.id} className="py-3"><a href={"#" + a.id} className="text-[16px] font-semibold leading-snug hover:text-accent">{tr(a, "title", lang)}</a></li>)}
                </ul>
              </div>
            );
          })}
        </div>
      </section>

      {/* ---------- Редакция ---------- */}
      <section className="wrap mt-16">
        <div className="flex flex-col items-start justify-between gap-6 border-y border-ink py-8 sm:flex-row sm:items-center">
          <div><h2 className="h-title text-[26px]">{t.team}</h2><p className="mt-1 max-w-xl font-serif text-muted">{t.teamLead}</p></div>
          <div className="flex items-center gap-5"><HoverMembers people={CN.state.authors} max={8} /><a href="#authors" className="btn-line">{t.authors}</a></div>
        </div>
      </section>
    </div>
  );
}
const hasEn = (a, lang) => lang !== "en" || a.title_en;

export function Newsletter({ onSubscribe }) {
  const t = useT(), lang = useLang();
  const dig = CN.prefs.digest;
  const top = published(lang).slice(0, 3);
  return (
    <section className="mt-16 bg-[#4B44F0] text-white">
      <div className="wrap grid items-center gap-10 py-14 lg:grid-cols-2">
        <div>
          <p className="font-mono text-[11px] uppercase tracking-wider opacity-80">08:00 · Telegram / email</p>
          <h2 className="h-title mt-3 text-[40px] sm:text-[60px]">{t.newsletterTitle}</h2>
          <p className="mt-4 max-w-lg font-serif text-[19px] opacity-90">{t.newsletterLead}</p>
          {dig ? <p className="mt-6 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-2 font-semibold">✓ {t.subscribed}</p> : (
            <div className="mt-7 flex flex-wrap items-center gap-3">
              <Magnetic><button onClick={onSubscribe} className="btn h-12 bg-white px-6 text-[15px] text-accent hover:bg-[#0E0E10] hover:text-white">{t.subscribe}</button></Magnetic>
              <a href="https://t.me/creata_news" target="_blank" rel="noopener" className="inline-flex items-center gap-2 font-semibold hover:underline"><Send size={16} /> {t.telegram}</a>
            </div>
          )}
        </div>
        {/* телефон с выпуском */}
        <div className="flex justify-center lg:justify-end">
          <motion.div initial={{ rotate: 6, y: 40, opacity: 0 }} whileInView={{ rotate: -3, y: 0, opacity: 1 }} viewport={{ once: true }} transition={{ duration: 0.9, ease }}
            className="w-[280px] rounded-[38px] border-[10px] border-[#0E0E10] bg-[#FBFAF7] p-4 text-[#0E0E10] shadow-2xl">
            <div className="mx-auto mb-3 h-5 w-24 rounded-full bg-[#0E0E10]" />
            <Logo className="text-[15px]" />
            <p className="mt-1 font-mono text-[9px] uppercase tracking-wider text-[#5B5B60]">{t.newsletterTitle} · 08:00</p>
            {top.map((a, i) => (
              <div key={a.id} className="mt-3 flex gap-2 border-t border-[#CACAC6] pt-3">
                <img src={a.photo.url} alt="" className="h-12 w-12 shrink-0 rounded object-cover" />
                <p className="text-[12px] font-semibold leading-tight">{tr(a, "title", lang)}</p>
              </div>
            ))}
            <div className="mt-3 rounded-lg bg-[#4B44F0] p-3 text-[11px] text-white">{lang === "en" ? "Game of the Day: guess the 5-letter word →" : "Игра дня: угадай слово из 5 букв →"}</div>
          </motion.div>
        </div>
      </div>
    </section>
  );
}

export function OppCard({ o, i = 0, dark }) {
  const t = useT(), lang = useLang();
  const on = (CN.prefs.reminders || []).includes(o.id);
  return (
    <motion.article initial={{ opacity: 0, y: 20 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true }} transition={{ delay: i * 0.06 }}
      className={"group flex flex-col overflow-hidden rounded-xl " + (dark ? "bg-white/[.06] ring-1 ring-white/15" : "bg-card ring-1 ring-rule")}>
      {o.photo && <div className="relative aspect-[16/9] overflow-hidden"><img src={o.photo} alt="" loading="lazy" className="h-full w-full object-cover transition duration-700 group-hover:scale-105" />
        <span className="absolute left-3 top-3 rounded-full bg-[#0E0E10]/80 px-2.5 py-1 font-mono text-[10px] uppercase tracking-wider text-white">{tr(o, "type", lang)}</span></div>}
      <div className="flex flex-1 flex-col p-4">
        <p className={"font-mono text-[11px] uppercase tracking-wider " + (dark ? "text-[#8C87FF]" : "text-accent")}>{o.org} · {tr(o, "country", lang)}</p>
        <h3 className="h-title mt-1.5 text-[18px]">{tr(o, "title", lang)}</h3>
        <p className={"mt-1.5 text-[14px] " + (dark ? "opacity-70" : "text-muted")}>{t.forWhom}: {tr(o, "forWhom", lang)}</p>
        <div className="mt-auto pt-4">
          <p className={"mb-1.5 font-mono text-[10px] uppercase tracking-wider " + (dark ? "opacity-60" : "text-muted")}>{t.deadline}: {fmtDate(o.deadline, lang, { day: "numeric", month: "long" })}</p>
          <Countdown iso={o.deadline} />
          <div className="mt-4 flex gap-2">
            <a href={o.url} target="_blank" rel="noopener" className={"btn h-9 flex-1 text-[13px] " + (dark ? "bg-white text-[#0E0E10] hover:bg-[#8C87FF]" : "bg-ink text-paper hover:bg-accent")}>{t.apply}</a>
            <button onClick={() => togglePref("reminders", o.id)} aria-pressed={on} className={"btn h-9 border text-[13px] " + (dark ? "border-white/30 hover:bg-white/10" : "border-rule hover:border-ink")}>
              {on ? <BellRing size={14} /> : <Bell size={14} />}{on ? t.reminded : t.remind}
            </button>
          </div>
        </div>
      </div>
    </motion.article>
  );
}

function OppsBand() {
  const t = useT();
  const opps = CN.liveOpps().slice(0, 4);
  return (
    <section className="bg-[#0E0E10] py-14 text-[#EDECE8]">
      <div className="wrap">
        <div className="mb-6 flex items-end justify-between border-t-2 border-white pt-3">
          <div><h2 className="h-title text-[26px] sm:text-[32px]"><a href="#opportunities" className="hover:text-[#8C87FF]">{t.oppsTitle} →</a></h2><p className="mt-1 font-serif opacity-70">{t.oppsLead}</p></div>
        </div>
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">{opps.map((o, i) => <OppCard key={o.id} o={o} i={i} dark />)}</div>
      </div>
    </section>
  );
}

function IssueBand() {
  const t = useT(), lang = useLang();
  const pics = published(lang).filter((a) => a.layer === "eco").map((a) => ({ src: a.photo.url, alt: "", title: tr(a, "title", lang) }));
  return (
    <section className="mt-16 overflow-hidden">
      <div className="wrap grid items-end gap-6 lg:grid-cols-[1fr_auto]">
        <div>
          <p className="kicker">{tr(CN.state.issue, "month", lang)} · {tr(CN.state.issue, "city", lang)}</p>
          <h2 className="h-title mt-2 text-[44px] sm:text-[72px]">{t.issue} <span className="text-accent">№01</span></h2>
          <p className="mt-2 max-w-xl font-serif text-[18px] text-muted">{t.issueLead}</p>
        </div>
        <a href="#issue" className="btn-line h-11 self-start lg:self-end"><Newspaper size={16} /> {t.openIssue}</a>
      </div>
      <div className="mt-8 pb-8"><Carousel_006 images={pics} /></div>
    </section>
  );
}
