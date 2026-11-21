import React, { useEffect, useMemo, useRef, useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { Heart, Bookmark, BookmarkCheck, Share2, X, ChevronUp, ChevronDown, Play, Pause, ArrowRight } from "lucide-react";
import { useLenis } from "lenis/react";
import { CN, useLang, tr, published, vertName, authorName, togglePref, ago } from "./lib.js";
import { ccOf, ccName, flag } from "./geo.js";

/* ---------- Короткие новости в формате рилсов ----------
   Вертикальная лента 9:16: листается свайпом/колесом/стрелками,
   субтитры из TL;DR идут по фразам, после последней — следующая новость. */

const sentences = (s) => ((s || "").match(/[^.!?…]+[.!?…]+["»”]?/g) || [s || ""]).map((x) => x.trim()).filter(Boolean);
const STEP = 3200;
export const shortsList = (lang) => published(lang).filter((a) => a.photo && tr(a, "tldr", lang));

function Slide({ a, i, active, lang, onDone, paused, setPaused }) {
  const caps = useMemo(() => sentences(tr(a, "tldr", lang)), [a, lang]);
  const [k, setK] = useState(0);
  const [likes] = useState(() => 120 + ((a.views * 7) % 900));
  useEffect(() => { if (active) setK(0); }, [active]);
  useEffect(() => {
    if (!active || paused) return;
    const t = setTimeout(() => (k < caps.length - 1 ? setK(k + 1) : onDone()), STEP);
    return () => clearTimeout(t);
  }, [active, paused, k, caps.length]);
  const liked = (CN.prefs.liked || []).includes(a.id), saved = (CN.prefs.saved || []).includes(a.id);
  const share = async () => {
    const url = location.href.split("#")[0] + "#" + a.id;
    try { if (navigator.share) await navigator.share({ title: tr(a, "title", lang), url }); else { await navigator.clipboard.writeText(url); } } catch (_) {}
  };
  const cc = a.layer === "world" ? ccOf(a)[0] : null;
  const Rail = ({ icon, label, on, onClick }) => (
    <button onClick={(e) => { e.stopPropagation(); onClick(); }} className="flex flex-col items-center gap-1 text-white" aria-pressed={on}>
      <motion.span whileTap={{ scale: 0.8 }} className={"grid h-12 w-12 place-items-center rounded-full backdrop-blur " + (on ? "bg-white text-[#4B44F0]" : "bg-white/15")}>{icon}</motion.span>
      <span className="font-mono text-[11px] font-semibold">{label}</span>
    </button>
  );
  return (
    <section data-i={i} className="relative h-full w-full shrink-0 snap-start snap-always overflow-hidden bg-[#0b0b14]" onClick={() => setPaused(!paused)}>
      <img src={a.photo.url} alt="" className={"absolute inset-0 h-full w-full object-cover " + (active && !paused ? "kb" : "")} loading={i < 2 ? "eager" : "lazy"} />
      <div className="absolute inset-0 bg-[linear-gradient(180deg,rgba(0,0,0,.55)_0%,rgba(0,0,0,0)_22%,rgba(0,0,0,0)_45%,rgba(0,0,0,.85)_100%)]" />
      {/* прогресс по фразам */}
      <div className="absolute left-3 right-3 top-3 flex gap-1">
        {caps.map((_, j) => (
          <span key={j} className="h-[3px] flex-1 overflow-hidden rounded-full bg-white/30">
            <motion.span className="block h-full bg-white" initial={false}
              animate={{ width: !active ? "0%" : j < k ? "100%" : j === k ? (paused ? undefined : "100%") : "0%" }}
              transition={{ duration: active && j === k && !paused ? STEP / 1000 : 0, ease: "linear" }} />
          </span>
        ))}
      </div>
      <div className="absolute left-4 right-4 top-7 flex items-center gap-2 text-white">
        <span className="rounded-full bg-[#4B44F0] px-2.5 py-1 font-mono text-[10px] font-semibold uppercase tracking-wider">{lang === "en" ? "Short" : "Коротко"}</span>
        <span className="truncate font-mono text-[10px] uppercase tracking-wider text-white/80">{vertName(a.vertical, lang)}{cc ? " · " + flag(cc) + " " + ccName(cc, lang) : ""}</span>
      </div>
      {/* субтитры */}
      <div className="absolute inset-x-5 top-[30%] flex justify-center">
        <AnimatePresence mode="wait">
          {active && (
            <motion.p key={k} initial={{ opacity: 0, y: 18, scale: 0.96 }} animate={{ opacity: 1, y: 0, scale: 1 }} exit={{ opacity: 0, y: -12 }} transition={{ duration: 0.35 }}
              className="max-w-[92%] text-center font-head text-[22px] font-bold leading-[1.2] text-white [text-shadow:0_2px_16px_rgba(0,0,0,.7)] sm:text-[24px]">
              <span className="box-decoration-clone rounded bg-black/45 px-2 py-0.5">{caps[k]}</span>
            </motion.p>
          )}
        </AnimatePresence>
      </div>
      {paused && active && <span className="absolute left-1/2 top-1/2 grid h-16 w-16 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full bg-black/50 text-white"><Play size={28} /></span>}
      {/* правая колонка */}
      <div className="absolute bottom-28 right-3 flex flex-col gap-4">
        <Rail icon={<Heart size={22} fill={liked ? "currentColor" : "none"} />} label={likes + (liked ? 1 : 0)} on={liked} onClick={() => togglePref("liked", a.id)} />
        <Rail icon={saved ? <BookmarkCheck size={22} /> : <Bookmark size={22} />} label={lang === "en" ? "Save" : "В закладки"} on={saved} onClick={() => togglePref("saved", a.id)} />
        <Rail icon={<Share2 size={21} />} label={lang === "en" ? "Share" : "Поделиться"} onClick={share} />
      </div>
      {/* низ */}
      <div className="absolute bottom-0 left-0 right-20 p-4 pb-6 text-white">
        <p className="font-mono text-[11px] font-semibold">@{(CN.author(a.author)?.id || "creata")} · {ago(a.published, lang)}</p>
        <h2 className="mt-1.5 font-head text-[19px] font-bold leading-tight">{tr(a, "title", lang)}</h2>
        <a href={"#" + a.id} onClick={(e) => e.stopPropagation()} className="mt-3 inline-flex h-9 items-center gap-1.5 rounded-full bg-white px-3.5 text-[13px] font-semibold text-[#0E0E10]">
          {lang === "en" ? "Read the full story" : "Читать полностью"} <ArrowRight size={14} />
        </a>
      </div>
    </section>
  );
}

export function Shorts({ start }) {
  const lang = useLang();
  const list = useMemo(() => shortsList(lang), [lang]);
  const first = Math.max(0, list.findIndex((a) => a.id === start));
  const [idx, setIdx] = useState(first);
  const [paused, setPaused] = useState(false);
  const box = useRef();
  const lenis = useLenis();
  useEffect(() => { lenis?.stop(); document.body.style.overflow = "hidden"; return () => { lenis?.start(); document.body.style.overflow = ""; }; }, [lenis]);
  useEffect(() => { const el = box.current?.children[first]; if (el) box.current.scrollTop = el.offsetTop; }, []);
  useEffect(() => {
    const io = new IntersectionObserver((es) => es.forEach((e) => { if (e.isIntersecting) { setIdx(+e.target.dataset.i); setPaused(false); } }), { root: box.current, threshold: 0.6 });
    [...box.current.children].forEach((c) => io.observe(c));
    return () => io.disconnect();
  }, [list.length]);
  useEffect(() => { const a = list[idx]; if (a) history.replaceState(null, "", "#shorts-" + a.id); }, [idx]);
  const go = (n) => { const el = box.current?.children[Math.max(0, Math.min(list.length - 1, n))]; el && box.current.scrollTo({ top: el.offsetTop, behavior: "smooth" }); };
  useEffect(() => {
    const f = (e) => { if (e.key === "ArrowDown") { e.preventDefault(); go(idx + 1); } if (e.key === "ArrowUp") { e.preventDefault(); go(idx - 1); } if (e.key === " ") { e.preventDefault(); setPaused((p) => !p); } if (e.key === "Escape") close(); };
    window.addEventListener("keydown", f); return () => window.removeEventListener("keydown", f);
  });
  const close = () => { location.hash = ""; };
  return (
    <div className="fixed inset-0 z-[80] flex items-center justify-center bg-[#0b0b14]">
      {/* фон на десктопе: размытая обложка текущей новости */}
      {list[idx] && <img src={list[idx].photo.url} alt="" aria-hidden="true" className="absolute inset-0 hidden h-full w-full scale-110 object-cover opacity-35 blur-2xl sm:block" />}
      <div className="absolute left-4 top-4 z-10 hidden items-center gap-3 text-white sm:flex">
        <span className="font-logo text-[18px] font-bold">CR<span className="logo-e" />ATA</span>
        <span className="font-mono text-[11px] uppercase tracking-wider text-white/70">{lang === "en" ? "Shorts" : "Коротко"} · {idx + 1}/{list.length}</span>
      </div>
      <button onClick={close} aria-label={lang === "en" ? "Close" : "Закрыть"} className="absolute right-3 top-12 z-20 grid h-10 w-10 place-items-center rounded-full bg-black/40 text-white backdrop-blur sm:right-5 sm:top-5 sm:bg-white/15"><X size={20} /></button>
      <div ref={box} data-lenis-prevent className="no-scrollbar relative h-full w-full snap-y snap-mandatory overflow-y-scroll overscroll-contain sm:h-[min(92vh,860px)] sm:w-auto sm:aspect-[9/16] sm:rounded-[28px] sm:shadow-[0_40px_120px_-30px_rgba(0,0,0,.8)] sm:ring-1 sm:ring-white/10">
        {list.map((a, i) => <Slide key={a.id} a={a} i={i} lang={lang} active={i === idx} paused={paused} setPaused={setPaused} onDone={() => go(i + 1 < list.length ? i + 1 : 0)} />)}
      </div>
      <div className="absolute right-6 top-1/2 z-10 hidden -translate-y-1/2 flex-col gap-3 sm:flex">
        <button onClick={() => go(idx - 1)} disabled={idx === 0} aria-label={lang === "en" ? "Previous" : "Предыдущая"} className="grid h-12 w-12 place-items-center rounded-full bg-white/15 text-white backdrop-blur hover:bg-white/25 disabled:opacity-30"><ChevronUp /></button>
        <button onClick={() => go(idx + 1)} disabled={idx === list.length - 1} aria-label={lang === "en" ? "Next" : "Следующая"} className="grid h-12 w-12 place-items-center rounded-full bg-white/15 text-white backdrop-blur hover:bg-white/25 disabled:opacity-30"><ChevronDown /></button>
      </div>
      <p className="absolute bottom-4 left-1/2 hidden -translate-x-1/2 font-mono text-[10px] uppercase tracking-wider text-white/50 sm:block">{lang === "en" ? "Scroll or ↑ ↓ · space to pause" : "Листайте колесом или ↑ ↓ · пробел — пауза"}</p>
    </div>
  );
}

/* полоса на главной */
export function ShortsStrip() {
  const lang = useLang();
  const list = shortsList(lang).slice(0, 10);
  return (
    <section className="wrap mt-16">
      <div className="mb-5 flex items-end justify-between gap-4 border-t-2 border-ink pt-3">
        <div><h2 className="h-title text-[28px] sm:text-[34px]">{lang === "en" ? "Shorts" : "Коротко"}</h2><p className="mt-1 font-serif text-[16px] text-muted">{lang === "en" ? "The news in 15 seconds. Swipe like reels." : "Новости за 15 секунд. Листайте как рилсы."}</p></div>
        <a href={"#shorts-" + (list[0]?.id || "")} className="btn-accent shrink-0"><Play size={15} /> {lang === "en" ? "Watch" : "Смотреть"}</a>
      </div>
      <div data-lenis-prevent className="no-scrollbar -mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-2 sm:mx-0 sm:px-0">
        {list.map((a, i) => (
          <motion.a key={a.id} href={"#shorts-" + a.id} whileHover={{ y: -6 }} className="group relative aspect-[9/16] w-[150px] shrink-0 snap-start overflow-hidden rounded-2xl bg-[#0b0b14] sm:w-[180px]">
            <img src={a.photo.url} alt="" loading="lazy" className="absolute inset-0 h-full w-full object-cover transition-transform duration-700 group-hover:scale-105" />
            <span className="absolute inset-0 bg-gradient-to-b from-black/10 via-transparent to-black/85" />
            <span className="absolute left-2 top-2 grid h-7 w-7 place-items-center rounded-full bg-white/90 text-[#0E0E10]"><Play size={13} fill="currentColor" /></span>
            <span className="absolute bottom-0 p-3 text-[13.5px] font-semibold leading-tight text-white">{tr(a, "title", lang)}</span>
          </motion.a>
        ))}
      </div>
    </section>
  );
}
