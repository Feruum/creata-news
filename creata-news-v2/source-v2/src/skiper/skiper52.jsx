/* Skiper UI — skiper52 «HoverExpand_001» (free, registry @skiper-ui/skiper52).
   Author: @gurvinder-singh02. Free version requires attribution to Skiper UI.
   Оригинальный код; адаптация: TS убран, неиспользуемые swiper-стили убраны, размеры параметризованы
   (на телефоне меньше), подпись `code` — ссылка на материал. */
import { AnimatePresence, motion } from "motion/react";
import React, { useState } from "react";
import { cn } from "./cn.js";

export const HoverExpand_001 = ({ images, className, active = "24rem", idle = "5rem", height = "24rem" }) => {
  const [activeImage, setActiveImage] = useState(1);

  return (
    <motion.div
      initial={{ opacity: 0, translateY: 20 }}
      whileInView={{ opacity: 1, translateY: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.3, delay: 0.2 }}
      className={cn("relative w-full max-w-6xl", className)}
    >
      <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }} className="w-full">
        <div className="flex w-full items-center justify-center gap-1">
          {images.map((image, index) => (
            <motion.div
              key={index}
              className="relative shrink-0 cursor-pointer overflow-hidden rounded-3xl"
              initial={{ width: "2.5rem", height: "20rem" }}
              animate={{ width: activeImage === index ? active : idle, height }}
              transition={{ duration: 0.3, ease: "easeInOut" }}
              onClick={() => setActiveImage(index)}
              onHoverStart={() => setActiveImage(index)}
            >
              <AnimatePresence>
                {activeImage === index && (
                  <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="absolute h-full w-full bg-gradient-to-t from-black/70 to-transparent" />
                )}
              </AnimatePresence>
              <AnimatePresence>
                {activeImage === index && (
                  <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="absolute flex h-full w-full flex-col items-start justify-end p-4">
                    {image.href ? (
                      <a href={image.href} className="text-left text-[13px] font-semibold leading-snug text-white hover:underline">{image.code}</a>
                    ) : (
                      <p className="text-left text-xs text-white/70">{image.code}</p>
                    )}
                  </motion.div>
                )}
              </AnimatePresence>
              <img src={image.src} className="size-full object-cover" alt={image.alt} loading="lazy" />
            </motion.div>
          ))}
        </div>
      </motion.div>
    </motion.div>
  );
};
