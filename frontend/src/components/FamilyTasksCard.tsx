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

  const carousel = useSwipeCarousel({ count: tasks.length });

  useEffect(() => {
    setCompleterByTaskId((current) => {
      const next: Record<string, string> = {};
      for (const task of tasks) {
        next[task.id] =
          current[task.id] ?? task.assignee ?? COMPLETER_OPTIONS[0];
      }
      return next;
    });
  }, [tasks]);

  useEffect(() => {
    // Keep height in sync when completion turns a full card into compact mode.
    carousel.refreshHeight();
  }, [carousel.refreshHeight, completedByTaskId]);

  const onCompleteTask = async (task: FamilyTask) => {
    const completer = completerByTaskId[task.id] ?? COMPLETER_OPTIONS[0];
    await completeTask({ taskId: task.id, completer });
    setCompletedByTaskId((current) => ({ ...current, [task.id]: completer }));
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
            const completer = completerByTaskId[task.id] ?? COMPLETER_OPTIONS[0];
            const localCompletedBy = completedByTaskId[task.id];
            const isTaskCompleted = task.completed === true || !!localCompletedBy;
            const completedByText =
              localCompletedBy || task.completedBy || "Unknown";
            const showNote = shouldShowNote(task.note);

            return (
              <article
                key={task.id}
                className={`${styles.taskCard} ${isTaskCompleted ? styles.taskCardCompleted : ""}`}
                data-testid="family-task-card"
              >
                <button
                  type="button"
                  className={styles.deleteTaskButton}
                  aria-label={`Delete task ${task.title}`}
                  onClick={() => onDeleteTask(task)}
                  disabled={isDeleting}
                >
                  <Cross2Icon />
                </button>

                {isTaskCompleted ? (
                  <div className={styles.completedRow} data-testid="task-completed">
                    <h3 className={styles.taskTitle}>{task.title}</h3>
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
                      </div>
                    </div>
                    <div className={styles.taskBody}>
                      {showNote ? (
                        <div className={styles.taskNote}>{task.note}</div>
                      ) : (
                        <div className={styles.taskNoteMuted}>
                          No additional notes
                        </div>
                      )}
                    </div>
                    <div className={styles.taskFooter}>
                      <div className={styles.assigneeControl}>
                        <span className={styles.assigneeInitial}>
                          {completer[0].toUpperCase()}
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
                    {task.size > 0 ? (
                      <span className={styles.sizeBadge}>{task.size}</span>
                    ) : null}
                  </>
                )}
              </article>
            );
          })}
        </div>
      </div>
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
