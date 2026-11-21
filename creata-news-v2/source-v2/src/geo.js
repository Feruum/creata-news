import countries from "i18n-iso-countries";
import ru from "i18n-iso-countries/langs/ru.json";
import en from "i18n-iso-countries/langs/en.json";
import { CN } from "./lib.js";
countries.registerLocale(ru); countries.registerLocale(en);
const SHORT = { ru: { US: "США", GB: "Великобритания", RU: "Россия", KR: "Южная Корея", AE: "ОАЭ" }, en: { US: "USA", GB: "UK", RU: "Russia", KR: "South Korea", AE: "UAE" } };
export const ccName = (cc, lang) => SHORT[lang]?.[cc] || countries.getName(cc, lang === "en" ? "en" : "ru") || cc;
export const numToCC = (n) => countries.numericToAlpha2(String(n).padStart(3, "0"));
export const flag = (cc) => cc.replace(/./g, (c) => String.fromCodePoint(127397 + c.charCodeAt(0)));
export const ccOf = (a) => a.countries || (a.country ? [a.country] : []);
/** { KZ: [stories], UZ: [...] } отсортировано по числу историй */
export function byCountry(list) {
  const m = {};
  list.forEach((a) => ccOf(a).forEach((c) => (m[c] = m[c] || []).push(a)));
  return Object.entries(m).sort((a, b) => b[1].length - a[1].length);
}
