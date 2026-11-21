/* shadcn/ui Carousel (Embla) — минимальная JS-версия стандартного компонента shadcn, нужна для skiper54 */
import React, { createContext, useContext, useEffect } from "react";
import useEmblaCarousel from "embla-carousel-react";
import { cn } from "./cn.js";

const Ctx = createContext(null);
export function Carousel({ opts, plugins, setApi, className, children, ...props }) {
  const [carouselRef, api] = useEmblaCarousel({ ...opts, axis: "x" }, plugins);
  useEffect(() => { if (api && setApi) setApi(api); }, [api, setApi]);
  return (
    <Ctx.Provider value={{ carouselRef, api }}>
      <div className={cn("relative", className)} role="region" aria-roledescription="carousel" {...props}>{children}</div>
    </Ctx.Provider>
  );
}
export function CarouselContent({ className, ...props }) {
  const { carouselRef } = useContext(Ctx);
  return <div ref={carouselRef} className="overflow-hidden"><div className={cn("flex -ml-4", className)} {...props} /></div>;
}
export function CarouselItem({ className, ...props }) {
  return <div role="group" aria-roledescription="slide" className={cn("min-w-0 shrink-0 grow-0 basis-full pl-4", className)} {...props} />;
}
