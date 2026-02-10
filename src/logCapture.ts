// logCapture.ts
// Wrap console methods to emit a custom event for reactive log display.

type ConsoleLevel = "log" | "info" | "warn" | "error";

interface ConsoleEventDetail {
  level: ConsoleLevel;
  // Use unknown to avoid implicit any; consumer can refine.
  args: unknown[];
  time: number; // epoch ms
}

declare global {
  interface WindowEventMap {
    "app:console": CustomEvent<ConsoleEventDetail>;
  }
}

function dispatch(level: ConsoleLevel, args: unknown[]) {
  const detail: ConsoleEventDetail = {
    level,
    args,
    time: Date.now(),
  };
  window.dispatchEvent(new CustomEvent("app:console", { detail }));
}

// Guard to avoid double-wrapping if module reloaded (e.g. HMR)
interface ConsoleCaptureFlag extends Window {
  __CONSOLE_CAPTURED__?: boolean;
}

const w = window as ConsoleCaptureFlag;

if (!w.__CONSOLE_CAPTURED__) {
  w.__CONSOLE_CAPTURED__ = true;
  const original: Record<ConsoleLevel, (...a: unknown[]) => void> = {
    log: console.log.bind(console),
    info: console.info.bind(console),
    warn: console.warn.bind(console),
    error: console.error.bind(console),
  };

  (console as unknown as Record<string, (...a: unknown[]) => void>).log = (
    ...args: unknown[]
  ) => {
    dispatch("log", args);
    original.log(...args);
  };
  (console as unknown as Record<string, (...a: unknown[]) => void>).info = (
    ...args: unknown[]
  ) => {
    dispatch("info", args);
    original.info(...args);
  };
  (console as unknown as Record<string, (...a: unknown[]) => void>).warn = (
    ...args: unknown[]
  ) => {
    dispatch("warn", args);
    original.warn(...args);
  };
  (console as unknown as Record<string, (...a: unknown[]) => void>).error = (
    ...args: unknown[]
  ) => {
    dispatch("error", args);
    original.error(...args);
  };
}

export {}; // module marker
