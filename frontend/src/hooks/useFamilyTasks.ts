import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from "@tanstack/react-query";
import { z } from "zod";
import {
  CompleteFamilyTaskResponseSchema,
  DeleteFamilyTaskResponseSchema,
  FamilyTaskSchema,
  type CompleteFamilyTaskResponse,
  type DeleteFamilyTaskResponse,
  type FamilyTask,
} from "../types/familyTask";

export const FAMILY_TASKS_QUERY_KEY = ["familyTasks"];
export const FAMILY_TASKS_REFRESH_WITH_DATA_MS = 60 * 60 * 1000;
export const FAMILY_TASKS_REFRESH_EMPTY_MS = 15 * 60 * 1000;

export function getFamilyTasksRefetchInterval(data?: FamilyTask[]): number {
  if (data && data.length > 0) {
    return FAMILY_TASKS_REFRESH_WITH_DATA_MS;
  }
  return FAMILY_TASKS_REFRESH_EMPTY_MS;
}

const fetchFamilyTasks = async (): Promise<FamilyTask[]> => {
  const response = await fetch("/api/family/tasks");
  if (!response.ok) {
    throw new Error("Failed to fetch family tasks");
  }
  const data = await response.json();
  return z.array(FamilyTaskSchema).parse(data);
};

const postCompleteTask = async (
  taskId: string,
  completer: string,
): Promise<CompleteFamilyTaskResponse> => {
  const response = await fetch("/api/family/tasks/complete", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ taskId, completer }),
  });
  if (!response.ok) {
    throw new Error("Failed to complete task");
  }
  const data = await response.json();
  return CompleteFamilyTaskResponseSchema.parse(data);
};

const postDeleteTask = async (taskId: string): Promise<DeleteFamilyTaskResponse> => {
  const response = await fetch("/api/family/tasks/delete", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ taskId }),
  });
  if (!response.ok) {
    throw new Error("Failed to delete task");
  }
  const data = await response.json();
  return DeleteFamilyTaskResponseSchema.parse(data);
};

export function useFamilyTasks(): UseQueryResult<FamilyTask[], Error> {
  return useQuery({
    queryKey: FAMILY_TASKS_QUERY_KEY,
    queryFn: fetchFamilyTasks,
    refetchInterval: (query) => getFamilyTasksRefetchInterval(query.state.data),
    refetchIntervalInBackground: true,
  });
}

export function useCompleteFamilyTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      taskId,
      completer,
    }: {
      taskId: string;
      completer: string;
    }) => postCompleteTask(taskId, completer),
    onSuccess: (data) => {
      queryClient.setQueryData(
        FAMILY_TASKS_QUERY_KEY,
        (oldData: FamilyTask[] | undefined) =>
          oldData?.map((task) =>
            task.id === data.completedTaskId
              ? {
                  ...task,
                  completed: true,
                  completedBy: data.completer,
                }
              : task,
          ) ?? [],
      );
    },
  });
}

export function useDeleteFamilyTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ taskId }: { taskId: string }) => postDeleteTask(taskId),
    onSuccess: (data) => {
      queryClient.setQueryData(
        FAMILY_TASKS_QUERY_KEY,
        (oldData: FamilyTask[] | undefined) =>
          oldData?.filter((task) => task.id !== data.deletedTaskId) ?? [],
      );
    },
  });
}
