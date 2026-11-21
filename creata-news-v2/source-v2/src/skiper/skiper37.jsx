/* Skiper UI — skiper37 «AnimatedNumber» (free, registry @skiper-ui/skiper37).
   Author: @gurvinder-singh02. Free version requires attribution to Skiper UI.
   Взяты AnimatedNumber_004 (число из x в y при появлении, NumberFlow) и NumberFlow-счётчик из AnimatedNumber_001;
   адаптация: TS убран, значения/формат — параметры, крупный демо-шрифт 20vw убран. */
import NumberFlow from "@number-flow/react";
import { animate, useMotionValue } from "motion/react";
import React, { useEffect, useState } from "react";
import { useInView } from "react-intersection-observer";

export function AnimatedNumber_004({ to, from = 0, prefix, suffix, locales, className }) {
  const [displayValue, setDisplayValue] = useState(from);
  const count = useMotionValue(from);
  const { ref, inView } = useInView({ triggerOnce: false });

  useEffect(() => {
    if (inView) {
      animate(count, to, {
        duration: 1,
        ease: "easeInOut",
        onUpdate: (latest) => setDisplayValue(Math.round(latest)),
      });
    } else {
      setDisplayValue(from);
      count.set(from);
    }
  }, [inView, count, to]);

  return (
    <span ref={ref} className={className}>
      <NumberFlow value={displayValue} prefix={prefix} suffix={suffix} locales={locales} />
    </span>
  );
}

/* Тикающая цифра (как в AnimatedNumber_001 — NumberFlow с каждой секундой) */
export function FlowDigit({ value, className }) {
  return <NumberFlow value={value} format={{ minimumIntegerDigits: 2 }} className={className} />;
}
