/* Skiper UI — skiper16 «StickyCard_001»: стопка карточек при прокрутке (free, registry @skiper-ui/skiper16).
   Author: @gurvinder-singh02. Free version requires attribution to Skiper UI.
   Оригинальная механика (sticky + useScroll/useTransform scale); адаптация: TS убран, карточка — материал газеты
   (фото + заголовок, снизу ProgressiveBlur из skiper41), ширина резиновая для телефона. ReactLenis подключён один раз на весь сайт в main.jsx. */
import { motion, useScroll, useTransform } from "motion/react";
import React, { useRef } from "react";
import { ProgressiveBlur } from "./skiper41.jsx";

export const StickyCard_001 = ({ i, title, kicker, src, href, progress, range, targetScale }) => {
  const container = useRef(null);
  const scale = useTransform(progress, range, [1, targetScale]);

  return (
    <div ref={container} className="sticky top-0 flex h-[70vh] items-center justify-center sm:h-screen">
      <motion.a
        href={href}
        style={{ scale, top: `calc(-5vh + ${i * 20 + 120}px)` }}
        className="group relative -top-1/4 flex h-[300px] w-[min(92vw,720px)] origin-top flex-col overflow-hidden rounded-[2rem] sm:h-[420px]"
      >
        <img src={src} alt={title} loading="lazy" className="h-full w-full object-cover" />
        <ProgressiveBlur position="bottom" backgroundColor="rgba(14,14,16,0.85)" height="55%" blurAmount="6px" />
        <span className="absolute bottom-0 left-0 right-0 p-5 text-white sm:p-8">
          <span className="block font-mono text-[11px] uppercase tracking-wider text-[#B9B6FF]">{kicker}</span>
          <span className="mt-1 block font-head text-[22px] font-semibold leading-tight group-hover:underline sm:text-[32px]">{title}</span>
        </span>
      </motion.a>
    </div>
  );
};

export const Skiper16 = ({ items, children }) => {
  const container = useRef(null);
  const { scrollYProgress } = useScroll({ target: container, offset: ["start start", "end end"] });

  return (
    <div ref={container} className="relative flex w-full flex-col items-center justify-center pb-[4vh] pt-[6vh]">
      {children}
      {items.map((project, i) => {
        const targetScale = Math.max(0.5, 1 - (items.length - i - 1) * 0.1);
        return (
          <StickyCard_001 key={`p_${i}`} i={i} {...project} progress={scrollYProgress} range={[i * 0.25, 1]} targetScale={targetScale} />
        );
      })}
    </div>
  );
};
