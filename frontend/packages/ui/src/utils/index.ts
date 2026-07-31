export const isBrowser = () => typeof window !== "undefined";

export { copyTextToClipboard } from "./clipboard";

export function isEmailDomainAllowed(email: string, allowedDomains: string[]) {
  const domain = email.split("@")[1]?.trim().toLowerCase();
  return Boolean(domain && allowedDomains.includes(domain));
}

type AppSubLinkValue =
  | string
  | number
  | boolean
  | null
  | AppSubLinkValue[]
  | {
      [key: string]: AppSubLinkValue;
    };

interface AppSubLinkContext {
  url: string;
  name: string;
}

/**
 * Extracts the full domain or root domain from a URL.
 *
 * @param url - The URL to extract the domain from.
 * @param extractRoot - If true, extracts the root domain (e.g., example.com). If false, extracts the full domain (e.g., sub.example.com).
 * @returns The extracted domain or root domain, or null if the URL is invalid.
 */
export function extractDomain(url: string, extractRoot = true): string | null {
  try {
    const { hostname } = new URL(url);
    if (/^\d{1,3}(\.\d{1,3}){3}$/.test(hostname)) {
      return hostname;
    }
    const domainParts = hostname.split(".").filter(Boolean);
    if (extractRoot && domainParts.length > 2) {
      return domainParts.slice(-2).join(".");
    }
    return hostname;
  } catch (error) {
    console.error("Invalid URL:", error);
    return null;
  }
}

export function renderAppSubLink(
  schema: string | undefined,
  context: AppSubLinkContext
) {
  if (!schema) return "url";
  try {
    let result = schema
      .replace(/\${url}/g, context.url)
      .replace(/\${name}/g, context.name);

    const maxLoop = 10;
    let prev: string;
    let loop = 0;
    do {
      prev = result;
      result = result.replace(
        /\${encodeURIComponent\(JSON\.stringify\(([^)]+)\)\)}/g,
        (match, expr) => {
          const value = resolveAppSubLinkJsonValue(expr, context);
          if (value === undefined) {
            return match;
          }
          return encodeURIComponent(JSON.stringify(value));
        }
      );

      result = result.replace(
        /\${encodeURIComponent\(([^)]+)\)}/g,
        (match, expr) => {
          const value = resolveAppSubLinkStringValue(expr, context);
          return value === undefined ? match : encodeURIComponent(value);
        }
      );

      result = result.replace(/\${window\.btoa\(([^)]+)\)}/g, (match, expr) => {
        const value = resolveAppSubLinkStringValue(expr, context);
        if (value === undefined) {
          return match;
        }
        const btoa = isBrowser() ? window.btoa : (str: string) => str;
        try {
          return btoa(value);
        } catch {
          return match;
        }
      });

      result = result.replace(
        /\${JSON\.stringify\(([^)]+)\)}/g,
        (match, expr) => {
          const value = resolveAppSubLinkJsonValue(expr, context);
          return value === undefined ? match : JSON.stringify(value);
        }
      );
      loop++;
    } while (result !== prev && loop < maxLoop);
    return result;
  } catch (_error) {
    return "";
  }
}

function resolveAppSubLinkStringValue(
  expr: string,
  context: AppSubLinkContext
): string | undefined {
  const trimmed = expr.trim();
  if (trimmed === "url") return context.url;
  if (trimmed === "name") return context.name;
  if (isQuotedString(trimmed)) {
    return trimmed.slice(1, -1);
  }
  if (/^[\w\s:/.?&=#,+%-]+$/.test(trimmed)) {
    return trimmed;
  }
  return;
}

function resolveAppSubLinkJsonValue(
  expr: string,
  context: AppSubLinkContext
): AppSubLinkValue | undefined {
  const trimmed = expr.trim();
  if (trimmed === "url" || trimmed === "name" || isQuotedString(trimmed)) {
    return resolveAppSubLinkStringValue(trimmed, context);
  }
  if (trimmed.includes("server_remote")) {
    return { server_remote: [`${context.url}, tag=${context.name}`] };
  }
  return parseSafeAppSubLinkObject(trimmed, context);
}

function parseSafeAppSubLinkObject(
  expr: string,
  context: AppSubLinkContext
): AppSubLinkValue | undefined {
  const jsonText = expr
    .replace(/([{,]\s*)([A-Za-z_][\w-]*)(\s*:)/g, '$1"$2"$3')
    .replace(/'([^'\\]*(?:\\.[^'\\]*)*)'/g, (_match, value: string) =>
      JSON.stringify(value)
    );
  const resolvedJsonText = replaceBareIdentifiers(jsonText, [
    ["url", JSON.stringify(context.url)],
    ["name", JSON.stringify(context.name)],
  ]);
  try {
    return JSON.parse(resolvedJsonText) as AppSubLinkValue;
  } catch {
    return;
  }
}

function replaceBareIdentifiers(
  source: string,
  replacements: ReadonlyArray<
    readonly [identifier: string, replacement: string]
  >
) {
  let result = "";
  let index = 0;
  let quote: '"' | null = null;
  scan: while (index < source.length) {
    const char = source[index];
    if (quote) {
      result += char;
      if (char === "\\" && index + 1 < source.length) {
        result += source[index + 1];
        index += 2;
        continue;
      }
      if (char === quote) {
        quote = null;
      }
      index++;
      continue;
    }

    if (char === '"') {
      quote = char;
      result += char;
      index++;
      continue;
    }

    for (const [identifier, replacement] of replacements) {
      if (
        source.startsWith(identifier, index) &&
        !isIdentifierChar(source[index - 1]) &&
        !isIdentifierChar(source[index + identifier.length])
      ) {
        result += replacement;
        index += identifier.length;
        continue scan;
      }
    }

    result += char;
    index++;
  }
  return result;
}

function isIdentifierChar(value: string | undefined) {
  return !!value && /[A-Za-z0-9_$]/.test(value);
}

function isQuotedString(value: string) {
  return (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  );
}
