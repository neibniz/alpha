export const settingsSections = [
  "account",
  "security",
  "wallet",
  "orders",
  "support",
  "resources",
] as const;

export type SettingsSection = (typeof settingsSections)[number];
