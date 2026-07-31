export const RESET_ALL_TRAFFIC_CONFIRMATION = "RESET ALL TRAFFIC";

export function canSubmitTrafficReset(
  scope: "all" | "users",
  confirmation: string,
  userIds: number[]
) {
  if (scope === "all") {
    return confirmation.trim() === RESET_ALL_TRAFFIC_CONFIRMATION;
  }
  return userIds.some((id) => Number.isSafeInteger(id) && id > 0);
}
