/* Skiper UI — skiper58 «TextRoll» (free, skiper-ui.com, registry @skiper-ui/skiper58).
   Оригинальный код; адаптация: framer-motion → motion/react (тот же API), TS-типы убраны, lineHeight вынесен в параметр (для строчных букв с выносными элементами). */
import { motion } from "motion/react";
import React from "react";
import { cn } from "./cn.js";

const STAGGER = 0.035;

export const TextRoll = ({ children, className, center = false, lineHeight = 0.75 }) => {
  return (
    <motion.span
      initial="initial"
      whileHover="hovered"
      className={cn("relative block overflow-hidden", className)}
      style={{
        lineHeight,
      }}
    >
      <div>
        {children.split("").map((l, i) => {
          const delay = center
            ? STAGGER * Math.abs(i - (children.length - 1) / 2)
            : STAGGER * i;

          return (
            <motion.span
              variants={{
                initial: {
                  y: 0,
                },
                hovered: {
                  y: "-100%",
                },
              }}
              transition={{
                ease: "easeInOut",
                delay,
              }}
              className="inline-block"
              key={i}
            >
              {l === " " ? " " : l}
            </motion.span>
          );
        })}
      </div>
      <div className="absolute inset-0">
        {children.split("").map((l, i) => {
          const delay = center
            ? STAGGER * Math.abs(i - (children.length - 1) / 2)
            : STAGGER * i;

          return (
            <motion.span
              variants={{
                initial: {
                  y: "100%",
                },
                hovered: {
                  y: 0,
                },
              }}
              transition={{
                ease: "easeInOut",
                delay,
              }}
              className="inline-block"
              key={i}
            >
              {l === " " ? " " : l}
            </motion.span>
          );
        })}
      </div>
    </motion.span>
  );
};
