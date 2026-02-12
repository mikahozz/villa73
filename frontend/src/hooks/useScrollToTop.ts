import { useEffect, useRef } from "react";

export function useScrollToTop(timeoutMs: number = 5000) {
  const timerRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    const resetTimer = () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        if (window.scrollY > 0) {
          console.log("Scrolling back to top");
          window.scrollTo({ top: 0, behavior: "smooth" });
        }
      }, timeoutMs);
    };

    window.addEventListener("scroll", resetTimer);
    window.addEventListener("mousemove", resetTimer);

    resetTimer(); // Start timer on mount

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      window.removeEventListener("scroll", resetTimer);
      window.removeEventListener("mousemove", resetTimer);
    };
  }, [timeoutMs]);
}
