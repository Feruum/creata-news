import React, { useEffect, useState } from "react";
import { motion, AnimatePresence, useScroll, useMotionValueEvent } from "motion/react";
import { Menu, Search, X, Moon, Sun, Send, ArrowRight, ArrowUp, Check } from "lucide-react";
import { CN, useLang, tr, setPref, published, vertName, fmtDate, bump } from "./lib.js";
import { useT } from "./i18n.js";
import { Logo, RollLink, ease, Magnetic } from "./ui.jsx";
import { TextRoll } from "./skiper/skiper58.jsx";

export const NAV = (t) => [
  ["world", t.world], ["shorts", t.shorts], ["ecosystem", t.eco], ["opportunities", t.opps], ["issue", t.issue], ["morning", t.morning], ["about", t.about]
];

function LangSwitch({ dark }) {
  const lang = useLang(), t = useT();
  return (
    <div className="flex items-center gap-1 font-mono text-[11px] uppercase" role="group" aria-label="Language / Язык">
      {["ru", "en"].map((l) => (
        <button key={l} onClick={() => setPref("lang", l)} aria-pressed={lang === l}
          className={"rounded px-1.5 py-0.5 transition " + (lang === l ? (dark ? "bg-white text-accent" : "bg-ink text-paper") : "opacity-80 hover:opacity-100")}>{l}</button>
      ))}
      <span className="hidden px-1.5 py-0.5 opacity-50 sm:inline" title={t.langSoon}>es · {t.langSoon}</span>
    </div>
  );
}
function ThemeBtn() {
  const t = useT();
  const cur = CN.prefs.theme === "auto" ? (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light") : CN.prefs.theme;
  const next = cur === "dark" ? "light" : "dark";
  return (
    <button onClick={() => setPref("theme", next)} aria-label={t.theme + ": " + (next === "dark" ? t.dark : t.light)} className="grid h-7 w-7 place-items-center rounded-full hover:bg-white/15">
      {cur === "dark" ? <Sun size={14} /> : <Moon size={14} />}
    </button>
  );
}

export function Header({ route, onMenu, onSearch, onSubscribe }) {
  const t = useT(), lang = useLang();
  const { scrollY } = useScroll();
  const [compact, setCompact] = useState(false);
  useMotionValueEvent(scrollY, "change", (v) => setCompact(v > 220));
  const today = fmtDate(new Date().toISOString(), lang, { weekday: "long", day: "numeric", month: "long", year: "numeric" });
  const nav = NAV(t);
  const active = (id) => route === id || (id === "world" && route.startsWith("world-")) || (id === "shorts" && route.startsWith("shorts")) || (id === "world" && route.startsWith("v-") && CN.vert(route.slice(2)).layer === "world") || (id === "ecosystem" && route.startsWith("v-eco"));
  return (
    <>
      <div className="bg-[#4B44F0] text-white">
        <div className="wrap flex h-9 items-center justify-between gap-3 text-[13px]">
          <span className="hidden font-mono text-[11px] uppercase tracking-wider sm:block">{today}</span>
          <button onClick={onSubscribe} className="truncate font-semibold hover:underline">{t.morning}: {lang === "en" ? "3 minutes at 08:00" : "3 минуты в 08:00"} →</button>
          <div className="flex items-center gap-2"><LangSwitch dark /><ThemeBtn /></div>
        </div>
      </div>
      <header className="bg-paper">
        <div className="wrap grid grid-cols-[1fr_auto_1fr] items-center py-4 sm:py-6">
          <div className="flex items-center gap-1">
            <button onClick={onMenu} aria-label={t.menu} className="grid h-10 w-10 place-items-center rounded-full hover:bg-band"><Menu size={22} /></button>
            <button onClick={onSearch} aria-label={t.search} className="grid h-10 w-10 place-items-center rounded-full hover:bg-band"><Search size={20} /></button>
          </div>
          <a href="#" aria-label="Creata News" className="text-center"><Logo className="text-[22px] sm:text-[44px] lg:text-[54px]" /></a>
          <div className="flex justify-end gap-2">
            <a href="admin.html" className="btn hidden text-ink/80 hover:text-accent md:inline-flex">{t.signIn}</a>
            <Magnetic className="hidden sm:inline-block"><button onClick={onSubscribe} className="btn-accent h-10 px-4 text-sm">{t.subscribe}</button></Magnetic>
          </div>
        </div>
        <nav aria-label={t.sections} className="wrap">
          <div className="no-scrollbar flex justify-start gap-6 overflow-x-auto border-y border-ink py-2.5 text-[15px] font-semibold sm:justify-center sm:gap-8">
            {nav.map(([id, label]) => <RollLink key={id} href={"#" + id} active={active(id)}>{label}</RollLink>)}
          </div>
        </nav>
      </header>
      {/* компактная шапка при прокрутке */}
      <AnimatePresence>
        {compact && (
          <motion.div initial={{ y: -70 }} animate={{ y: 0 }} exit={{ y: -70 }} transition={{ duration: 0.35, ease }}
            className="fixed inset-x-0 top-0 z-40 border-b border-rule bg-paper/90 backdrop-blur-md">
            <div className="wrap flex h-14 items-center gap-4">
              <button onClick={onMenu} aria-label={t.menu} className="grid h-9 w-9 place-items-center rounded-full hover:bg-band"><Menu size={20} /></button>
              <a href="#" aria-label="Creata News"><Logo className="text-[18px]" /></a>
              <nav className="no-scrollbar hidden flex-1 gap-6 overflow-x-auto text-sm font-semibold lg:flex">
                {nav.map(([id, label]) => <RollLink key={id} href={"#" + id}>{label}</RollLink>)}
              </nav>
              <div className="ml-auto flex items-center gap-1">
                <button onClick={onSearch} aria-label={t.search} className="grid h-9 w-9 place-items-center rounded-full hover:bg-band"><Search size={18} /></button>
                <button onClick={onSubscribe} className="btn-accent h-9 px-3 text-[13px]">{t.subscribe}</button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}

/* ---------- Dynamic island: свежая новость (skiper «Dynamic island») ---------- */
export function DynamicIsland({ route }) {
  const lang = useLang(), t = useT();
  const { scrollY } = useScroll();
  const [show, setShow] = useState(false);
  const [open, setOpen] = useState(false);
  useMotionValueEvent(scrollY, "change", (v) => setShow(v > 900));
  const a = published(lang)[0];
  if (!a || route === a.id) return null;
  return (
    <AnimatePresence>
      {show && (
        <motion.div key="island" initial={{ y: 80, opacity: 0 }} animate={{ y: 0, opacity: 1 }} exit={{ y: 80, opacity: 0 }} className="pointer-events-none fixed inset-x-0 bottom-4 z-40 flex justify-center px-4">
        <motion.a href={"#" + a.id} layout
          onMouseEnter={() => setOpen(true)} onMouseLeave={() => setOpen(false)} onFocus={() => setOpen(true)} onBlur={() => setOpen(false)}
          transition={{ type: "spring", stiffness: 380, damping: 32 }}
          style={{ borderRadius: 28 }}
          className="pointer-events-auto flex max-w-full items-center gap-3 overflow-hidden bg-[#0E0E10] p-1.5 pr-4 text-white shadow-2xl">
          <motion.img layout src={a.photo.url} alt="" style={{ borderRadius: 22 }} className={open ? "h-16 w-16 object-cover" : "h-9 w-9 object-cover"} />
          <motion.div layout className="min-w-0">
            <span className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-wider text-[#8C87FF]">
              <span className="relative flex h-2 w-2"><span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-[#8C87FF] opacity-70" /><span className="relative inline-flex h-2 w-2 rounded-full bg-[#8C87FF]" /></span>
              {t.breaking}
            </span>
            <span className={"block text-[13px] font-semibold leading-tight " + (open ? "max-w-[420px]" : "max-w-[240px] truncate sm:max-w-[320px]")}>{tr(a, "title", lang)}</span>
          </motion.div>
          <ArrowRight size={16} className="shrink-0 opacity-70" />
        </motion.a>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/* ---------- Меню во весь экран ---------- */
export function MenuOverlay({ open, onClose }) {
  const t = useT(), lang = useLang();
  const world = CN.state.verticals.filter((v) => v.layer === "world");
  const eco = CN.state.verticals.filter((v) => v.layer === "eco");
  useEffect(() => {
    if (!open) return;
    const f = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", f); window.addEventListener("hashchange", onClose);
    return () => { window.removeEventListener("keydown", f); window.removeEventListener("hashchange", onClose); };
  }, [open]);
  return (
    <AnimatePresence>
      {open && (
        <motion.div role="dialog" aria-modal="true" aria-label={t.menu} initial={{ clipPath: "inset(0 0 100% 0)" }} animate={{ clipPath: "inset(0 0 0% 0)" }} exit={{ clipPath: "inset(0 0 100% 0)" }}
          transition={{ duration: 0.55, ease }} className="fixed inset-0 z-50 overflow-y-auto bg-[#4B44F0] text-white">
          <div className="wrap flex h-16 items-center justify-between">
            <Logo className="text-[22px]" />
            <button onClick={onClose} autoFocus aria-label={t.close} className="grid h-11 w-11 place-items-center rounded-full border border-white/40 hover:bg-white hover:text-accent"><X /></button>
          </div>
          <div className="wrap grid gap-10 pb-16 pt-6 lg:grid-cols-[1.2fr_1fr_1fr]">
            <nav className="flex flex-col gap-1">
              {NAV(t).map(([id, label], i) => (
                <motion.a key={id} href={"#" + id} initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.15 + i * 0.05, ease }}
                  className="h-title block py-1 text-[40px] sm:text-[56px]" aria-label={label}><TextRoll lineHeight={1.08} className="[&>div:last-child]:text-[#0E0E10]">{label}</TextRoll></motion.a>
              ))}
            </nav>
            <div>
              <p className="mb-3 font-mono text-[11px] uppercase tracking-wider opacity-70">{t.world}</p>
              <ul className="grid grid-cols-2 gap-x-4 gap-y-2 text-[15px] lg:grid-cols-1">{world.map((v) => <li key={v.id}><a className="hover:underline" href={"#v-" + v.id}>{tr(v, "name", lang)}</a></li>)}</ul>
            </div>
            <div>
              <p className="mb-3 font-mono text-[11px] uppercase tracking-wider opacity-70">{t.eco}</p>
              <ul className="grid grid-cols-2 gap-x-4 gap-y-2 text-[15px] lg:grid-cols-1">{eco.map((v) => <li key={v.id}><a className="hover:underline" href={"#v-" + v.id}>{tr(v, "name", lang)}</a></li>)}</ul>
              <div className="mt-8 flex flex-col gap-2 text-[15px]">
                <a href="#authors" className="hover:underline">{t.authors}</a>
                <a href="#saved" className="hover:underline">{t.saved}</a>
                <a href="admin.html" className="hover:underline">{t.admin} →</a>
              </div>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/* ---------- Поиск-оверлей ---------- */
export function SearchOverlay({ open, onClose }) {
  const t = useT(), lang = useLang();
  const [q, setQ] = useState("");
  useEffect(() => { if (!open) return; const f = (e) => e.key === "Escape" && onClose(); window.addEventListener("keydown", f); window.addEventListener("hashchange", onClose); return () => { window.removeEventListener("keydown", f); window.removeEventListener("hashchange", onClose); }; }, [open]);
  const res = q.trim().length < 2 ? [] : published(lang).filter((a) => {
    const s = [tr(a, "title", lang), tr(a, "lead", lang), (tr(a, "tags", lang) || []).join(" "), vertName(a.vertical, lang), tr(CN.author(a.author), "name", lang)].join(" ").toLowerCase();
    return q.toLowerCase().split(/\s+/).every((w) => s.includes(w));
  });
  return (
    <AnimatePresence>
      {open && (
        <motion.div role="dialog" aria-modal="true" aria-label={t.search} initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="fixed inset-0 z-50 bg-paper/95 backdrop-blur-md">
          <div className="wrap max-w-3xl pt-6">
            <div className="flex items-center gap-3 border-b-2 border-ink pb-3">
              <Search size={26} />
              <input autoFocus value={q} onChange={(e) => setQ(e.target.value)} placeholder={t.searchPh} aria-label={t.search}
                className="h-title min-w-0 flex-1 bg-transparent text-[22px] outline-none placeholder:text-muted sm:text-[34px]" />
              <button onClick={onClose} aria-label={t.close} className="grid h-10 w-10 shrink-0 place-items-center rounded-full hover:bg-band"><X /></button>
            </div>
            <div className="mt-6 max-h-[70vh] overflow-y-auto">
              {q.trim().length >= 2 && !res.length && <p className="font-serif text-lg text-muted">{t.nothing}</p>}
              {res.map((a, i) => (
                <motion.a key={a.id} href={"#" + a.id} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.03 }}
                  className="group flex gap-4 border-b border-rule py-4">
                  <img src={a.photo.url} alt="" className="h-16 w-24 shrink-0 object-cover" />
                  <span><span className="kicker">{vertName(a.vertical, lang)}</span><span className="h-title mt-1 block text-[18px] group-hover:text-accent">{tr(a, "title", lang)}</span></span>
                </motion.a>
              ))}
              {q.trim().length < 2 && (
                <div className="flex flex-wrap gap-2">
                  {(lang === "en" ? ["gamedev", "Almaty", "AI", "education", "contest"] : ["геймдев", "Алматы", "AI", "образование", "конкурс"]).map((w) => (
                    <button key={w} onClick={() => setQ(w)} className="rounded-full border border-rule px-3 py-1.5 text-sm hover:border-accent hover:text-accent">{w}</button>
                  ))}
                </div>
              )}
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/* ---------- Подписка «Утро Creata»: три шага (PRD онбординг) ---------- */
export function Onboarding({ open, onClose }) {
  const t = useT(), lang = useLang();
  const [step, setStep] = useState(0);
  const [d, setD] = useState({ channel: "telegram", email: "", topics: [], stage: "exploring" });
  useEffect(() => { if (open) setStep(0); }, [open]);
  useEffect(() => { if (!open) return; const f = (e) => e.key === "Escape" && onClose(); window.addEventListener("keydown", f); return () => window.removeEventListener("keydown", f); }, [open]);
  const verts = CN.state.verticals;
  const finish = () => { setPref("digest", d); setPref("follows", d.topics); setStep(3); };
  const tog = (id) => setD((s) => ({ ...s, topics: s.topics.includes(id) ? s.topics.filter((x) => x !== id) : s.topics.length < 5 ? [...s.topics, id] : s.topics }));
  const okEmail = d.channel === "telegram" || /.+@.+\..+/.test(d.email);
  return (
    <AnimatePresence>
      {open && (
        <motion.div className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} onClick={onClose}>
          <motion.div role="dialog" aria-modal="true" aria-label={t.onbTitle} onClick={(e) => e.stopPropagation()}
            initial={{ y: 40, scale: 0.97 }} animate={{ y: 0, scale: 1 }} exit={{ y: 40, opacity: 0 }} transition={{ type: "spring", stiffness: 300, damping: 30 }}
            className="relative grid max-h-[92vh] w-full max-w-3xl overflow-hidden rounded-xl bg-card text-ink shadow-2xl md:grid-cols-[0.9fr_1.1fr]">
            <div className="relative hidden md:block">
              <img src="assets/photos/c47.jpg" alt="" className="kb absolute inset-0 h-full w-full object-cover" />
              <div className="absolute inset-0 bg-gradient-to-t from-black/70 to-transparent" />
              <div className="absolute bottom-5 left-5 right-5 text-white"><Logo className="text-[20px]" /><p className="mt-2 font-serif text-[15px] opacity-90">{t.newsletterLead}</p></div>
            </div>
            <div className="overflow-y-auto p-6 sm:p-8">
              <button onClick={onClose} aria-label={t.close} className="absolute right-3 top-3 grid h-9 w-9 place-items-center rounded-full hover:bg-band"><X size={18} /></button>
              <p className="kicker">{t.onbTitle}{step < 3 && ` · ${step + 1}/3`}</p>
              <div className="mt-2 h-1 overflow-hidden rounded bg-band"><motion.div className="h-full bg-accent" animate={{ width: `${((step + 1) / 3) * 100}%` }} /></div>
              <AnimatePresence mode="wait">
                <motion.div key={step} initial={{ opacity: 0, x: 24 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -24 }} transition={{ duration: 0.25 }} className="mt-5">
                  {step === 0 && (<>
                    <h2 className="h-title text-[26px]">{t.onbStep1}</h2>
                    <div className="mt-4 grid grid-cols-2 gap-3">
                      {[["telegram", "Telegram", <Send size={18} key="i" />], ["email", t.email, "@"]].map(([id, l, ic]) => (
                        <button key={id} onClick={() => setD({ ...d, channel: id })} aria-pressed={d.channel === id}
                          className={"flex items-center gap-2 rounded-lg border-2 p-4 text-left font-semibold " + (d.channel === id ? "border-accent bg-soft" : "border-rule hover:border-ink")}>{ic}{l}</button>
                      ))}
                    </div>
                    {d.channel === "email" && <input type="email" value={d.email} onChange={(e) => setD({ ...d, email: e.target.value })} placeholder={t.emailPh} aria-label={t.email} className="mt-3 h-11 w-full rounded-lg border border-rule bg-paper px-3 outline-none focus:border-accent" />}
                    <button disabled={!okEmail} onClick={() => setStep(1)} className="btn-accent mt-6 w-full disabled:opacity-40">{t.continue}</button>
                  </>)}
                  {step === 1 && (<>
                    <h2 className="h-title text-[26px]">{t.onbStep2}</h2>
                    <div className="mt-4 flex flex-wrap gap-2">
                      {verts.map((v) => { const on = d.topics.includes(v.id); return (
                        <button key={v.id} onClick={() => tog(v.id)} aria-pressed={on} className={"flex items-center gap-1 rounded-full border px-3 py-1.5 text-sm font-semibold " + (on ? "border-accent bg-accent text-onaccent" : "border-rule hover:border-ink")}>
                          {on && <Check size={14} />}{tr(v, "name", lang)}</button>); })}
                    </div>
                    <div className="mt-6 flex gap-2"><button onClick={() => setStep(0)} className="btn-line">{t.back}</button><button disabled={d.topics.length < 3} onClick={() => setStep(2)} className="btn-accent flex-1 disabled:opacity-40">{t.continue} · {d.topics.length}/5</button></div>
                  </>)}
                  {step === 2 && (<>
                    <h2 className="h-title text-[26px]">{t.onbStep3}</h2>
                    <div className="mt-4 grid gap-2">
                      {[["exploring", t.stageExploring], ["trying", t.stageTrying], ["building", t.stageBuilding]].map(([id, l]) => (
                        <button key={id} onClick={() => setD({ ...d, stage: id })} aria-pressed={d.stage === id} className={"rounded-lg border-2 p-3 text-left font-semibold " + (d.stage === id ? "border-accent bg-soft" : "border-rule hover:border-ink")}>{l}</button>
                      ))}
                    </div>
                    <div className="mt-6 flex gap-2"><button onClick={() => setStep(1)} className="btn-line">{t.back}</button><button onClick={finish} className="btn-accent flex-1">{t.done}</button></div>
                  </>)}
                  {step === 3 && (
                    <div className="py-6 text-center">
                      <motion.div initial={{ scale: 0 }} animate={{ scale: 1 }} transition={{ type: "spring" }} className="mx-auto grid h-14 w-14 place-items-center rounded-full bg-accent text-onaccent"><Check /></motion.div>
                      <p className="h-title mt-4 text-[22px]">{t.subscribed}</p>
                      <button onClick={onClose} className="btn-accent mt-6">{t.close}</button>
                    </div>
                  )}
                </motion.div>
              </AnimatePresence>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

export function Footer({ onSubscribe }) {
  const t = useT(), lang = useLang();
  return (
    <footer className="mt-20 bg-[#0E0E10] text-[#EDECE8]">
      <div className="wrap py-14">
        <div className="grid gap-10 lg:grid-cols-[1.4fr_1fr_1fr_1fr]">
          <div>
            <Logo className="text-[30px] sm:text-[40px]" />
            <p className="mt-4 max-w-sm font-serif text-[17px] opacity-80">{t.tagline}</p>
            <button onClick={onSubscribe} className="btn mt-6 bg-[#8C87FF] text-[#0E0E10] hover:bg-white">{t.subscribe}</button>
          </div>
          <FooterCol title={t.world} items={CN.state.verticals.filter((v) => v.layer === "world").slice(0, 7).map((v) => ["#v-" + v.id, tr(v, "name", lang)])} />
          <FooterCol title={t.eco} items={CN.state.verticals.filter((v) => v.layer === "eco").map((v) => ["#v-" + v.id, tr(v, "name", lang)])} />
          <FooterCol title="Creata News" items={[["#about", t.about], ["#authors", t.authors], ["#issue", t.issue], ["#opportunities", t.opps], ["https://t.me/creata_news", "Telegram"], ["admin.html", t.admin]]} />
        </div>
        <div className="mt-12 flex flex-col gap-3 border-t border-white/15 pt-6 font-mono text-[11px] uppercase tracking-wider opacity-60 sm:flex-row sm:justify-between">
          <span>© 2026 Creata Group · creata.news</span>
          <span>{t.footerNote} · UI: <a href="https://skiper-ui.com" target="_blank" rel="noopener" className="underline hover:opacity-100">Skiper UI</a></span>
          <button onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })} className="inline-flex items-center gap-1 hover:opacity-100">{t.backTop} <ArrowUp size={12} /></button>
        </div>
      </div>
      <div className="overflow-hidden" aria-hidden="true"><Logo className="block translate-y-[18%] whitespace-nowrap text-center text-[17vw] leading-[.8] text-white/[.06]" small /></div>
    </footer>
  );
}
function FooterCol({ title, items }) {
  return (
    <div>
      <p className="mb-3 font-mono text-[11px] uppercase tracking-wider text-[#8C87FF]">{title}</p>
      <ul className="space-y-2 text-[15px]">{items.map(([h, l]) => <li key={h + l}><a href={h} className="opacity-85 hover:text-[#8C87FF] hover:opacity-100" {...(h.startsWith("http") ? { target: "_blank", rel: "noopener" } : {})}>{l}</a></li>)}</ul>
    </div>
  );
}
