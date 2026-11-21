# Creata News — фронтенд v2

Открыть: `index.html` (сайт) и `admin.html` (админка). Работает без сервера, данные хранятся в браузере (localStorage).

- Стиль и структура — по restofworld.org, бренд Creata (CR≡ATA, #4B44F0).
- Стек: React 18 + Tailwind 3 + Motion, собрано esbuild в `assets/app.js` / `assets/app.css`.
- Компоненты Skiper UI (бесплатные, skiper-ui.com, атрибуция в футере): skiper58 TextRoll (меню), skiper89 круговой прогресс чтения, skiper37 AnimatedNumber (цифры, обратный отсчёт), skiper16 StickyCard (стопка «Экосистемы»), skiper41 ProgressiveBlur (подписи карточек), skiper54 clipPath-карусель (фото номера), skiper52 HoverExpand (шапка разделов), skiper61 SpringMouseFollow («Читать →» на главном фото). Плюс Lenis — плавная прокрутка из их зависимостей.
- Фото: пересказы — Forbes.kz, Zakon.kz, BES.media, Tengrinews.kz (с подписью и ссылкой на оригинал); «Экосистема» — creata.news/issues/01.
- Языки: RU и EN полностью, ES — «скоро».
- Исходники: `../source-v2` (`npm i && bash build.sh`; тексты и переводы в `content.js`).
