import { useEffect, useState } from "react";
import { CheckCircledIcon, Cross2Icon } from "@radix-ui/react-icons";
import {
  useCompleteFamilyTask,
  useDeleteFamilyTask,
  useFamilyTasks,
} from "../hooks/useFamilyTasks";
import { useSwipeCarousel } from "../hooks/useSwipeCarousel";
import type { FamilyTask } from "../types/familyTask";
import styles from "./FamilyTasksCard.module.css";

const COMPLETER_OPTIONS = ["Elias", "Elise", "Ella", "Iskä", "Äiti"];
const ASSIGNEE_PATTERN = /\b(elias|elise|ella|iskä|äiti)\b/giu;

export function FamilyTasksCard() {
  const { data: tasks = [], isLoading } = useFamilyTasks();
  const { mutateAsync: completeTask, isPending: isCompleting } =
    useCompleteFamilyTask();
  const { mutateAsync: deleteTask, isPending: isDeleting } =
    useDeleteFamilyTask();

  const [completerByTaskId, setCompleterByTaskId] = useState<
    Record<string, string>
  >({});
  const [completedByTaskId, setCompletedByTaskId] = useState<
    Record<string, string>
  >({});
  const [completionPromptTaskId, setCompletionPromptTaskId] = useState<
    string | null
  >(null);

  const carousel = useSwipeCarousel({ count: tasks.length });

  useEffect(() => {
    setCompleterByTaskId((current) => {
      const next: Record<string, string> = {};
      for (const task of tasks) {
        next[task.id] = current[task.id] ?? task.assignee ?? "";
      }
      return next;
    });
  }, [tasks]);

  useEffect(() => {
    // Keep height in sync when completion turns a full card into compact mode.
    carousel.refreshHeight();
  }, [carousel.refreshHeight, completedByTaskId]);

  const completeTaskWithCompleter = async (
    task: FamilyTask,
    completer?: string,
  ) => {
    const normalizedCompleter = completer?.trim() || "Unknown";
    await completeTask({ taskId: task.id, completer: normalizedCompleter });
    setCompletedByTaskId((current) => ({
      ...current,
      [task.id]: normalizedCompleter,
    }));
  };

  const onCompleteTask = async (task: FamilyTask) => {
    const completer = (completerByTaskId[task.id] ?? "").trim();
    if (!task.assignee && !completer) {
      setCompletionPromptTaskId(task.id);
      return;
    }
    await completeTaskWithCompleter(task, completer || task.assignee);
  };

  const onDeleteTask = async (task: FamilyTask) => {
    if (!window.confirm(`Delete task "${task.title}" permanently?`)) {
      return;
    }
    await deleteTask({ taskId: task.id });
  };

  if (!isLoading && tasks.length === 0) {
    return null;
  }

  const promptTask = tasks.find((task) => task.id === completionPromptTaskId);
  const promptCompleter = promptTask
    ? (completerByTaskId[promptTask.id] ?? "").trim()
    : "";

  return (
    <div className={styles.container} data-testid="family-tasks-container">
      {tasks.length > 1 ? (
        <div
          className={styles.carouselIndicators}
          data-testid="carousel-indicators"
        >
          {tasks.map((task, taskIndex) => (
            <button
              key={`indicator-${task.id}`}
              type="button"
              className={`${styles.indicatorButton} ${taskIndex === carousel.index ? styles.indicatorButtonActive : ""}`}
              onClick={() => {
                carousel.setIndex(taskIndex);
                carousel.scrollToIndex(taskIndex, "smooth");
              }}
              aria-label={`Show task ${taskIndex + 1}`}
            />
          ))}
        </div>
      ) : null}

      <div
        className={`${styles.cardsViewport} ${carousel.isInteracting ? styles.cardsViewportInteractive : ""}`}
        style={
          carousel.viewportHeight
            ? { height: `${carousel.viewportHeight}px` }
            : undefined
        }
      >
        <div
          ref={carousel.trackRef}
          className={`${styles.cardsTrack} ${carousel.isInteracting ? styles.cardsTrackInteractive : ""}`}
          onScroll={carousel.onTrackScroll}
          onPointerDown={(event) => {
            event.currentTarget.setPointerCapture(event.pointerId);
            carousel.onPointerDown(event.clientX);
          }}
          onPointerUp={(event) => {
            if (event.currentTarget.hasPointerCapture(event.pointerId)) {
              event.currentTarget.releasePointerCapture(event.pointerId);
            }
            carousel.onPointerUp(event.clientX);
          }}
          onPointerCancel={(event) => {
            if (event.currentTarget.hasPointerCapture(event.pointerId)) {
              event.currentTarget.releasePointerCapture(event.pointerId);
            }
            carousel.onPointerCancel();
          }}
          data-testid="family-task-track"
        >
          {tasks.map((task) => {
            const completer = completerByTaskId[task.id] ?? "";
            const localCompletedBy = completedByTaskId[task.id];
            const isTaskCompleted =
              task.completed === true || !!localCompletedBy;
            const completedByText =
              localCompletedBy || task.completedBy || "Unknown";
            const showNote = shouldShowNote(task.note);
            const sizeEmoji = toSizeEmoji(task.size);

            return (
              <article
                key={task.id}
                className={`${styles.taskCard} ${isTaskCompleted ? styles.taskCardCompleted : ""}`}
                data-testid="family-task-card"
              >
                {isTaskCompleted ? (
                  <div
                    className={styles.completedRow}
                    data-testid="task-completed"
                  >
                    <div className={styles.taskTitleWrap}>
                      <h3 className={styles.taskTitle}>{task.title}</h3>
                      {sizeEmoji ? (
                        <span className={styles.sizeBiceps}>{sizeEmoji}</span>
                      ) : null}
                    </div>
                    <span className={styles.completedByText}>
                      Completed by {completedByText}
                    </span>
                  </div>
                ) : (
                  <>
                    <div className={styles.taskCardHeader}>
                      <div className={styles.taskTitleWrap}>
                        <span className={styles.taskIcon} aria-hidden="true">
                          <CheckCircledIcon width={26} height={26} />
                        </span>
                        <h3 className={styles.taskTitle}>{task.title}</h3>
                        {sizeEmoji ? (
                          <span className={styles.sizeBiceps}>{sizeEmoji}</span>
                        ) : null}
                      </div>
                      <button
                        type="button"
                        className={styles.deleteTaskButton}
                        aria-label={`Delete task ${task.title}`}
                        onClick={() => onDeleteTask(task)}
                        disabled={isDeleting}
                      >
                        <Cross2Icon />
                      </button>
                    </div>
                    <div className={styles.taskBody}>
                      {showNote ? (
                        <div className={styles.taskNote}>{task.note}</div>
                      ) : null}
                    </div>
                    <div className={styles.taskFooter}>
                      {task.assignee ? (
                        <div className={styles.assigneeControl}>
                          <span className={styles.assigneeInitial}>
                            {completer ? completer[0].toUpperCase() : "?"}
                          </span>
                          <select
                            id="cardCompleterSelect"
                            className={styles.completerSelect}
                            value={completer}
                            onChange={(event) =>
                              setCompleterByTaskId((current) => ({
                                ...current,
                                [task.id]: event.target.value,
                              }))
                            }
                            data-testid="completer-select"
                          >
                            {COMPLETER_OPTIONS.map((option) => (
                              <option key={option} value={option}>
                                {option}
                              </option>
                            ))}
                          </select>
                        </div>
                      ) : null}
                      <button
                        type="button"
                        className={styles.openCompleteButton}
                        onClick={() => onCompleteTask(task)}
                        disabled={isCompleting}
                        data-testid="complete-task-button"
                      >
                        <CheckCircledIcon /> Complete task
                      </button>
                    </div>
                  </>
                )}
              </article>
            );
          })}
        </div>
      </div>

      {promptTask ? (
        <div
          className={styles.completePromptOverlay}
          data-testid="complete-prompt"
        >
          <div className={styles.completePromptCard}>
            <h4 className={styles.completePromptTitle}>Complete task</h4>
            <p className={styles.completePromptText}>
              Select a person or complete without assignee.
            </p>
            <select
              className={styles.completePromptSelect}
              value={promptCompleter}
              onChange={(event) =>
                setCompleterByTaskId((current) => ({
                  ...current,
                  [promptTask.id]: event.target.value,
                }))
              }
              data-testid="complete-prompt-select"
            >
              <option value="">Select a person (optional)</option>
              {COMPLETER_OPTIONS.map((option) => (
                <option key={`prompt-${option}`} value={option}>
                  {option}
                </option>
              ))}
            </select>
            <div className={styles.completePromptActions}>
              <button
                type="button"
                className={styles.completePromptButtonGhost}
                onClick={() => setCompletionPromptTaskId(null)}
              >
                Cancel
              </button>
              <button
                type="button"
                className={styles.completePromptButton}
                onClick={async () => {
                  await completeTaskWithCompleter(promptTask, promptCompleter);
                  setCompletionPromptTaskId(null);
                }}
                disabled={isCompleting}
              >
                Complete task
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function shouldShowNote(note?: string): boolean {
  if (!note) {
    return false;
  }
  const withoutAssignee = note.replaceAll(ASSIGNEE_PATTERN, " ");
  const compact = withoutAssignee
    .replaceAll(/[\s,.:;!?()[\]{}"'`-]/g, "")
    .trim();
  return compact.length > 0;
}

function toSizeEmoji(size: number): string {
  if (size <= 0) {
    return "";
  }
  const count = Math.max(1, Math.min(5, Math.trunc(size)));
  return "💪".repeat(count);
}
