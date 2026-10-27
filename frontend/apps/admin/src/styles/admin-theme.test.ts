import { readFileSync } from "node:fs";
import { describe, expect, test } from "vitest";

const css = readFileSync(new URL("./admin.css", import.meta.url), "utf8");

function luminance(hex: string) {
  const channels = hex.match(/[a-f\d]{2}/gi)?.map((value) => {
    const channel = Number.parseInt(value, 16) / 255;
    return channel <= 0.040_45
      ? channel / 12.92
      : ((channel + 0.055) / 1.055) ** 2.4;
  });
  if (channels?.length !== 3) throw new Error(`Invalid color: ${hex}`);
  return channels[0]! * 0.2126 + channels[1]! * 0.7152 + channels[2]! * 0.0722;
}

function contrast(a: string, b: string) {
  const values = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (values[0]! + 0.05) / (values[1]! + 0.05);
}

describe.each(["", ".dark"])("Admin theme %s", (mode) => {
  const selector = `html[data-app="admin"]${mode} {`;
  const block = css
    .slice(css.indexOf(selector) + selector.length)
    .split("}")[0]!;
  const tokens = Object.fromEntries(
    [...block.matchAll(/--([\w-]+): (#[a-f\d]{6});/gi)].map((match) => [
      match[1],
      match[2],
    ])
  );

  test("keeps text readable on semantic surfaces", () => {
    for (const [foreground, background] of [
      ["foreground", "background"],
      ["card-foreground", "card"],
      ["primary-foreground", "primary"],
      ["muted-foreground", "muted"],
      ["accent-foreground", "accent"],
    ]) {
      expect(
        contrast(tokens[foreground!]!, tokens[background!]!)
      ).toBeGreaterThanOrEqual(4.5);
    }
  });

  test("keeps chart series visible against chart surfaces", () => {
    for (let series = 1; series <= 5; series++) {
      expect(
        contrast(tokens[`chart-${series}`]!, tokens.card!)
      ).toBeGreaterThanOrEqual(3);
    }
  });
});
