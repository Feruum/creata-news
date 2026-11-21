import React, { useEffect, useMemo, useRef, useState } from "react";
import Globe from "react-globe.gl";
import { motion, AnimatePresence } from "motion/react";
import * as THREE from "three";
import * as topo from "topojson-client";
import atlas from "world-atlas/countries-110m.json";
import { geoCentroid } from "d3-geo";
import { numToCC, ccName, flag } from "./geo.js";
import { tr, vertName } from "./lib.js";

const FEATURES = topo.feature(atlas, atlas.objects.countries).features
  .filter((f) => f.id !== "010")
  .map((f) => ({ ...f, cc: numToCC(f.id), c: geoCentroid(f) }));
/* центры для стран, у которых центроид смещён */
const FIX = { US: [-98, 39], RU: [90, 60], FR: [2.3, 46.5], NO: [9, 61], CA: [-100, 58] };
export const centerOf = (cc) => { if (FIX[cc]) return FIX[cc]; const f = FEATURES.find((x) => x.cc === cc); return f ? f.c : [0, 20]; };

/* где вокруг страны «парят» карточки новостей: смещение по широте/долготе */

/* тема сайта: следим за data-theme и системной настройкой */
function useDark() {
  const get = () => { const t = document.documentElement.getAttribute("data-theme"); return t ? t === "dark" : matchMedia("(prefers-color-scheme: dark)").matches; };
  const [d, setD] = useState(get);
  useEffect(() => {
    const mo = new MutationObserver(() => setD(get())); mo.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    const mq = matchMedia("(prefers-color-scheme: dark)"); const f = () => setD(get()); mq.addEventListener("change", f);
    return () => { mo.disconnect(); mq.removeEventListener("change", f); };
  }, []);
  return d;
}
const PAL = {
  dark: { sea: "#14132b", glow: "#0d0c22", land: "rgba(120,118,150,0.55)", news: "#7d77ff", stroke: "rgba(10,10,20,.6)", side: "rgba(75,68,240,.25)", atmo: "#6b65ff", ring: "160,155,255", line: "rgba(168,163,255,.55)" },
  light: { sea: "#c8c4ef", glow: "#b9b4ec", land: "#a19eba", news: "#8c86ff", stroke: "rgba(255,255,255,.9)", side: "rgba(75,68,240,.18)", atmo: "#9d98ff", ring: "75,68,240", line: "rgba(75,68,240,.45)" },
};

/** 3D-глобус: крутится, флажки стран с новостями зовут внимание,
 *  при наведении вокруг страны всплывают её новости, клик фильтрует список. */
export function StoryGlobe({ groups, selected, onSelect, lang, height = 520 }) {
  const ref = useRef(), box = useRef();
  const [w, setW] = useState(0);
  const [hover, setHover] = useState(null); // cc под курсором (или карточкой)
  const [tour, setTour] = useState(null); // cc, который глобус показывает сам
  const [idle, setIdle] = useState(true);
  const leave = useRef(), idleT = useRef(), first = useRef(true);
  const counts = useMemo(() => Object.fromEntries(groups.map(([c, l]) => [c, l.length])), [groups]);
  const focus = selected || hover || tour;
  const selRef = useRef(selected); selRef.current = selected;

  useEffect(() => {
    const ro = new ResizeObserver(([e]) => setW(Math.round(e.contentRect.width)));
    ro.observe(box.current); return () => ro.disconnect();
  }, []);
  const dark = useDark(), P = dark ? PAL.dark : PAL.light;
  const mat = useMemo(() => new THREE.MeshPhongMaterial({ color: P.sea, emissive: P.glow, emissiveIntensity: dark ? 1 : 0.35, shininess: dark ? 8 : 4 }), [dark]);

  /* наведение: держим карточки, пока курсор переходит со страны на карточку */
  const hoverOn = (cc) => { clearTimeout(leave.current); if (cc && counts[cc]) setHover(cc); };
  const hoverOff = () => { clearTimeout(leave.current); leave.current = setTimeout(() => setHover(null), 900); };
  const touch = () => { setIdle(false); setTour(null); clearTimeout(idleT.current); idleT.current = setTimeout(() => setIdle(true), 9000); };

  /* автотур: пока никто не трогает глобус, он сам по очереди показывает страны,
     начиная с тех, где новостей меньше всего, чтобы маленькие не терялись */
  useEffect(() => {
    if (!idle || selected || hover || !groups.length) { if (!idle || selected) setTour(null); return; }
    const order = [...groups].sort((a, b) => a[1].length - b[1].length).map(([c]) => c);
    let i = tour ? (order.indexOf(tour) + 1) % order.length : 0;
    setTour(order[i]);
    const t = setInterval(() => { i = (i + 1) % order.length; setTour(order[i]); }, 5200);
    return () => clearInterval(t);
  }, [idle, selected, hover, groups.length]);

  useEffect(() => {
    const g = ref.current; if (!g) return;
    const c = g.controls(); c.autoRotate = !focus; c.autoRotateSpeed = 0.4; c.minDistance = 180; c.maxDistance = 520;
    if (hover) { g.pointOfView(g.pointOfView(), 0); return; } // при наведении камера замирает
    const cc = selected || tour || (first.current ? "KZ" : null); first.current = false;
    if (!cc) return;
    const [lng, lat] = centerOf(cc);
    g.pointOfView({ lat: lat - 4, lng, altitude: selected ? 1.7 : 2.05 }, 1400);
  }, [selected, tour, w, hover]);

  useEffect(() => {
    const g = ref.current; if (!g) return;
    const c = g.controls(); const f = () => touch(); c.addEventListener("start", f);
    return () => c.removeEventListener("start", f);
  }, [w]);

  /* HTML-слои: флажки всех стран с новостями + карточки вокруг страны в фокусе */
  const pins = useMemo(() => groups.map(([cc, l]) => { const [lng, lat] = centerOf(cc); return { kind: "pin", key: "pin-" + cc, cc, lat, lng, alt: 0.02, n: l.length }; }), [groups]);
  const cards = useMemo(() => focus ? (groups.find(([c]) => c === focus) || [, []])[1].slice(0, 4) : [], [focus, groups]);
  const L = useRef({}); L.current = { lang, onSelect, hoverOn, hoverOff, touch };

  const make = useMemo(() => (d) => {
    const el = document.createElement("div");
    el.className = "gpin"; el.dataset.cc = d.cc;
    el.innerHTML = `<span class="gflag">${flag(d.cc)}</span><span class="gnum">${d.n}</span>`;
    el.title = ccName(d.cc, L.current.lang);
    return el;
  }, []);

  /* карточки новостей «парят» рядом со страной: экранные координаты точки обновляются каждый кадр */
  const anchor = useRef(), [side, setSide] = useState(1), sideRef = useRef(1);
  useEffect(() => {
    if (!focus) return;
    const [lng, lat] = centerOf(focus);
    let raf, first = true;
    const tick = () => {
      const g = ref.current, el = anchor.current;
      if (g && el) {
        const p = g.getScreenCoords(lat, lng, 0.03);
        if (p && isFinite(p.x)) {
          el.style.transform = `translate(${p.x}px, ${p.y}px)`;
          const want = p.x > w * 0.58 ? -1 : p.x < w * 0.42 ? 1 : sideRef.current;
          if (want !== sideRef.current || first) { sideRef.current = want; setSide(want); first = false; }
        }
      }
      raf = requestAnimationFrame(tick);
    };
    tick(); return () => cancelAnimationFrame(raf);
  }, [focus, w]);
  const mobile = w < 640;
  const CW = mobile ? 170 : 230;
  const slot = (i, n) => {
    const step = mobile ? 62 : 74, y = (i - (n - 1) / 2) * step - (mobile ? 0 : 10);
    const x = (mobile ? 34 : 60) + Math.abs(i - (n - 1) / 2) * (mobile ? 6 : 22);
    return { x: side > 0 ? x : -x - CW, y: y - 27 };
  };

  useEffect(() => { box.current?.querySelectorAll(".gpin").forEach((e) => e.classList.toggle("on", e.dataset.cc === focus)); }, [focus, pins]);
  const cap = (f) => f.cc === focus ? "#4B44F0" : counts[f.cc] ? P.news : P.land;
  return (
    <div ref={box} className="relative w-full select-none" style={{ height }} onPointerDown={touch}
      onPointerOver={(e) => { const pin = e.target.closest?.(".gpin"); if (pin) { touch(); hoverOn(pin.dataset.cc); } }}
      onPointerOut={(e) => { const pin = e.target.closest?.(".gpin"); if (pin && !pin.contains(e.relatedTarget)) hoverOff(); }}
      onClick={(e) => { const pin = e.target.closest?.(".gpin"); if (pin) onSelect(pin.dataset.cc === selected ? null : pin.dataset.cc); }}>
      {w > 0 && <Globe ref={ref} width={w} height={height} backgroundColor="rgba(0,0,0,0)"
        globeMaterial={mat} showAtmosphere atmosphereColor={P.atmo} atmosphereAltitude={0.18} showGraticules
        polygonsData={FEATURES} polygonAltitude={(f) => (f.cc === focus ? 0.035 : counts[f.cc] ? 0.012 : 0.006)}
        polygonCapColor={cap} polygonSideColor={() => P.side} polygonStrokeColor={() => P.stroke}
        polygonsTransitionDuration={300} polygonLabel={() => ""}
        onPolygonHover={(f) => { if (f && counts[f.cc]) { touch(); hoverOn(f.cc); } else hoverOff(); }}
        onPolygonClick={(f) => counts[f.cc] && onSelect(f.cc === selected ? null : f.cc)}
        htmlElementsData={pins} htmlLat="lat" htmlLng="lng" htmlAltitude="alt" htmlElement={make} htmlTransitionDuration={0}
        ringsData={pins} ringColor={() => (t) => `rgba(${P.ring},${1 - t})`} ringMaxRadius={(d) => 3 + 6 / d.n} ringPropagationSpeed={1.4} ringRepeatPeriod={(d) => 900 + d.n * 300} ringAltitude={0.015}
      />}
      <div className="pointer-events-none absolute inset-0 z-10 overflow-hidden">
        <div ref={anchor} className="absolute left-0 top-0" style={{ willChange: "transform" }}>
          <AnimatePresence>
            {focus && <motion.span key={"dot" + focus} initial={{ scale: 0 }} animate={{ scale: 1 }} exit={{ scale: 0 }} className="absolute -left-[7px] -top-[7px] h-3.5 w-3.5 rounded-full border-2 border-white bg-[#4B44F0] shadow-[0_0_0_6px_rgba(75,68,240,.35)]" />}
            {cards.map((a, i) => {
              const s2 = slot(i, cards.length);
              return (
                <motion.div key={focus + a.id} className="gcard absolute" style={{ left: s2.x, top: s2.y, width: CW }}
                  initial={{ opacity: 0, x: -side * 30, scale: 0.85 }} animate={{ opacity: 1, x: 0, scale: 1 }} exit={{ opacity: 0, scale: 0.9, transition: { duration: 0.15 } }}
                  transition={{ delay: i * 0.08, type: "spring", stiffness: 260, damping: 24 }}
                  onMouseEnter={() => { clearTimeout(leave.current); touch(); }} onMouseLeave={hoverOff}>
                  <svg className="pointer-events-none absolute overflow-visible" style={{ left: 0, top: 0 }} width="1" height="1" aria-hidden="true">
                    <line x1={side > 0 ? 0 : CW} y1={27} x2={-s2.x} y2={-s2.y} stroke={P.line} strokeWidth="1" strokeDasharray="3 3" />
                  </svg>
                  <a href={"#" + a.id}><img src={a.photo.url} alt="" /><span><b>{vertName(a.vertical, lang)}</b>{tr(a, "title", lang)}</span></a>
                </motion.div>
              );
            })}
          </AnimatePresence>
        </div>
      </div>
      <div className="pointer-events-none absolute inset-x-0 bottom-2 text-center">
        {focus && <p className="font-head text-[15px] font-bold text-ink">{flag(focus)} {ccName(focus, lang)} · {counts[focus]} {lang === "en" ? (counts[focus] === 1 ? "story" : "stories") : counts[focus] === 1 ? "новость" : counts[focus] < 5 ? "новости" : "новостей"}{tour && !hover && !selected ? (lang === "en" ? " · in focus" : " · в фокусе") : ""}</p>}
        <p className="mt-1 font-mono text-[11px] uppercase tracking-wider text-muted">{lang === "en" ? "Drag to rotate · hover a flag · click to filter" : "Крутите глобус · наведите на флажок · клик — фильтр"}</p>
      </div>
    </div>
  );
}
