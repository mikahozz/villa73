import { z } from "zod";

export const FamilyTaskSchema = z.object({
  id: z.string(),
  title: z.string(),
  note: z.string().optional().default(""),
  labels: z.array(z.string()).optional().default([]),
  size: z.number().int().min(0).max(5),
  assignee: z.string().optional(),
  completed: z.boolean().optional().default(false),
  completedBy: z.string().optional(),
  createdAt: z.string(),
});

export type FamilyTask = z.infer<typeof FamilyTaskSchema>;

export const CompleteFamilyTaskResponseSchema = z.object({
  completedTaskId: z.string(),
  completer: z.string(),
});

export type CompleteFamilyTaskResponse = z.infer<
  typeof CompleteFamilyTaskResponseSchema
>;

export const DeleteFamilyTaskResponseSchema = z.object({
  deletedTaskId: z.string(),
});

export type DeleteFamilyTaskResponse = z.infer<
  typeof DeleteFamilyTaskResponseSchema
>;
