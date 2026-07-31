import type { AccessPreset } from "@/lib/models";

const PRESET_QR_COLORS = [
  "#1e3a8a",
  "#14532d",
  "#7f1d1d",
  "#4c1d95",
  "#164e63",
  "#831843",
  "#713f12",
  "#312e81",
  "#3f6212",
  "#111827",
] as const;

export function findDefaultPreset(presets: AccessPreset[]) {
  return presets.find((preset) => preset.isDefault && preset.enabled);
}

export function presetKeyForSelection(
  selectedPresetId: number | undefined,
  selectedPreset: AccessPreset | undefined
) {
  return selectedPresetId === undefined ? undefined : selectedPreset?.key;
}

export function presetQrColor(presets: AccessPreset[], presetId: number) {
  const index = presets.findIndex((preset) => preset.id === presetId);
  return PRESET_QR_COLORS[Math.max(0, index) % PRESET_QR_COLORS.length];
}
