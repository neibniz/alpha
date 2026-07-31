import { describe, expect, test } from "bun:test";
import type { AccessPreset } from "@/lib/models";
import {
  findDefaultPreset,
  presetKeyForSelection,
  presetQrColor,
} from "@/lib/preset-utils";

function preset(id: number, isDefault = false): AccessPreset {
  return {
    enabled: true,
    id,
    isDefault,
    items: [],
    key: `ps_${id}`,
    mode: "custom",
    name: `Preset ${id}`,
    profileKey: "",
  };
}

describe("preset utilities", () => {
  test("finds the enabled default preset", () => {
    const disabledDefault = { ...preset(1, true), enabled: false };
    expect(findDefaultPreset([disabledDefault, preset(2, true)])?.id).toBe(2);
  });

  test("keeps the default subscription link free of a preset key", () => {
    const selected = preset(1, true);
    expect(presetKeyForSelection(undefined, selected)).toBe(undefined);
    expect(presetKeyForSelection(1, selected)).toBe("ps_1");
  });

  test("assigns a different QR color to all ten allowed presets", () => {
    const presets = Array.from({ length: 10 }, (_, index) => preset(index + 1));
    const colors = presets.map((item) => presetQrColor(presets, item.id));
    expect(new Set(colors).size).toBe(10);
  });
});
