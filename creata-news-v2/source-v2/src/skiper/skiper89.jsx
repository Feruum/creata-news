/* Skiper UI — skiper89 «круговой прогресс прокрутки» (free, registry @skiper-ui/skiper89).
   Оригинальная логика и разметка индикатора; убран демо-текст Lorem ipsum, цвета через токены сайта. */
import NumberFlow from "@number-flow/react";
import { motion, useMotionValueEvent, useScroll, useTransform } from "motion/react";
import React, { useState } from "react";
import { cn } from "./cn.js";

export const Skiper89 = ({ className }) => {
  const { scrollYProgress } = useScroll();
  const [progressPercent, setProgressPercent] = useState(0);

  const clampedProgress = useTransform(scrollYProgress, (value) => Math.min(Math.max(value, 0), 1));
  const progressAsPercent = useTransform(clampedProgress, (value) => Math.round(value * 100));

  useMotionValueEvent(progressAsPercent, "change", (value) => {
    setProgressPercent(value);
  });

  const svgRadius = 18;
  const circumference = 2 * Math.PI * svgRadius;

  return (
    <motion.div
      drag
      dragMomentum={false}
      className={cn("group fixed bottom-4 right-4 z-40 cursor-grab items-center gap-1 text-accent active:cursor-grabbing", className)}
    >
      <NumberFlow
        value={progressPercent}
        className={cn("absolute top-1 flex h-8 -translate-y-full items-center justify-center px-4 text-xs font-medium tabular-nums text-ink/60 opacity-0 group-hover:opacity-100")}
        suffix="%"
      />
      <div className="flex size-12 items-center justify-center rounded-2xl border border-rule bg-paper/30 backdrop-blur">
        <svg className={cn("size-10")} viewBox="0 0 48 48" role="presentation">
          <circle cx="24" cy="24" r={svgRadius} stroke="currentColor" strokeWidth="3" className={cn("opacity-30")} fill="none" />
          <motion.circle
            cx="24"
            cy="24"
            r={svgRadius}
            stroke="currentColor"
            strokeWidth="3"
            fill="none"
            strokeLinecap="round"
            strokeDasharray={`${circumference}`}
            style={{ pathLength: clampedProgress, rotate: -90, transformOrigin: "50% 50%" }}
          />
        </svg>
      </div>
    </motion.div>
  );
};
