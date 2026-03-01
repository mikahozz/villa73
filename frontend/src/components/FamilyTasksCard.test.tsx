// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { FamilyTasksCard } from "./FamilyTasksCard";
import * as familyTasksHook from "../hooks/useFamilyTasks";

vi.mock("../hooks/useFamilyTasks");

const mockedUseFamilyTasks = vi.mocked(familyTasksHook.useFamilyTasks);
const mockedUseCompleteFamilyTask = vi.mocked(
  familyTasksHook.useCompleteFamilyTask,
);
const mockedUseDeleteFamilyTask = vi.mocked(familyTasksHook.useDeleteFamilyTask);

function renderComponent() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <FamilyTasksCard />
    </QueryClientProvider>,
  );
}

describe("FamilyTasksCard", () => {
  beforeEach(() => {
    mockedUseCompleteFamilyTask.mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue({
        completedTaskId: "task-1",
        completer: "Ella",
      }),
      isPending: false,
    } as unknown as ReturnType<typeof familyTasksHook.useCompleteFamilyTask>);
    mockedUseDeleteFamilyTask.mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue({
        deletedTaskId: "task-1",
      }),
      isPending: false,
    } as unknown as ReturnType<typeof familyTasksHook.useDeleteFamilyTask>);
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    cleanup();
  });

  test("does not render when there are no tasks", () => {
    mockedUseFamilyTasks.mockReturnValue({
      data: [],
      isLoading: false,
    } as unknown as ReturnType<typeof familyTasksHook.useFamilyTasks>);

    renderComponent();
    expect(screen.queryByTestId("family-task-card")).toBeNull();
  });

  test("renders oldest task first", () => {
    mockedUseFamilyTasks.mockReturnValue({
      data: [
        {
          id: "task-1",
          title: "First task",
          note: "tehtava: iskä",
          labels: ["2"],
          size: 2,
          assignee: "Iskä",
          createdAt: new Date().toISOString(),
        },
        {
          id: "task-2",
          title: "Second task",
          note: "ella, jotain",
          labels: ["1"],
          size: 1,
          assignee: "Ella",
          createdAt: new Date().toISOString(),
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof familyTasksHook.useFamilyTasks>);

    renderComponent();
    expect(screen.getByText("First task")).toBeDefined();
    expect(screen.getByTestId("carousel-indicators")).toBeDefined();
  });

  test("completes task and shows compact completed row", async () => {
    const mutateAsync = vi.fn().mockResolvedValue({
      completedTaskId: "task-1",
      completer: "Ella",
    });
    mockedUseCompleteFamilyTask.mockReturnValue({
      mutateAsync,
      isPending: false,
    } as unknown as ReturnType<typeof familyTasksHook.useCompleteFamilyTask>);
    mockedUseFamilyTasks.mockReturnValue({
      data: [
        {
          id: "task-1",
          title: "Task to complete",
          note: "ella, jotain",
          labels: ["3"],
          size: 3,
          assignee: "Ella",
          createdAt: new Date().toISOString(),
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof familyTasksHook.useFamilyTasks>);

    renderComponent();
    fireEvent.change(screen.getByTestId("completer-select"), {
      target: { value: "Elias" },
    });
    fireEvent.click(screen.getByTestId("complete-task-button"));

    expect(mutateAsync).toHaveBeenCalledWith({
      taskId: "task-1",
      completer: "Elias",
    });
    expect(await screen.findByTestId("task-completed")).toBeDefined();
  });

  test("prompts when task has no assignee and allows skipping completer", async () => {
    const mutateAsync = vi.fn().mockResolvedValue({
      completedTaskId: "task-1",
      completer: "Unknown",
    });
    mockedUseCompleteFamilyTask.mockReturnValue({
      mutateAsync,
      isPending: false,
    } as unknown as ReturnType<typeof familyTasksHook.useCompleteFamilyTask>);
    mockedUseFamilyTasks.mockReturnValue({
      data: [
        {
          id: "task-1",
          title: "Task without assignee",
          note: "",
          labels: ["1"],
          size: 1,
          createdAt: new Date().toISOString(),
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof familyTasksHook.useFamilyTasks>);

    renderComponent();
    expect(screen.queryByTestId("completer-select")).toBeNull();
    fireEvent.click(screen.getByTestId("complete-task-button"));
    const prompt = await screen.findByTestId("complete-prompt");
    fireEvent.click(within(prompt).getByRole("button", { name: "Complete task" }));

    expect(mutateAsync).toHaveBeenCalledWith({
      taskId: "task-1",
      completer: "Unknown",
    });
    expect(await screen.findByTestId("task-completed")).toBeDefined();
  });

  test("deletes task after confirmation", async () => {
    const mutateAsync = vi.fn().mockResolvedValue({
      deletedTaskId: "task-1",
    });
    mockedUseDeleteFamilyTask.mockReturnValue({
      mutateAsync,
      isPending: false,
    } as unknown as ReturnType<typeof familyTasksHook.useDeleteFamilyTask>);
    mockedUseFamilyTasks.mockReturnValue({
      data: [
        {
          id: "task-1",
          title: "Task to delete",
          note: "ella, jotain",
          labels: [],
          size: 0,
          assignee: "Ella",
          createdAt: new Date().toISOString(),
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof familyTasksHook.useFamilyTasks>);

    renderComponent();
    fireEvent.click(screen.getByRole("button", { name: "Delete task Task to delete" }));

    expect(window.confirm).toHaveBeenCalled();
    expect(mutateAsync).toHaveBeenCalledWith({ taskId: "task-1" });
  });
});
