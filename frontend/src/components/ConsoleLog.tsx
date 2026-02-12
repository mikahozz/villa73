import styles from "./ConsoleLog.module.css";
import { ActivityLogIcon, Cross1Icon } from "@radix-ui/react-icons";
import * as Dialog from "@radix-ui/react-dialog";
import { useEffect, useState, useCallback, useRef } from "react";
import { ValueView } from "./ValueView";

type LogEntry = {
  id: number;
  level: "log" | "info" | "warn" | "error";
  args: unknown[];
  time: number;
};

export default function ConsoleLog() {
  const [open, setOpen] = useState(false);
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const nextId = useRef(1);
  const listRef = useRef<HTMLDivElement | null>(null);
  const maxEntriesShown = 50;

  const onConsoleEvent = useCallback((e: CustomEvent) => {
    const detail = e.detail as {
      level: LogEntry["level"];
      args: unknown[];
      time: number;
    };
    setEntries((prev) => [
      {
        id: nextId.current++,
        level: detail.level,
        args: detail.args,
        time: detail.time,
      },
      ...prev,
    ]);
  }, []);

  useEffect(() => {
    const handler = (e: Event) => onConsoleEvent(e as CustomEvent);
    window.addEventListener("app:console", handler as EventListener);
    return () =>
      window.removeEventListener("app:console", handler as EventListener);
  }, [onConsoleEvent]);

  // Auto-scroll to bottom when new entries appear and dialog open
  useEffect(() => {
    if (open && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [entries, open]);

  const clear = () => setEntries([]);

  return (
    <div className={styles.consoleLog}>
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog.Trigger asChild>
          <button className={styles.iconTrigger} aria-label="Open logs">
            <ActivityLogIcon width={24} height={24} />
          </button>
        </Dialog.Trigger>
        <Dialog.Portal>
          <Dialog.Overlay className="DialogOverlay" />
          <Dialog.Content className="DialogContent">
            <Dialog.Title asChild className={styles.title}>
              <div>
                <h2>Console Output</h2>
                <button
                  onClick={clear}
                  className={styles.clearBtn}
                  disabled={!entries.length}
                >
                  Clear ({entries.length})
                </button>
                <input
                  type="text"
                  placeholder="Search logs..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </Dialog.Title>
            <Dialog.Description
              className={styles.description}
            ></Dialog.Description>
            <div className={styles.toolbar}>
              <button
                onClick={() => setOpen(false)}
                className={styles.closeBtn}
                aria-label="Close"
              >
                <Cross1Icon />
              </button>
            </div>
            <div ref={listRef} className={styles.logScrollArea}>
              {!entries.length ? (
                <div className={styles.empty}>
                  No messages yet. Open dev tools and generate logs!
                </div>
              ) : (
                entries
                  .filter((entry) =>
                    entry.args.some((arg) =>
                      String(arg)
                        .toLowerCase()
                        .includes(searchTerm.toLowerCase())
                    )
                  )
                  .slice(0, maxEntriesShown)
                  .map((entry) => <LogEntryView key={entry.id} entry={entry} />)
              )}
            </div>
            <Dialog.Close asChild>
              <button
                className={styles.hiddenClose}
                aria-label="Close dialog"
              />
            </Dialog.Close>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  );
}

function LogEntryView({ entry }: { entry: LogEntry }) {
  const ts = new Date(entry.time).toLocaleTimeString();
  return (
    <div className={styles.logEntry} data-level={entry.level}>
      <div className={styles.logMeta}>
        <span className={styles.time}>{ts}</span>
        <span className={styles.level + " " + styles["level_" + entry.level]}>
          {entry.level.toUpperCase()}
        </span>
      </div>
      <div className={styles.args}>
        {entry.args.map((arg, idx) => (
          <ValueView key={idx} value={arg} depth={0} />
        ))}
      </div>
    </div>
  );
}
