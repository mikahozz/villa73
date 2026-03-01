import { describe, expect, test } from "vitest";
import {
  FAMILY_TASKS_REFRESH_EMPTY_MS,
  FAMILY_TASKS_REFRESH_WITH_DATA_MS,
  getFamilyTasksRefetchInterval,
} from "./useFamilyTasks";

describe("useFamilyTasks polling intervals", () => {
  test("uses 1h interval when tasks exist", () => {
    const interval = getFamilyTasksRefetchInterval([
      {
        id: "1",
        title: "Task",
        note: "",
        labels: [],
        size: 1,
        createdAt: new Date().toISOString(),
      },
    ]);
    expect(interval).toBe(FAMILY_TASKS_REFRESH_WITH_DATA_MS);
  });

  test("uses 15min interval when no tasks", () => {
    expect(getFamilyTasksRefetchInterval([])).toBe(
      FAMILY_TASKS_REFRESH_EMPTY_MS,
    );
    expect(getFamilyTasksRefetchInterval(undefined)).toBe(
      FAMILY_TASKS_REFRESH_EMPTY_MS,
    );
  });
});

