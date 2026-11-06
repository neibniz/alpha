export const REVEAL_DURATION_MINUTES = [5, 15, 30] as const;
export type RevealDurationMinutes = (typeof REVEAL_DURATION_MINUTES)[number];
export const DEFAULT_REVEAL_DURATION_MINUTES: RevealDurationMinutes = 15;
