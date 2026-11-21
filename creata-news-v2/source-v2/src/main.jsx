import React, { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { motion, AnimatePresence, MotionConfig } from "motion/react";
import { CN, useStore, useHash, LangCtx } from "./lib.js";
import { Header, DynamicIsland, MenuOverlay, SearchOverlay, Onboarding, Footer } from "./chrome.jsx";
import { Home } from "./home.jsx";
import { Article } from "./article.jsx";
import { Issue } from "./paper.jsx";
import { Shorts } from "./shorts.jsx";
import { Grid, Author, Authors, Opps, Morning, About, Saved, NotFound } from "./pages.jsx";
import { ease } from "./ui.jsx";
import ReactLenis, { useLenis } from "lenis/react";

function applyTheme() {
  const th = CN.prefs.theme;
  if (th === "dark" || th === "light") document.documentElement.setAttribute("data-theme", th);
  else document.documentElement.removeAttribute("data-theme");
  document.documentElement.classList.toggle("reduce", CN.prefs.motion === false);
}

function App() {
  useStore();
  const route = useHash();
  const lang = CN.prefs.lang === "en" ? "en" : "ru";
  const [menu, setMenu] = useState(false), [search, setSearch] = useState(false), [sub, setSub] = useState(false);
  useEffect(() => { document.documentElement.lang = lang; }, [lang]);
  applyTheme();
  const lenis = useLenis();
  useEffect(() => { if (route.startsWith("shorts")) return; window.scrollTo(0, 0); lenis?.scrollTo(0, { immediate: true, force: true }); }, [route]);

  let page, title = "Creata News";
  const art = route && CN.article(route);
  const shorts = route.startsWith("shorts");
  if (!route || shorts) page = <Home onSubscribe={() => setSub(true)} />;
  else if (art && art.status === "published" && (lang !== "en" || art.title_en)) { page = <Article a={art} />; title = (lang === "en" ? art.title_en : art.title) + " — Creata News"; }
  else if (route === "world") page = <Grid layer="world" />;
  else if (/^world-[A-Z]{2}$/.test(route)) page = <Grid layer="world" country={route.slice(6)} />;
  else if (route === "ecosystem") page = <Grid layer="eco" />;
  else if (route.startsWith("v-")) page = <Grid vertical={route.slice(2)} />;
  else if (route.startsWith("author-")) page = <Author id={route.slice(7)} />;
  else if (route === "authors") page = <Authors />;
  else if (route === "opportunities") page = <Opps />;
  else if (route === "issue") page = <Issue />;
  else if (route === "morning") page = <Morning onSubscribe={() => setSub(true)} />;
  else if (route === "about") page = <About />;
  else if (route === "saved") page = <Saved />;
  else page = <NotFound />;
  useEffect(() => { document.title = title; }, [title]);

  return (
    <LangCtx.Provider value={lang}>
      <MotionConfig reducedMotion={CN.prefs.motion === false ? "always" : "user"}>
        <a href="#view" onClick={(e) => { e.preventDefault(); document.getElementById("view").focus(); }} className="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-3 focus:z-[70] focus:rounded focus:bg-accent focus:px-3 focus:py-2 focus:text-white">
          {lang === "en" ? "Skip to content" : "К материалам"}
        </a>
        <Header route={route} onMenu={() => setMenu(true)} onSearch={() => setSearch(true)} onSubscribe={() => setSub(true)} />
        <main id="view" tabIndex={-1} className="outline-none">
          <AnimatePresence mode="wait">
            <motion.div key={(shorts ? "" : route) + lang} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} transition={{ duration: 0.35, ease }}>
              {page}
            </motion.div>
          </AnimatePresence>
        </main>
        <Footer onSubscribe={() => setSub(true)} />
        <DynamicIsland route={route} />
        {shorts && <Shorts start={route.slice(7)} />}
        <MenuOverlay open={menu} onClose={() => setMenu(false)} />
        <SearchOverlay open={search} onClose={() => setSearch(false)} />
        <Onboarding open={sub} onClose={() => setSub(false)} />
      </MotionConfig>
    </LangCtx.Provider>
  );
}

createRoot(document.getElementById("root")).render(
  <ReactLenis root options={{ lerp: 0.12 }}>
    <App />
  </ReactLenis>
);
