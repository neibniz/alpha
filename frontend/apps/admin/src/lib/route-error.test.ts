import { expect, test } from "vitest";
import { classifyRouteError } from "./route-error";

test.each([
  [
    new TypeError(
      "Failed to fetch dynamically imported module: https://example.com/chunk.js"
    ),
    "resource",
  ],
  [new Error("Unable to preload CSS for /static/page.css"), "resource"],
  [new TypeError("Importing a module script failed."), "resource"],
  [{ response: { status: 503 } }, "server"],
  [{ response: { status: 401 } }, "session"],
  [new TypeError("Failed to fetch"), "network"],
  [{ code: "ERR_NETWORK" }, "network"],
  [
    new TypeError("Cannot read properties of null (reading 'site_name')"),
    "application",
  ],
  [
    new Error("Rendered more hooks than during the previous render."),
    "application",
  ],
  [null, "application"],
])("classifies %j as %s without exposing response contents", (error, expected) => {
  expect(classifyRouteError(error)).toBe(expected);
});
