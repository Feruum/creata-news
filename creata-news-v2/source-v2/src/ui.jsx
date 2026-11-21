import { ccOf, ccName } from "./geo.js";
import React, { useEffect, useRef, useState } from "react";
import { motion, AnimatePresence, useMotionValue, useSpring, useTransform } from "motion/react";
import { ChevronRight, Bookmark, BookmarkCheck } from "lucide-react";
import { CN, useLang, tr, vertName, authorName, fmtDate, ago, togglePref, useNow } from "./lib.js";
import { useT } from "./i18n.js";
import { TextRoll } from "./skiper/skiper58.jsx";
import { FlowDigit } from "./skiper/skiper37.jsx";

export const ease = [0.2, 0.7, 0.2, 1];

/* ---------- Логотип CR≡ATA NEWS ---------- */
export function Logo({ className = "", small }) {
  return (
    <span className={"font-logo font-bold leading-none tracking-[-0.02em] whitespace-nowrap " + className} aria-label="Creata News">
      <span aria-hidden="true">CR<span className="logo-e" />ATA</span>
      {!small && <span aria-hidden="true" className="ml-[0.28em] font-medium">NEWS</span>}
    </span>
  );
}

/* ---------- Фото с раскрытием при прокрутке (skiper «Image reveal») ---------- */
export function Photo({ src, alt = "", className = "", ratio = "aspect-[3/2]", live, reveal = true, eager, imgClass = "", children }) {
  const reduce = document.documentElement.classList.contains("reduce");
  return (
    <motion.div
      className={"relative overflow-hidden bg-band " + ratio + " " + className}
      initial={reveal && !reduce ? { clipPath: "inset(0 0 100% 0)" } : false}
      whileInView={{ clipPath: "inset(0 0 0% 0)" }}
      viewport={{ once: true, margin: "0px 0px -8% 0px" }}
      transition={{ duration: 0.9, ease }}
    >
      <img src={src} alt={alt} loading={eager ? "eager" : "lazy"} decoding="async"
        className={"absolute inset-0 h-full w-full object-cover transition-transform duration-700 ease-out group-hover:scale-[1.035] " + (live ? "kb " : "") + imgClass} />
      {children}
    </motion.div>
  );
}

/* ---------- Мелочи ---------- */
export function Kicker({ a, className = "" }) {
  const lang = useLang();
  return (
    <span className={"kicker " + className}>
      <a href={"#v-" + a.vertical} className="hover:underline" onClick={(e) => e.stopPropagation()}>{vertName(a.vertical, lang)}</a>
      {a.layer === "world" && a.country && <a href={"#world-" + a.country} className="text-muted hover:underline" onClick={(e) => e.stopPropagation()}> · {ccOf(a).map((c) => ccName(c, lang)).join(", ")}</a>}
    </span>
  );
}
export function Byline({ a, date, className = "" }) {
  const lang = useLang(), t = useT();
  return (
    <div className={"meta flex flex-wrap items-center gap-x-2 gap-y-1 " + className}>
      <a href={"#author-" + a.author} className="text-ink/80 hover:text-accent">{authorName(a.author, lang)}</a>
      <span aria-hidden="true">·</span>
      <span>{a.readMin} {t.minRead} {t.read}</span>
      {date && <><span aria-hidden="true">·</span><time dateTime={a.published}>{ago(a.published, lang)}</time></>}
    </div>
  );
}
export function LabelTag({ a }) {
  const t = useT();
  const map = { "Редакция": null, "Экосистема Creata": t.labelEco, "Партнёрский материал": t.labelPartner, "AI-подборка": t.labelAi };
  const txt = map[a.label];
  if (!txt) return null;
  return <span className="inline-block rounded-sm bg-soft px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wider text-accent-ink">{txt}</span>;
}
export function SaveBtn({ id, className = "" }) {
  const t = useT();
  const on = (CN.prefs.saved || []).includes(id);
  return (
    <button type="button" aria-pressed={on} aria-label={on ? t.saved : t.save} title={on ? t.saved : t.save}
      onClick={(e) => { e.preventDefault(); e.stopPropagation(); togglePref("saved", id); }}
      className={"grid h-9 w-9 place-items-center rounded-full border border-rule bg-card/90 text-ink backdrop-blur transition hover:border-accent hover:text-accent " + className}>
      {on ? <BookmarkCheck size={16} className="text-accent" /> : <Bookmark size={16} />}
    </button>
  );
}

/* ---------- Карточка материала ---------- */
export function StoryCard({ a, size = "md", dek = true, img = true, className = "", ratio }) {
  const lang = useLang();
  const title = tr(a, "title", lang), lead = tr(a, "lead", lang);
  const T = { xl: "text-[30px] sm:text-[40px] lg:text-[46px]", lg: "text-[26px] sm:text-[30px]", md: "text-[20px] sm:text-[22px]", sm: "text-[17px]" }[size];
  return (
    <article className={"group relative " + className}>
      {img && a.photo && (
        <a href={"#" + a.id} tabIndex={-1} aria-hidden="true" className="block">
          <Photo src={a.photo.url} ratio={ratio || (size === "xl" ? "aspect-[16/10]" : "aspect-[3/2]")} live={a.live && size === "xl"} eager={size === "xl"} />
        </a>
      )}
      <div className={img ? "mt-3" : ""}>
        <div className="flex items-center gap-2"><Kicker a={a} /><LabelTag a={a} /></div>
        <h3 className={"h-title mt-1.5 " + T}>
          <a href={"#" + a.id} className="hover-u focus-visible:underline">{title}</a>
        </h3>
        {dek && lead && <p className={"mt-2 font-serif text-muted " + (size === "xl" ? "text-[19px] leading-snug" : "text-[16px] leading-snug")}>{lead}</p>}
        <Byline a={a} date className="mt-3" />
      </div>
    </article>
  );
}

/* RoW-заголовок секции: «Название >» на линейке */
export function SectionHead({ title, href, lead, className = "", invert, compact }) {
  const t = useT();
  return (
    <div className={"mb-6 " + className}>
      <div className={"flex items-end justify-between gap-4 border-t-2 pt-3 " + (invert ? "border-current" : "border-ink")}>
        <h2 className={"h-title " + (compact ? "text-[24px]" : "text-[26px] sm:text-[32px]")}>
          {href ? <a href={href} className="group inline-flex items-center gap-1 hover:text-accent">{title}<ChevronRight className="transition-transform group-hover:translate-x-1" size={26} strokeWidth={2.4} /></a> : title}
        </h2>
        {href && !compact && <a href={href} className="meta hidden hover:text-accent sm:block">{t.seeAll}</a>}
      </div>
      {lead && <p className={"mt-2 max-w-2xl font-serif text-[17px] " + (invert ? "opacity-80" : "text-muted")}>{lead}</p>}
    </div>
  );
}

/* ---------- Ссылка навигации: Skiper UI skiper58 «TextRoll» ---------- */
export function RollLink({ href, children, className = "", active }) {
  return (
    <a href={href} aria-current={active ? "page" : undefined} aria-label={children} className={"relative inline-block shrink-0 whitespace-nowrap py-1 " + className}>
      <TextRoll lineHeight={1.25} className="[&>div:last-child]:text-accent">{children}</TextRoll>
      {active && <motion.span layoutId="nav-underline" className="absolute inset-x-0 -bottom-[3px] h-[2px] bg-accent" />}
    </a>
  );
}

/* ---------- Чипы-фильтры с плавающей подложкой ---------- */
export function Chips({ items, value, onChange, id = "chips" }) {
  return (
    <div role="tablist" className="no-scrollbar -mx-4 flex gap-2 overflow-x-auto px-4 pb-1 sm:mx-0 sm:flex-wrap sm:px-0">
      {items.map((it) => {
        const on = it.id === value;
        return (
          <button key={it.id} role="tab" aria-selected={on} onClick={() => onChange(it.id)}
            className={"relative h-9 shrink-0 rounded-full border px-4 text-sm font-semibold transition-colors " + (on ? "border-transparent text-onaccent" : "border-rule text-ink hover:border-ink")}>
            {on && <motion.span layoutId={id + "-pill"} className="absolute inset-0 rounded-full bg-accent" transition={{ type: "spring", stiffness: 420, damping: 34 }} />}
            <span className="relative">{it.label}{it.count != null && <span className="ml-1.5 font-mono text-[11px] opacity-70">{it.count}</span>}</span>
          </button>
        );
      })}
    </div>
  );
}

/* ---------- Обратный отсчёт ---------- */
export function Countdown({ iso, compact }) {
  const t = useT();
  useNow(1000);
  const ms = Math.max(0, new Date(iso) - Date.now());
  const parts = [[Math.floor(ms / 864e5), t.days], [Math.floor(ms / 36e5) % 24, t.hours], [Math.floor(ms / 6e4) % 60, t.mins], [Math.floor(ms / 1e3) % 60, t.secs]];
  if (compact) return <span className="font-mono tabular-nums">{parts.slice(0, 3).map(([n, u]) => String(n).padStart(2, "0") + u).join(" ")}</span>;
  return (
    <div className="flex gap-1.5 font-mono tabular-nums" role="timer">
      {parts.map(([n, u], i) => (
        <span key={i} className="flex min-w-[44px] flex-col items-center rounded-md bg-ink/[.06] px-1.5 py-1 dark:bg-white/10">
          <FlowDigit value={n} className="text-[18px] font-semibold leading-none" />
          <span className="mt-0.5 text-[10px] uppercase opacity-60">{u}</span>
        </span>
      ))}
    </div>
  );
}

/* ---------- След из фото за курсором (skiper «Image cursor trail») ---------- */
export function ImageTrail({ images, children, className = "" }) {
  const [items, setItems] = useState([]);
  const last = useRef({ x: 0, y: 0, i: 0, t: 0 });
  const ref = useRef(null);
  function move(e) {
    if (e.pointerType === "touch") return;
    const r = ref.current.getBoundingClientRect();
    const x = e.clientX - r.left, y = e.clientY - r.top;
    const L = last.current;
    if (Math.hypot(x - L.x, y - L.y) < 90) return;
    L.x = x; L.y = y; L.i = (L.i + 1) % images.length;
    const id = ++L.t;
    setItems((s) => [...s.slice(-7), { id, x, y, src: images[L.i], r: (Math.random() - 0.5) * 16 }]);
    setTimeout(() => setItems((s) => s.filter((k) => k.id !== id)), 1100);
  }
  return (
    <div ref={ref} onPointerMove={move} className={"relative overflow-hidden " + className}>
      <div className="pointer-events-none absolute inset-0 z-0" aria-hidden="true">
        <AnimatePresence>
          {items.map((it) => (
            <motion.img key={it.id} src={it.src} alt=""
              initial={{ opacity: 0, scale: 0.6, rotate: it.r }} animate={{ opacity: 1, scale: 1, rotate: it.r }} exit={{ opacity: 0, scale: 0.85, y: 24 }}
              transition={{ duration: 0.45, ease }}
              className="absolute h-[150px] w-[120px] rounded-md object-cover shadow-xl sm:h-[190px] sm:w-[150px]"
              style={{ left: it.x - 75, top: it.y - 95 }} />
          ))}
        </AnimatePresence>
      </div>
      <div className="relative z-10">{children}</div>
    </div>
  );
}

/* ---------- Лента, которую тянут мышью (skiper «Things drag and scroll») ---------- */
export function DragStrip({ children, className = "" }) {
  const outer = useRef(null), inner = useRef(null);
  const [w, setW] = useState(0);
  useEffect(() => {
    const f = () => setW(Math.max(0, inner.current.scrollWidth - outer.current.offsetWidth));
    f(); window.addEventListener("resize", f); const t = setTimeout(f, 600);
    return () => { window.removeEventListener("resize", f); clearTimeout(t); };
  }, [children]);
  return (
    <div ref={outer} className={"overflow-hidden " + className}>
      <motion.div ref={inner} drag="x" dragConstraints={{ left: -w, right: 0 }} dragElastic={0.08} className="flex w-max cursor-grab gap-4 active:cursor-grabbing">
        {children}
      </motion.div>
    </div>
  );
}

/* ---------- Стопка аватаров, раскрывающаяся при наведении (skiper «Hover members») ---------- */
export function HoverMembers({ people, max = 6 }) {
  const lang = useLang();
  const [hover, setHover] = useState(null);
  return (
    <div className="flex items-center">
      {people.slice(0, max).map((p, i) => (
        <a key={p.id} href={"#author-" + p.id} onMouseEnter={() => setHover(p.id)} onMouseLeave={() => setHover(null)} onFocus={() => setHover(p.id)} onBlur={() => setHover(null)}
          className="relative -ml-3 first:ml-0" style={{ zIndex: hover === p.id ? 20 : 10 - i }}>
          <motion.span animate={{ y: hover === p.id ? -6 : 0, scale: hover === p.id ? 1.12 : 1 }} className="block">
            <Avatar a={p} size={44} ring />
          </motion.span>
          <AnimatePresence>
            {hover === p.id && (
              <motion.span initial={{ opacity: 0, y: 6, scale: 0.9 }} animate={{ opacity: 1, y: 0, scale: 1 }} exit={{ opacity: 0, y: 6 }}
                className="absolute bottom-full left-1/2 mb-2 -translate-x-1/2 whitespace-nowrap rounded-md bg-ink px-2 py-1 text-xs font-semibold text-paper">
                {tr(p, "name", lang)}<span className="block text-[10px] font-normal opacity-70">{tr(p, "role", lang)}</span>
              </motion.span>
            )}
          </AnimatePresence>
        </a>
      ))}
    </div>
  );
}
export function Avatar({ a, size = 40, ring }) {
  const lang = useLang();
  const name = tr(a, "name", lang) || "?";
  return a.avatar ? (
    <img src={a.avatar} alt="" className={"rounded-full object-cover " + (ring ? "ring-2 ring-paper" : "")} style={{ width: size, height: size }} />
  ) : (
    <span className={"grid place-items-center rounded-full bg-accent font-logo font-bold text-onaccent " + (ring ? "ring-2 ring-paper" : "")} style={{ width: size, height: size, fontSize: size * 0.4 }}>{name[0]}</span>
  );
}

/* ---------- Магнитная кнопка ---------- */
export function Magnetic({ children, className = "" }) {
  const x = useMotionValue(0), y = useMotionValue(0);
  const sx = useSpring(x, { stiffness: 300, damping: 20 }), sy = useSpring(y, { stiffness: 300, damping: 20 });
  return (
    <motion.span className={"inline-block " + className} style={{ x: sx, y: sy }}
      onPointerMove={(e) => { const r = e.currentTarget.getBoundingClientRect(); x.set((e.clientX - r.left - r.width / 2) * 0.25); y.set((e.clientY - r.top - r.height / 2) * 0.35); }}
      onPointerLeave={() => { x.set(0); y.set(0); }}>
      {children}
    </motion.span>
  );
}

/* Появление блока при прокрутке */
export function Rise({ children, delay = 0, className = "", as = "div" }) {
  const C = motion[as];
  return (
    <C className={className} initial={{ opacity: 0, y: 24 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true, margin: "0px 0px -10% 0px" }} transition={{ duration: 0.6, delay, ease }}>
      {children}
    </C>
  );
}

export { motion, AnimatePresence, useTransform };
