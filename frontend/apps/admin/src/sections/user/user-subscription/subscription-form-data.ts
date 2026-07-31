import * as z from "zod";

const byteValue = z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);

export const subscriptionFormSchema = z.object({
  subscribe_id: z.number().int().positive(),
  traffic: byteValue,
  expired_at: z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER),
  upload: byteValue,
  download: byteValue,
  id: z.number().int().positive().optional(),
});

export type SubscriptionFormValues = z.infer<typeof subscriptionFormSchema>;
