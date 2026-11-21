/* Skiper UI — skiper61 «SpringMouseFollow» (free, registry @skiper-ui/skiper61).
   Оригинальные пружины SPRING и логика; адаптация: вместо пустого квадрата 500px — обёртка над фото материала,
   за курсором едет фиолетовая плашка «Читать →». */
import { motion, useSpring } from "motion/react";
import React from "react";
import { cn } from "./cn.js";

const SPRING = {
  mass: 0.1,
  damping: 10,
  stiffness: 131,
};

export const SpringMouseFollow = ({ children, label, className }) => {
  const xSpring = useSpring(0, SPRING);
  const ySpring = useSpring(0, SPRING);
  const opacitySpring = useSpring(0, SPRING);
  const scaleSpring = useSpring(0, SPRING);

  return (
    <div
      onPointerMove={(e) => {
        if (e.pointerType === "touch") return;
        const bounds = e.currentTarget.getBoundingClientRect();
        xSpring.set(e.clientX - bounds.left);
        ySpring.set(e.clientY - bounds.top);
      }}
      onPointerEnter={(e) => {
        if (e.pointerType === "touch") return;
        opacitySpring.set(1);
        scaleSpring.set(1);
      }}
      onPointerLeave={() => {
        opacitySpring.set(0);
        scaleSpring.set(0);
      }}
      className={cn("relative overflow-hidden", className)}
    >
      {children}
      <motion.div
        aria-hidden="true"
        style={{ x: xSpring, y: ySpring, opacity: opacitySpring, scale: scaleSpring }}
        className="pointer-events-none absolute left-0 top-0 z-10 -ml-12 -mt-6 grid h-12 w-24 place-items-center rounded-full bg-accent text-[13px] font-semibold text-white shadow-lg"
      >
        {label}
      </motion.div>
    </div>
  );
};
