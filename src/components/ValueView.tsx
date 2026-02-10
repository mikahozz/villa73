import { useState } from "react";
import styles from "./ConsoleLog.module.css";

export interface ValueViewProps {
  value: unknown;
  depth: number;
  maxDepth?: number;
}

// Simple drill-down viewer for objects/arrays.
export function ValueView({ value, depth, maxDepth = 5 }: ValueViewProps) {
  const [open, setOpen] = useState(false);

  if (value === null) {
    return <span className={styles.valNull}>null</span>; // null
  }

  const type = typeof value;
  if (type === "string") {
    return (
      <span className={styles.valString}>&quot;{value as string}&quot;</span>
    );
  }
  if (type === "number" || type === "bigint") {
    return <span className={styles.valNumber}>{String(value)}</span>;
  }
  if (type === "boolean") {
    return <span className={styles.valBoolean}>{String(value)}</span>;
  }
  if (type === "undefined") {
    return <span className={styles.valUndefined}>undefined</span>; // undefined
  }
  if (type === "function") {
    const fn = value as (...args: unknown[]) => unknown;
    return (
      <span className={styles.valFunction}>ƒ {fn.name || "anonymous"}</span>
    );
  }

  // Handle Date
  if (value instanceof Date) {
    return <span className={styles.valDate}>{value.toISOString()}</span>;
  }

  // Handle Error
  if (value instanceof Error) {
    return (
      <span className={styles.valError}>
        {value.name}: {value.message}
      </span>
    );
  }

  const isArray = Array.isArray(value);
  const isObject = !isArray && type === "object";

  if ((isArray || isObject) && depth >= maxDepth) {
    return (
      <span className={styles.valTruncated}>
        {isArray ? "[Array]" : "{Object}"}
      </span>
    );
  }

  if (isArray) {
    const arr = value as unknown[];
    return (
      <span className={styles.valArray}>
        <Toggle open={open} setOpen={setOpen} />[ {arr.length} ]
        {open && (
          <span className={styles.children}>
            {arr.map((v, i) => (
              <div key={i} className={styles.childRow}>
                <span className={styles.key}>{i}:</span>{" "}
                <ValueView value={v} depth={depth + 1} maxDepth={maxDepth} />
              </div>
            ))}
          </span>
        )}
      </span>
    );
  }

  if (isObject) {
    const obj = value as Record<string, unknown>;
    const entries = Object.entries(obj);
    return (
      <span className={styles.valObject}>
        <Toggle open={open} setOpen={setOpen} />
        {"{"}
        {"}"} {entries.length} keys
        {open && (
          <span className={styles.children}>
            {entries.map(([k, v]) => (
              <div key={k} className={styles.childRow}>
                <span className={styles.key}>{k}:</span>{" "}
                <ValueView value={v} depth={depth + 1} maxDepth={maxDepth} />
              </div>
            ))}
          </span>
        )}
      </span>
    );
  }

  // Fallback serialization
  try {
    return <span className={styles.valOther}>{JSON.stringify(value)}</span>;
  } catch {
    return <span className={styles.valOther}>[unserializable]</span>;
  }
}

function Toggle({
  open,
  setOpen,
}: {
  open: boolean;
  setOpen: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      className={styles.toggleBtn}
      onClick={() => setOpen(!open)}
      aria-label={open ? "Collapse" : "Expand"}
    >
      {open ? "▾" : "▸"}
    </button>
  );
}
