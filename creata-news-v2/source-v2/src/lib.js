import { useEffect, useState, useSyncExternalStore, createContext, useContext } from "react";

const CN = window.CN;
export { CN };

/* ---- подписка React на общее хранилище (сайт + админка) ---- */
let ver = 0;
const subs = new Set();
export function bump() { ver++; subs.forEach((f) => f()); }
export function useStore() {
  return useSyncExternalStore((f) => { subs.add(f); return () => subs.delete(f); }, () => ver);
}
window.addEventListener("storage", (e) => {
  if (e.key === "creata-news-state" && e.newValue) { try { CN.state = JSON.parse(e.newValue); bump(); } catch (_) {} }
});
export function setPref(k, v) { CN.prefs[k] = v; CN.savePrefs(); bump(); }
export function togglePref(k, id) {
  const arr = CN.prefs[k] || [];
  setPref(k, arr.includes(id) ? arr.filter((x) => x !== id) : [...arr, id]);
}

/* ---- маршрутизация по хэшу ---- */
export function useHash() {
  const [h, setH] = useState(() => location.hash.slice(1));
  useEffect(() => {
    const f = () => setH(location.hash.slice(1));
    window.addEventListener("hashchange", f);
    return () => window.removeEventListener("hashchange", f);
  }, []);
  return h;
}
export const go = (h) => { location.hash = h; };

/* ---- язык ---- */
export const LangCtx = createContext("ru");
export const useLang = () => useContext(LangCtx);
/** поле с учётом языка: obj.key_en для EN */
export function tr(obj, key, lang) {
  if (!obj) return "";
  if (lang === "en") { const v = obj[key + "_en"]; return v != null && v !== "" ? v : obj[key]; }
  return obj[key];
}
export const hasLang = (a, lang) => lang !== "en" || !!a.title_en;

export function fmtDate(iso, lang, opts) {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d)) return iso;
  return d.toLocaleDateString(lang === "en" ? "en-GB" : "ru-RU", opts || { day: "numeric", month: "long", year: "numeric" });
}
export function ago(iso, lang) {
  const m = Math.max(1, Math.round((Date.now() - new Date(iso)) / 6e4));
  const en = lang === "en";
  if (m < 60) return en ? `${m} min ago` : `${m} мин назад`;
  const h = Math.round(m / 60);
  if (h < 24) return en ? `${h} h ago` : `${h} ч назад`;
  return fmtDate(iso, lang, { day: "numeric", month: "short" });
}
export function plural(n, one, few, many) {
  const a = Math.abs(n) % 100, b = a % 10;
  if (a > 10 && a < 20) return many;
  if (b > 1 && b < 5) return few;
  if (b === 1) return one;
  return many;
}

/* ---- данные ---- */
export function published(lang) { return CN.published().filter((a) => hasLang(a, lang)); }
export const vertName = (id, lang) => tr(CN.vert(id), "name", lang);
export const authorName = (id, lang) => tr(CN.author(id), "name", lang);

export function useNow(ms = 1000) {
  const [n, setN] = useState(Date.now());
  useEffect(() => { const t = setInterval(() => setN(Date.now()), ms); return () => clearInterval(t); }, [ms]);
  return n;
}
export function useMedia(q) {
  const [m, setM] = useState(() => window.matchMedia(q).matches);
  useEffect(() => { const mq = window.matchMedia(q); const f = () => setM(mq.matches); mq.addEventListener("change", f); return () => mq.removeEventListener("change", f); }, [q]);
  return m;
}
