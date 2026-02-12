import { z } from "zod";

export const ElectricityPriceSchema = z.object({
  DateTime: z.string(),
  Price: z.number(),
});

export type ElectricityPrice = z.infer<typeof ElectricityPriceSchema>;
export type ElectricityPriceResponse = ElectricityPrice[];
