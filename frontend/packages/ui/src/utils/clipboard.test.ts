import { afterEach, describe, expect, it, vi } from "vitest";
import { copyTextToClipboard } from "./clipboard";

describe("copyTextToClipboard", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("uses the legacy copy command when a secure clipboard is unavailable", async () => {
    const copy = vi.fn(() => true);
    const { document, wasRemoved } = clipboardDocument(copy);
    const restore = installClipboardGlobals(document);

    try {
      await expect(copyTextToClipboard("alpha-link")).resolves.toBe(true);
      expect(copy).toHaveBeenCalledWith("copy");
      expect(wasRemoved()).toBe(true);
    } finally {
      restore();
    }
  });

  it("reports a failed fallback without leaving temporary elements", async () => {
    const { document, wasRemoved } = clipboardDocument(vi.fn(() => false));
    const restore = installClipboardGlobals(document);

    try {
      await expect(copyTextToClipboard("alpha-link")).resolves.toBe(false);
      expect(wasRemoved()).toBe(true);
    } finally {
      restore();
    }
  });
});

function clipboardDocument(execCommand: (command: string) => boolean) {
  let removed = false;
  const textarea = {
    value: "",
    style: {},
    setAttribute: vi.fn(),
    select: vi.fn(),
    setSelectionRange: vi.fn(),
    remove: () => {
      removed = true;
    },
  };
  return {
    document: {
      body: { append: vi.fn() },
      createElement: vi.fn(() => textarea),
      execCommand,
      getSelection: vi.fn(() => null),
    },
    wasRemoved: () => removed,
  };
}

function installClipboardGlobals(documentValue: unknown) {
  const windowDescriptor = Object.getOwnPropertyDescriptor(
    globalThis,
    "window"
  );
  const documentDescriptor = Object.getOwnPropertyDescriptor(
    globalThis,
    "document"
  );
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: { isSecureContext: false },
  });
  Object.defineProperty(globalThis, "document", {
    configurable: true,
    value: documentValue,
  });
  return () => {
    restoreGlobal("window", windowDescriptor);
    restoreGlobal("document", documentDescriptor);
  };
}

function restoreGlobal(
  key: "window" | "document",
  descriptor: PropertyDescriptor | undefined
) {
  if (descriptor) {
    Object.defineProperty(globalThis, key, descriptor);
  } else {
    Reflect.deleteProperty(globalThis, key);
  }
}
