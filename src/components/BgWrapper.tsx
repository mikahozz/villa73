import styles from "./BgWrapper.module.css";
import { useEffect, useRef, useState } from "react";
import { useDayTime } from "../hooks/useDayTime";
import { getTimeOfYear } from "../hooks/useTimeOfYear";

// Import all images from assets/img
const images = Object.values(
  import.meta.glob("../assets/img/*.jpeg", {
    eager: true,
    query: "?url",
    import: "default",
  })
) as string[];

function useBg(intervalMs: number = 1 * 60 * 60 * 1000) {
  const indexRef = useRef(0);
  const { isDayTime } = useDayTime();
  const [bgPath, setBgPath] = useState<string | null>(null);

  useEffect(() => {
    console.log(
      "Bg: useEffect called. isDayTime:",
      isDayTime(new Date()),
      " intervalMs:",
      intervalMs
    );
    if (images.length === 0) return;
    const filteredImages = images
      .filter((img) => img.includes(getTimeOfYear()))
      .filter((img) => {
        if (isDayTime(new Date())) {
          // Include images with "day" in the filename
          return img.includes("day");
        } else {
          // Include images with "night" in the filename
          return img.includes("night");
        }
      });
    console.log(
      "Bg: Filtered images:",
      filteredImages.map((path) => path.split("/").at(-1))
    );
    const setBg = () => {
      indexRef.current = Math.floor(Math.random() * filteredImages.length);
      console.log(
        "Bg: Swapping background to:",
        filteredImages[indexRef.current]
      );
      setBgPath(filteredImages[indexRef.current]);
    };

    setBg();

    const interval = setInterval(() => {
      setBg();
    }, intervalMs);

    return () => {
      console.log("Bg: Clearing background image interval.");
      clearInterval(interval);
      document.body.style.backgroundImage = "";
    };
  }, [intervalMs, isDayTime]);

  return { bgPath };
}

interface BgProps {
  children: React.ReactNode;
}

export default function BgWrapper({ children }: BgProps) {
  const { bgPath } = useBg();
  const [dimmed, setDimmed] = useState(false);
  const bgReturnTimeout = 30000;

  const handleBgClick = () => {
    const updatedDimmedVal = !dimmed;
    console.log("Bg: Setting foreground dimmed to ", updatedDimmedVal);
    setDimmed(updatedDimmedVal);
    if (updatedDimmedVal) {
      const bringBack = setTimeout(() => {
        console.log("Bg: Returning foreground to normal");
        setDimmed(false);
      }, bgReturnTimeout);
      return () => clearTimeout(bringBack);
    }
  };

  return (
    <>
      <div
        className={styles.bg}
        style={{ backgroundImage: `url(${bgPath})` }}
        onClick={handleBgClick}
      >
        <div className={styles.bgOverlay} onClick={handleBgClick}>
          <div
            className={styles.bgChildren}
            style={{
              opacity: dimmed ? 0.1 : 1,
              transition: "opacity 0.3s",
              pointerEvents: dimmed ? "none" : "auto",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {children}
          </div>
        </div>
      </div>
    </>
  );
}
