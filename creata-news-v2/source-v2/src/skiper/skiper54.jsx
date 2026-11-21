/* Skiper UI — skiper54 «Carousel_006»: clipPath-карусель (free, registry @skiper-ui/skiper54).
   Author: @gurvinder-singh02. Built with shadcn/ui + Embla. Free version requires attribution to Skiper UI.
   Оригинальный код; адаптация: TS убран, цвета подписи/точек под тему сайта, подпись может быть массивом. */
import Autoplay from "embla-carousel-autoplay";
import { AnimatePresence, motion } from "motion/react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import React, { useEffect, useState } from "react";
import { cn } from "./cn.js";
import { Carousel, CarouselContent, CarouselItem } from "./carousel.jsx";

export const Carousel_006 = ({ images, className, autoplay = false, loop = true, showNavigation = true, showPagination = true }) => {
  const [api, setApi] = useState();
  const [current, setCurrent] = useState(0);

  useEffect(() => {
    if (!api) return;
    api.on("select", () => {
      setCurrent(api.selectedScrollSnap());
    });
  }, [api]);

  return (
    <Carousel
      setApi={setApi}
      className={cn("w-full", className)}
      opts={{ loop, slidesToScroll: 1 }}
      plugins={autoplay ? [Autoplay({ delay: 2000, stopOnInteraction: true, stopOnMouseEnter: true })] : []}
    >
      <CarouselContent className="flex h-[500px] w-full">
        {images.map((img, index) => (
          <CarouselItem
            key={index}
            className="relative flex h-[81.5%] w-full basis-[73%] items-center justify-center sm:basis-[50%] md:basis-[30%] lg:basis-[25%] xl:basis-[21%]"
          >
            <motion.div
              initial={false}
              animate={{ clipPath: current !== index ? "inset(15% 0 15% 0 round 2rem)" : "inset(0 0 0 0 round 2rem)" }}
              className="h-full w-full overflow-hidden rounded-3xl"
            >
              <div className="relative h-full w-full border border-rule">
                <img src={img.src} alt={img.alt} loading="lazy" className="h-full w-full scale-105 object-cover" />
              </div>
            </motion.div>
            <AnimatePresence mode="wait">
              {current === index && (
                <motion.div
                  initial={{ opacity: 0, filter: "blur(10px)" }}
                  animate={{ opacity: 1, filter: "blur(0px)" }}
                  transition={{ duration: 0.5 }}
                  className="absolute bottom-0 left-2 flex h-[14%] w-full translate-y-full items-center justify-center p-2 text-center text-[14px] font-medium leading-tight tracking-tight text-ink/70"
                >
                  {img.title}
                </motion.div>
              )}
            </AnimatePresence>
          </CarouselItem>
        ))}
      </CarouselContent>

      {showNavigation && (
        <div className="absolute -bottom-4 right-0 flex w-full items-center justify-between gap-2 px-4">
          <button aria-label="Previous slide" onClick={() => api?.scrollPrev()} className="rounded-full bg-ink/80 p-2 hover:bg-accent">
            <ChevronLeft className="text-paper" />
          </button>
          <button aria-label="Next slide" onClick={() => api?.scrollNext()} className="rounded-full bg-ink/80 p-2 hover:bg-accent">
            <ChevronRight className="text-paper" />
          </button>
        </div>
      )}

      {showPagination && (
        <div className="flex w-full items-center justify-center">
          <div className="flex items-center justify-center gap-2">
            {Array.from({ length: images.length }).map((_, index) => (
              <button
                key={index}
                onClick={() => api?.scrollTo(index)}
                className={cn("h-2 w-2 cursor-pointer rounded-full transition-all", current === index ? "bg-ink" : "bg-rule")}
                aria-label={`Go to slide ${index + 1}`}
              />
            ))}
          </div>
        </div>
      )}
    </Carousel>
  );
};
