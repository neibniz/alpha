// @ts-nocheck
import { describe, expect, test } from "bun:test";
import { readdirSync, readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

type Contract = {
  key: string;
  callKey?: string;
  requestInputTypes?: string[];
  requestInputInlineFields?: string[];
  requestType?: string;
  requestSource?: RequestSource;
  hasInlineRequestInput?: boolean;
  type: string;
  file: string;
};

type RequestSource = "none" | "params" | "data" | "form";
type ApiFieldSource = "json" | "form" | "path";
type ApiField = {
  name: string;
  source: ApiFieldSource;
  type: string;
  optional: boolean;
  required: boolean;
};
type ApiStruct = {
  file: string;
  fields: ApiField[];
  embeds: string[];
};
type TypeScriptType = {
  bases: string[];
  fields: string[];
  fieldTypes: Map<string, string>;
  fieldOptional: Map<string, boolean>;
};

const testDir = dirname(fileURLToPath(import.meta.url));
const workspaceRoot = join(testDir, "../../../../../");
const apiRoot = join(workspaceRoot, "server/apis");
const serviceRoot = join(workspaceRoot, "frontend/packages/ui/src/services");

describe("frontend service API contracts", () => {
  test("covers every browser-facing backend route", () => {
    const apiRoutes = parseApiRoutes();
    const serviceContracts = parseServiceContracts();
    const serviceKeys = new Set(
      serviceContracts.map((contract) => contract.callKey || contract.key)
    );
    const browserPrefixes = [
      "/v1/admin/",
      "/v1/auth/",
      "/v1/common/",
      "/v1/public/",
    ];
    const missing = [...apiRoutes.keys()].filter((key) => {
      const browserFacing = browserPrefixes.some((prefix) =>
        key.includes(` ${prefix}`)
      );
      return browserFacing && !serviceKeys.has(key);
    });

    expect(missing).toEqual([]);
  });

  test("matches backend API return data shapes", () => {
    const apiRoutes = parseApiRoutes();
    const serviceContracts = parseServiceContracts();
    const errors: string[] = [];

    expect(serviceContracts.length).toBeGreaterThan(0);
    for (const contract of serviceContracts) {
      const lookupKey = contract.callKey || contract.key;
      if (contract.callKey && contract.callKey !== contract.key) {
        errors.push(
          `${contract.file}: service comment ${contract.key} does not match request call ${contract.callKey}`
        );
      }

      const apiRoute = apiRoutes.get(lookupKey);
      if (!apiRoute) {
        errors.push(`${contract.file}: ${lookupKey} is not defined in .api`);
        continue;
      }

      if (apiRoute.type === "unknown") {
        if (contract.type !== "never") {
          errors.push(
            `${contract.file}: ${lookupKey} should declare data?: never because .api has no returns (...), got ${contract.type}`
          );
        }
        continue;
      }

      if (contract.type === "unknown" || contract.type === "never") {
        errors.push(
          `${contract.file}: ${lookupKey} should return API.${apiRoute.type}, not ${contract.type}`
        );
        continue;
      }

      if (contract.type !== apiRoute.type) {
        errors.push(
          `${contract.file}: ${lookupKey} returns API.${contract.type}, want API.${apiRoute.type} from ${apiRoute.file}`
        );
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API request payload shapes", () => {
    const apiRoutes = parseApiRoutes();
    const serviceContracts = parseServiceContracts();
    const errors: string[] = [];

    expect(serviceContracts.length).toBeGreaterThan(0);
    for (const contract of serviceContracts) {
      const lookupKey = contract.callKey || contract.key;
      const apiRoute = apiRoutes.get(lookupKey);
      if (!apiRoute) {
        continue;
      }
      if (contract.requestSource !== apiRoute.requestSource) {
        errors.push(
          `${contract.file}: ${lookupKey} sends request ${contract.requestSource}, want ${apiRoute.requestSource} from ${apiRoute.file}`
        );
      }
    }

    expect(errors).toEqual([]);
  });

  test("uses generated API request types for body payloads", () => {
    const apiRoutes = parseApiRoutes();
    const serviceContracts = parseServiceContracts();
    const errors: string[] = [];

    expect(serviceContracts.length).toBeGreaterThan(0);
    for (const contract of serviceContracts) {
      const lookupKey = contract.callKey || contract.key;
      const apiRoute = apiRoutes.get(lookupKey);
      if (
        !(
          apiRoute?.requestType &&
          ["data", "form"].includes(apiRoute.requestSource || "none")
        )
      ) {
        continue;
      }

      if (contract.hasInlineRequestInput) {
        errors.push(
          `${contract.file}: ${lookupKey} uses an inline request body; use API.${apiRoute.requestType}`
        );
        continue;
      }
      if (!contract.requestInputTypes?.includes(apiRoute.requestType)) {
        errors.push(
          `${contract.file}: ${lookupKey} request body uses ${contract.requestInputTypes?.join(", ") || "no API type"}, want API.${apiRoute.requestType}`
        );
      }
    }

    expect(errors).toEqual([]);
  });

  test("uses generated API request types for service inputs", () => {
    const apiRoutes = parseApiRoutes();
    const apiStructs = parseApiStructs();
    const serviceContracts = parseServiceContracts();
    const errors: string[] = [];

    expect(serviceContracts.length).toBeGreaterThan(0);
    for (const contract of serviceContracts) {
      const lookupKey = contract.callKey || contract.key;
      const apiRoute = apiRoutes.get(lookupKey);
      if (!apiRoute?.requestType) {
        continue;
      }

      const backendRequestFields = apiFieldNames(
        apiStructs,
        apiRoute.requestType,
        requestFieldSources(apiRoute.requestSource || "none")
      );
      if (backendRequestFields.length === 0) {
        continue;
      }

      if (contract.hasInlineRequestInput) {
        errors.push(
          `${contract.file}: ${lookupKey} uses an inline request input; use a shared API type`
        );
      }
      if (!contract.requestInputTypes?.length) {
        errors.push(
          `${contract.file}: ${lookupKey} has no API request input type`
        );
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API request and response field names", () => {
    const apiRoutes = parseApiRoutes();
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const serviceContracts = parseServiceContracts();
    const errors: string[] = [];

    expect(serviceContracts.length).toBeGreaterThan(0);
    for (const contract of serviceContracts) {
      const lookupKey = contract.callKey || contract.key;
      const apiRoute = apiRoutes.get(lookupKey);
      if (!apiRoute) {
        continue;
      }
      const scope = serviceScope(contract.file);
      const scopeTypes = typeScriptTypes.get(scope);
      if (!scopeTypes) {
        errors.push(
          `${contract.file}: missing typings.d.ts for service scope ${scope}`
        );
        continue;
      }

      if (
        apiRoute.type !== "unknown" &&
        contract.type !== "unknown" &&
        contract.type !== "never"
      ) {
        const backendFields = apiFieldNames(apiStructs, apiRoute.type, [
          "json",
        ]);
        const frontendFields = typeScriptFieldNames(scopeTypes, contract.type);
        compareFieldNames(
          `${contract.file}: ${lookupKey} response API.${contract.type}`,
          backendFields,
          frontendFields,
          errors
        );
      }

      if (!apiRoute.requestType) {
        continue;
      }

      const backendRequestFields = apiFieldNames(
        apiStructs,
        apiRoute.requestType,
        requestFieldSources(apiRoute.requestSource || "none")
      );
      if (backendRequestFields.length === 0) {
        continue;
      }

      const frontendRequestFields = serviceRequestFieldNames(
        scopeTypes,
        contract
      );
      compareFieldNames(
        `${contract.file}: ${lookupKey} request ${apiRoute.requestType}`,
        backendRequestFields,
        frontendRequestFields,
        errors
      );
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API scalar map field types", () => {
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const errors: string[] = [];

    for (const [typeName] of apiStructs) {
      const backendFields = apiFields(apiStructs, typeName, [
        "json",
        "form",
        "path",
      ]).filter((field) => isScalarRecordType(field.type));
      if (backendFields.length === 0) {
        continue;
      }

      for (const [scope, scopeTypes] of typeScriptTypes) {
        if (!scopeTypes.has(typeName)) {
          continue;
        }
        for (const field of backendFields) {
          const frontendType = typeScriptFieldType(
            scopeTypes,
            typeName,
            field.name
          );
          if (frontendType !== field.type) {
            errors.push(
              `${scope}/typings.d.ts: API.${typeName}.${field.name} is ${frontendType || "missing"}, want ${field.type}`
            );
          }
        }
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API struct map value field names", () => {
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const errors: string[] = [];

    for (const [typeName] of apiStructs) {
      const backendFields = apiFields(apiStructs, typeName, [
        "json",
        "form",
        "path",
      ]).filter((field) => {
        const record = parseRecordType(field.type);
        return Boolean(record && apiStructs.has(record.value));
      });
      if (backendFields.length === 0) {
        continue;
      }

      for (const [scope, scopeTypes] of typeScriptTypes) {
        if (!scopeTypes.has(typeName)) {
          continue;
        }
        for (const field of backendFields) {
          const record = parseRecordType(field.type);
          if (!record) {
            continue;
          }
          const backendValueFields = apiFieldNames(apiStructs, record.value, [
            "json",
          ]);
          const frontendValueFields = typeScriptRecordValueFieldNames(
            scopeTypes,
            typeName,
            field.name
          );
          compareFieldNames(
            `${scope}/typings.d.ts: API.${typeName}.${field.name} Record value`,
            backendValueFields,
            frontendValueFields,
            errors
          );
        }
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API scalar field types", () => {
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const errors: string[] = [];

    for (const [typeName] of apiStructs) {
      const backendFields = apiFields(apiStructs, typeName, [
        "json",
        "form",
        "path",
      ]).filter((field) => isScalarOrScalarArrayType(field.type));
      if (backendFields.length === 0) {
        continue;
      }

      for (const [scope, scopeTypes] of typeScriptTypes) {
        if (!scopeTypes.has(typeName)) {
          continue;
        }
        for (const field of backendFields) {
          const frontendType = typeScriptFieldType(
            scopeTypes,
            typeName,
            field.name
          );
          if (!isCompatibleScalarType(field.type, frontendType)) {
            errors.push(
              `${scope}/typings.d.ts: API.${typeName}.${field.name} is ${frontendType || "missing"}, want ${field.type}`
            );
          }
        }
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API raw JSON field types", () => {
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const errors: string[] = [];

    for (const [typeName] of apiStructs) {
      const backendFields = apiFields(apiStructs, typeName, [
        "json",
        "form",
        "path",
      ]).filter((field) => isRawJSONType(field.type));
      if (backendFields.length === 0) {
        continue;
      }

      for (const [scope, scopeTypes] of typeScriptTypes) {
        if (!scopeTypes.has(typeName)) {
          continue;
        }
        for (const field of backendFields) {
          const frontendType = typeScriptFieldType(
            scopeTypes,
            typeName,
            field.name
          );
          if (!isCompatibleRawJSONType(field.type, frontendType)) {
            errors.push(
              `${scope}/typings.d.ts: API.${typeName}.${field.name} is ${frontendType || "missing"}, want ${field.type}`
            );
          }
        }
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API struct array field types", () => {
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const errors: string[] = [];

    for (const [typeName] of apiStructs) {
      const backendFields = apiFields(apiStructs, typeName, [
        "json",
        "form",
        "path",
      ]).filter((field) => {
        const itemType = arrayItemType(field.type);
        return Boolean(itemType && apiStructs.has(itemType));
      });
      if (backendFields.length === 0) {
        continue;
      }

      for (const [scope, scopeTypes] of typeScriptTypes) {
        if (!scopeTypes.has(typeName)) {
          continue;
        }
        for (const field of backendFields) {
          const frontendType = typeScriptFieldType(
            scopeTypes,
            typeName,
            field.name
          );
          if (frontendType !== field.type) {
            errors.push(
              `${scope}/typings.d.ts: API.${typeName}.${field.name} is ${frontendType || "missing"}, want ${field.type}`
            );
          }
        }
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API request field optionality", () => {
    const apiRoutes = parseApiRoutes();
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const serviceContracts = parseServiceContracts();
    const errors: string[] = [];

    for (const contract of serviceContracts) {
      const lookupKey = contract.callKey || contract.key;
      const apiRoute = apiRoutes.get(lookupKey);
      if (!apiRoute?.requestType) {
        continue;
      }
      const scope = serviceScope(contract.file);
      const scopeTypes = typeScriptTypes.get(scope);
      if (!scopeTypes) {
        continue;
      }

      const backendFields = apiFields(
        apiStructs,
        apiRoute.requestType,
        requestFieldSources(apiRoute.requestSource || "none")
      ).filter((field) => field.required || field.optional);

      for (const field of backendFields) {
        const frontendOptional = serviceRequestFieldOptional(
          scopeTypes,
          contract,
          field.name
        );
        if (frontendOptional === undefined) {
          continue;
        }
        if (field.required && frontendOptional) {
          errors.push(
            `${contract.file}: ${lookupKey} request ${apiRoute.requestType}.${field.name} is optional in frontend, want required`
          );
        } else if (!field.required && field.optional && !frontendOptional) {
          errors.push(
            `${contract.file}: ${lookupKey} request ${apiRoute.requestType}.${field.name} is required in frontend, want optional`
          );
        }
      }
    }

    expect(errors).toEqual([]);
  });

  test("matches backend API omitempty response field optionality", () => {
    const apiStructs = parseApiStructs();
    const typeScriptTypes = parseScopedTypeScriptTypes();
    const errors: string[] = [];

    for (const [typeName] of apiStructs) {
      const backendFields = apiFields(apiStructs, typeName, ["json"]).filter(
        (field) => field.optional
      );
      if (backendFields.length === 0) {
        continue;
      }

      for (const [scope, scopeTypes] of typeScriptTypes) {
        if (!scopeTypes.has(typeName)) {
          continue;
        }
        for (const field of backendFields) {
          const frontendOptional = typeScriptFieldOptional(
            scopeTypes,
            typeName,
            field.name
          );
          if (frontendOptional === false) {
            errors.push(
              `${scope}/typings.d.ts: API.${typeName}.${field.name} is required in frontend, want optional because backend uses omitempty`
            );
          }
        }
      }
    }

    expect(errors).toEqual([]);
  });

  test("does not use any in generated API typings", () => {
    const errors: string[] = [];

    for (const file of walkFiles(serviceRoot, (path) =>
      path.endsWith("typings.d.ts")
    )) {
      const source = stripTypeScriptComments(readFileSync(file, "utf8"));
      const matches = [...source.matchAll(/\bany\b/g)];
      for (const match of matches) {
        const line = source.slice(0, match.index).split(/\r?\n/).length;
        errors.push(`${relative(workspaceRoot, file)}:${line}: uses any`);
      }
    }

    expect(errors).toEqual([]);
  });

  test("does not use unknown in generated API fields except the base response payload", () => {
    const errors: string[] = [];

    for (const file of walkFiles(serviceRoot, (path) =>
      path.endsWith("typings.d.ts")
    )) {
      const source = stripTypeScriptComments(readFileSync(file, "utf8"));
      const lines = source.split(/\r?\n/);
      lines.forEach((line, index) => {
        if (!/\bunknown\b/.test(line)) {
          return;
        }
        if (line.trim() === "data?: unknown;") {
          return;
        }
        errors.push(
          `${relative(workspaceRoot, file)}:${index + 1}: uses unknown`
        );
      });
    }

    expect(errors).toEqual([]);
  });
});

function parseApiRoutes() {
  const routes = new Map<string, Contract>();
  const requestSources = parseApiRequestSources();
  for (const file of walkFiles(apiRoot, (path) => path.endsWith(".api"))) {
    const lines = readFileSync(file, "utf8").split(/\r?\n/);
    let prefix = "";
    let inServerBlock = false;

    for (const line of lines) {
      if (line.includes("@server")) {
        inServerBlock = true;
        continue;
      }

      if (inServerBlock) {
        const prefixMatch = line.match(/prefix:\s*([^\s)]+)/);
        if (prefixMatch) {
          prefix = prefixMatch[1].replace(/^\/+|\/+$/g, "");
        }
        if (line.includes(")")) {
          inServerBlock = false;
        }
        continue;
      }

      const routeMatch = line
        .trim()
        .match(
          /^(get|post|put|delete|patch)\s+(\/\S*)(?:\s+\(([^)]+)\))?(?:\s+returns\s+\(([^)]+)\))?/i
        );
      if (!routeMatch) {
        continue;
      }

      const method = routeMatch[1].toUpperCase();
      const requestType = routeMatch[3] || "";
      const key = `${routeMatch[1].toUpperCase()} ${normalizeRoute(
        `/${prefix}${routeMatch[2]}`
      )}`;
      routes.set(key, {
        key,
        requestType,
        requestSource: apiRequestSource(
          method,
          requestSources.get(requestType) || "none"
        ),
        type: routeMatch[4] || "unknown",
        file: relative(workspaceRoot, file),
      });
    }
  }
  return routes;
}

function parseApiRequestSources() {
  const sources = new Map<string, RequestSource>();
  const structs = parseApiStructs();
  for (const [name] of structs) {
    sources.set(
      name,
      requestSourceFromTags(
        apiFields(structs, name, ["json", "form", "path"]).map(
          (field) => field.source
        )
      )
    );
  }
  return sources;
}

function parseApiStructs() {
  const structs = new Map<string, ApiStruct>();
  const structRe = /^\s*([A-Za-z0-9_]+)\s*\{/gm;
  for (const file of walkFiles(apiRoot, (path) => path.endsWith(".api"))) {
    const source = readFileSync(file, "utf8");
    let match = structRe.exec(source);
    while (match) {
      const bodyStart = structRe.lastIndex;
      const bodyEnd = matchingBraceEnd(source, bodyStart - 1);
      if (bodyEnd >= 0) {
        const body = source.slice(bodyStart, bodyEnd);
        structs.set(match[1], parseApiStruct(match[1], body, file));
      }
      match = structRe.exec(source);
    }
  }
  return structs;
}

function parseApiStruct(_name: string, body: string, file: string): ApiStruct {
  const fields: ApiField[] = [];
  const embeds: string[] = [];
  for (const rawLine of body.split(/\r?\n/)) {
    const line = rawLine.replace(/\/\/.*$/, "").trim();
    if (!line) {
      continue;
    }

    const embedMatch = line.match(/^([A-Za-z_][A-Za-z0-9_]*)$/);
    if (embedMatch) {
      embeds.push(embedMatch[1]);
      continue;
    }

    for (const tagMatch of line.matchAll(/(?:json|form|path):"([^"]+)"/g)) {
      const tagPrefix = tagMatch[0].split(":")[0] as ApiFieldSource;
      const tagName = tagMatch[1].split(",")[0];
      if (tagName && tagName !== "-") {
        fields.push({
          name: tagName,
          source: tagPrefix,
          type: normalizeApiFieldType(apiLineFieldType(line)),
          optional: apiTagIsOptional(tagMatch[1]),
          required: apiLineIsRequired(line),
        });
      }
    }
  }

  return {
    file: relative(workspaceRoot, file),
    fields,
    embeds,
  };
}

function apiLineFieldType(line: string) {
  const beforeTag = line.split("`")[0].trim();
  const match = beforeTag.match(/^[A-Za-z_][A-Za-z0-9_]*\s+(.+)$/);
  return match?.[1].trim() || "";
}

function apiTagIsOptional(tagValue: string) {
  return tagValue.split(",").slice(1).includes("omitempty");
}

function apiLineIsRequired(line: string) {
  const validate = line.match(/validate:"([^"]+)"/)?.[1] || "";
  return validate.split(",").includes("required");
}

function normalizeApiFieldType(typeName: string) {
  const compact = typeName.replace(/\s+/g, "");
  const mapMatch = compact.match(/^map\[([A-Za-z0-9_]+)\]([A-Za-z0-9_]+)$/);
  if (!mapMatch) {
    return normalizeApiNonMapFieldType(compact);
  }
  const keyType = normalizeGoScalarType(mapMatch[1]);
  const valueType = normalizeGoScalarType(mapMatch[2]) || mapMatch[2];
  if (!(keyType && valueType)) {
    return "";
  }
  return `Record<${keyType}, ${valueType}>`;
}

function normalizeApiNonMapFieldType(typeName: string): string {
  if (typeName.startsWith("*")) {
    return normalizeApiNonMapFieldType(typeName.slice(1));
  }
  if (typeName.startsWith("[]")) {
    const item = normalizeApiNonMapFieldType(typeName.slice(2));
    return item ? `${item}[]` : "";
  }
  if (typeName === "json.RawMessage") {
    return "JsonValue";
  }
  return normalizeGoScalarType(typeName) || typeName;
}

function normalizeGoScalarType(typeName: string) {
  if (typeName === "string") {
    return "string";
  }
  if (typeName === "bool") {
    return "boolean";
  }
  if (/^(?:u?int(?:8|16|32|64)?|float(?:32|64)?)$/.test(typeName)) {
    return "number";
  }
  return "";
}

function isScalarRecordType(typeName: string) {
  return /^Record<[^,]+, (?:string|number|boolean)>$/.test(typeName);
}

function isScalarOrScalarArrayType(typeName: string) {
  return /^(?:string|number|boolean)(?:\[\])?$/.test(typeName);
}

function isRawJSONType(typeName: string) {
  return typeName === "JsonValue" || typeName === "JsonValue[]";
}

function isCompatibleRawJSONType(backendType: string, frontendType: string) {
  const normalized = frontendType.replace(/^API\./, "");
  if (backendType === "JsonValue") {
    return normalized === "JsonValue" || normalized === "JsonObject";
  }
  return normalized === backendType;
}

function arrayItemType(typeName: string) {
  return typeName.endsWith("[]") ? typeName.slice(0, -2) : "";
}

function isCompatibleScalarType(backendType: string, frontendType: string) {
  if (backendType === frontendType) {
    return true;
  }
  if (backendType === "string" && isStringLiteralUnion(frontendType)) {
    return true;
  }
  return false;
}

function isStringLiteralUnion(typeName: string) {
  if (!typeName) {
    return false;
  }
  const members = typeName
    .split("|")
    .map((part) => part.trim())
    .filter(Boolean);
  return (
    members.length > 0 &&
    members.every((part) => /^"[^"]*"$/.test(part) || /^'[^']*'$/.test(part))
  );
}

function apiFieldNames(
  structs: Map<string, ApiStruct>,
  typeName: string,
  sources: ApiFieldSource[]
) {
  return uniqueSorted(
    apiFields(structs, typeName, sources).map((field) => field.name)
  );
}

function apiFields(
  structs: Map<string, ApiStruct>,
  typeName: string,
  sources: ApiFieldSource[],
  seen = new Set<string>()
): ApiField[] {
  if (seen.has(typeName)) {
    return [];
  }
  seen.add(typeName);
  const struct = structs.get(typeName);
  if (!struct) {
    return [];
  }

  return [
    ...struct.embeds.flatMap((embed) =>
      apiFields(structs, embed, sources, seen)
    ),
    ...struct.fields.filter((field) => sources.includes(field.source)),
  ];
}

function requestFieldSources(source: RequestSource): ApiFieldSource[] {
  if (source === "data") {
    return ["json"];
  }
  if (source === "params" || source === "form") {
    return ["form", "path"];
  }
  return ["json", "form", "path"];
}

function requestSourceFromTags(tags: string[]): RequestSource {
  if (tags.includes("json")) {
    return "data";
  }
  if (tags.includes("form")) {
    return "params";
  }
  if (tags.includes("path")) {
    return "params";
  }
  return "none";
}

function apiRequestSource(
  method: string,
  source: RequestSource
): RequestSource {
  if (source === "params" && method === "POST") {
    return "form";
  }
  return source;
}

function parseServiceContracts() {
  const contracts: Contract[] = [];
  for (const file of walkFiles(serviceRoot, (path) => path.endsWith(".ts"))) {
    const source = readFileSync(file, "utf8");
    const commentRe =
      /\/\*\*[\s\S]*?\b(GET|POST|PUT|DELETE|PATCH)\s+(\/v1\/[^\s*]+)[\s\S]*?\*\//g;
    let match = commentRe.exec(source);
    while (match) {
      const nextCommentIndex = source.indexOf("/**", commentRe.lastIndex);
      const block = source.slice(
        commentRe.lastIndex,
        nextCommentIndex < 0 ? source.length : nextCommentIndex
      );
      const type = extractRequestDataType(block);
      const requestCall = extractRequestCall(block);
      const requestInput = extractRequestInput(block);
      if (type) {
        contracts.push({
          key: `${match[1].toUpperCase()} ${match[2]}`,
          callKey: requestCall.path
            ? `${requestCall.method} ${requestCall.path}`
            : undefined,
          requestInputTypes: requestInput.apiTypes,
          requestInputInlineFields: requestInput.inlineFields,
          requestSource: requestCall.source,
          hasInlineRequestInput: requestInput.hasInlineInput,
          type,
          file: relative(workspaceRoot, file),
        });
      }
      match = commentRe.exec(source);
    }
  }
  return contracts;
}

function extractRequestDataType(block: string) {
  const marker = "return request<";
  const start = block.indexOf(marker);
  if (start < 0) {
    return "";
  }

  const genericEnd = findRequestGenericEnd(block, start + marker.length);
  if (genericEnd < 0) {
    return "";
  }
  const generic = block.slice(start + marker.length, genericEnd);
  const match = generic.match(
    /data\?\s*:\s*(?:API\.)?([A-Za-z0-9_]+|unknown|never)/
  );
  return match?.[1] || "";
}

function extractRequestInput(block: string) {
  const args = extractFunctionArgs(block)
    .replace(/\/\/.*$/gm, "")
    .replace(/\/\*[\s\S]*?\*\//g, "");
  const inlineFields = inlineRequestInputFields(args);
  return {
    apiTypes: [
      ...args.matchAll(/\b(?:body|params)\??\s*:\s*API\.([A-Za-z0-9_]+)/g),
    ].map((match) => match[1]),
    inlineFields,
    hasInlineInput: /\b(?:body|params)\??:\s*\{/.test(args),
  };
}

function inlineRequestInputFields(args: string) {
  const inlineMatch = args.match(/\b(?:body|params)\??:\s*\{/);
  if (!inlineMatch) {
    return [];
  }
  const objectStart = args.indexOf("{", inlineMatch.index);
  const objectEnd = matchingBraceEnd(args, objectStart);
  if (objectEnd < 0) {
    return [];
  }
  return parseTypeScriptObjectFields(args.slice(objectStart + 1, objectEnd));
}

function extractFunctionArgs(block: string) {
  const functionMatch = block.match(
    /export\s+async\s+function\s+[A-Za-z0-9_]+/
  );
  if (!functionMatch) {
    return "";
  }
  const start = block.indexOf("(", functionMatch.index);
  if (start < 0) {
    return "";
  }
  const end = matchingParenEnd(block, start);
  if (end < 0) {
    return "";
  }
  return block.slice(start + 1, end);
}

function extractRequestCall(block: string) {
  const marker = "return request<";
  const start = block.indexOf(marker);
  if (start < 0) {
    return { method: "", path: "" };
  }

  const genericEnd = findRequestGenericEnd(block, start + marker.length);
  if (genericEnd < 0) {
    return { method: "", path: "" };
  }

  const call = block.slice(genericEnd + 1);
  const pathMatch = call.match(/^\s*\(\s*["']([^"']+)["']/);
  const methodMatch = call.match(
    /method:\s*["'](GET|POST|PUT|DELETE|PATCH)["']/i
  );
  const source = serviceRequestSource(call);
  return {
    method: (methodMatch?.[1] || "GET").toUpperCase(),
    path: pathMatch?.[1] || "",
    source,
  };
}

function serviceRequestSource(call: string): RequestSource {
  if (/requestType:\s*["']form["']/.test(call)) {
    return "form";
  }
  if (/\bdata\s*:|\bdata\s*,/.test(call)) {
    return "data";
  }
  if (/\bparams\s*:|\bparams\s*,/.test(call)) {
    return "params";
  }
  return "none";
}

function findRequestGenericEnd(block: string, genericStart: number) {
  let depth = 1;
  for (let index = genericStart; index < block.length; index += 1) {
    const character = block[index];
    if (character === "<") {
      depth += 1;
    } else if (character === ">") {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }
  return -1;
}

function matchingParenEnd(source: string, openParen: number) {
  let depth = 0;
  for (let index = openParen; index < source.length; index += 1) {
    const character = source[index];
    if (character === "(") {
      depth += 1;
    } else if (character === ")") {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }
  return -1;
}

function matchingBraceEnd(source: string, openBrace: number) {
  let depth = 0;
  for (let index = openBrace; index < source.length; index += 1) {
    const character = source[index];
    if (character === "{") {
      depth += 1;
    } else if (character === "}") {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }
  return -1;
}

function parseScopedTypeScriptTypes() {
  const scopes = new Map<string, Map<string, TypeScriptType>>();
  for (const file of walkFiles(serviceRoot, (path) =>
    path.endsWith("typings.d.ts")
  )) {
    scopes.set(
      dirname(file).split(/[\\/]/).pop() || "",
      parseTypeScriptTypes(readFileSync(file, "utf8"))
    );
  }
  return scopes;
}

function parseTypeScriptTypes(source: string) {
  const types = new Map<string, TypeScriptType>();
  const typeRe = /\btype\s+([A-Za-z0-9_]+)\s*=/g;
  let match = typeRe.exec(source);
  while (match) {
    const declarationStart = typeRe.lastIndex;
    const declarationEnd = topLevelSemicolon(source, declarationStart);
    if (declarationEnd < 0) {
      match = typeRe.exec(source);
      continue;
    }
    const declaration = source.slice(declarationStart, declarationEnd);
    types.set(match[1], parseTypeScriptTypeDeclaration(declaration));
    typeRe.lastIndex = declarationEnd + 1;
    match = typeRe.exec(source);
  }
  return types;
}

function parseTypeScriptTypeDeclaration(declaration: string): TypeScriptType {
  const objectFields: string[] = [];
  const fieldTypes = new Map<string, string>();
  const fieldOptional = new Map<string, boolean>();
  for (const body of topLevelObjectBodies(declaration)) {
    for (const field of parseTypeScriptObjectFieldEntries(body)) {
      objectFields.push(field.name);
      fieldTypes.set(field.name, field.type);
      fieldOptional.set(field.name, field.optional);
    }
  }

  return {
    bases: declaration
      .split("&")
      .map((part) => part.trim())
      .filter((part) => !part.startsWith("{"))
      .map((part) => part.match(/^(?:API\.)?([A-Za-z0-9_]+)$/)?.[1] || "")
      .filter(Boolean),
    fields: uniqueSorted(objectFields),
    fieldTypes,
    fieldOptional,
  };
}

function topLevelObjectBodies(source: string) {
  const bodies: string[] = [];
  let angleDepth = 0;
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index];
    if (character === "<") {
      angleDepth += 1;
    } else if (character === ">") {
      angleDepth = Math.max(0, angleDepth - 1);
    } else if (character === "{" && angleDepth === 0) {
      const end = matchingBraceEnd(source, index);
      if (end >= 0) {
        bodies.push(source.slice(index + 1, end));
        index = end;
      }
    }
  }
  return bodies;
}

function parseTypeScriptObjectFields(body: string) {
  return uniqueSorted(
    parseTypeScriptObjectFieldEntries(body).map((field) => field.name)
  );
}

function parseTypeScriptObjectFieldEntries(body: string) {
  return uniqueSorted(
    splitTopLevelTypeMembers(body)
      .map((member) => {
        const match = stripTypeScriptComments(member).match(
          /^\s*(?:readonly\s+)?["']?([A-Za-z0-9_$-]+)["']?(\?)?\s*:\s*([\s\S]+)$/
        );
        if (!match) {
          return null;
        }
        return {
          name: match[1],
          optional: Boolean(match[2]),
          type: normalizeTypeScriptFieldType(match[3]),
        };
      })
      .filter(Boolean)
      .map((field) => `${field.name}\0${field.type}\0${field.optional}`)
  ).map((entry) => {
    const [name, type, optional] = entry.split("\0");
    return { name, type, optional: optional === "true" };
  });
}

function normalizeTypeScriptFieldType(typeName: string) {
  const collapsed = typeName.replace(/\s+/g, " ").trim();
  const record = parseRecordType(collapsed);
  if (!record) {
    return normalizeArrayType(collapsed);
  }
  return `Record<${record.key}, ${record.value}>`;
}

function normalizeArrayType(typeName: string) {
  const arrayMatch = typeName.match(
    /^Array\s*<\s*([A-Za-z_][A-Za-z0-9_]*)\s*>$/
  );
  if (arrayMatch) {
    return `${arrayMatch[1]}[]`;
  }
  return typeName;
}

function parseRecordType(typeName: string) {
  const source = typeName.trim();
  if (!source.startsWith("Record")) {
    return null;
  }
  const open = source.indexOf("<");
  if (open < 0) {
    return null;
  }
  const close = matchingAngleEnd(source, open);
  if (close !== source.length - 1) {
    return null;
  }
  const body = source.slice(open + 1, close);
  const comma = topLevelComma(body);
  if (comma < 0) {
    return null;
  }
  return {
    key: normalizeTypePart(body.slice(0, comma)),
    value: normalizeTypePart(body.slice(comma + 1)),
  };
}

function normalizeTypePart(value: string) {
  return value.replace(/\s+/g, " ").trim();
}

function typeScriptRecordValueFieldNames(
  scopeTypes: Map<string, TypeScriptType>,
  typeName: string,
  fieldName: string
) {
  const fieldType = typeScriptFieldType(scopeTypes, typeName, fieldName);
  const record = parseRecordType(fieldType);
  if (!record) {
    return [];
  }
  const valueType = record.value;
  if (valueType.startsWith("{")) {
    const end = matchingBraceEnd(valueType, 0);
    if (end === valueType.length - 1) {
      return parseTypeScriptObjectFields(valueType.slice(1, end));
    }
  }
  const namedType = valueType.match(
    /^(?:API\.)?([A-Za-z_][A-Za-z0-9_]*)$/
  )?.[1];
  if (!namedType) {
    return [];
  }
  return typeScriptFieldNames(scopeTypes, namedType);
}

function typeScriptFieldType(
  scopeTypes: Map<string, TypeScriptType>,
  typeName: string,
  fieldName: string,
  seen = new Set<string>()
) {
  if (seen.has(typeName)) {
    return "";
  }
  seen.add(typeName);
  const type = scopeTypes.get(typeName);
  if (!type) {
    return "";
  }
  const ownType = type.fieldTypes.get(fieldName);
  if (ownType) {
    return ownType;
  }
  for (const base of type.bases) {
    const baseType = typeScriptFieldType(scopeTypes, base, fieldName, seen);
    if (baseType) {
      return baseType;
    }
  }
  return "";
}

function serviceRequestFieldOptional(
  scopeTypes: Map<string, TypeScriptType>,
  contract: Contract,
  fieldName: string
) {
  for (const typeName of contract.requestInputTypes || []) {
    const optional = typeScriptFieldOptional(scopeTypes, typeName, fieldName);
    if (optional !== undefined) {
      return optional;
    }
  }
  return;
}

function typeScriptFieldOptional(
  scopeTypes: Map<string, TypeScriptType>,
  typeName: string,
  fieldName: string,
  seen = new Set<string>()
): boolean | undefined {
  if (seen.has(typeName)) {
    return;
  }
  seen.add(typeName);
  const type = scopeTypes.get(typeName);
  if (!type) {
    return;
  }
  if (type.fieldOptional.has(fieldName)) {
    return type.fieldOptional.get(fieldName);
  }
  for (const base of type.bases) {
    const baseOptional = typeScriptFieldOptional(
      scopeTypes,
      base,
      fieldName,
      seen
    );
    if (baseOptional !== undefined) {
      return baseOptional;
    }
  }
  return;
}

function splitTopLevelTypeMembers(body: string) {
  const members: string[] = [];
  let memberStart = 0;
  let braceDepth = 0;
  let bracketDepth = 0;
  let parenDepth = 0;
  let angleDepth = 0;
  for (let index = 0; index < body.length; index += 1) {
    const character = body[index];
    if (character === "{") {
      braceDepth += 1;
    } else if (character === "}") {
      braceDepth -= 1;
    } else if (character === "[") {
      bracketDepth += 1;
    } else if (character === "]") {
      bracketDepth -= 1;
    } else if (character === "(") {
      parenDepth += 1;
    } else if (character === ")") {
      parenDepth -= 1;
    } else if (character === "<") {
      angleDepth += 1;
    } else if (character === ">") {
      angleDepth = Math.max(0, angleDepth - 1);
    } else if (
      character === ";" &&
      braceDepth === 0 &&
      bracketDepth === 0 &&
      parenDepth === 0 &&
      angleDepth === 0
    ) {
      members.push(body.slice(memberStart, index));
      memberStart = index + 1;
    }
  }
  if (memberStart < body.length) {
    members.push(body.slice(memberStart));
  }
  return members;
}

function typeScriptFieldNames(
  scopeTypes: Map<string, TypeScriptType>,
  typeName: string,
  seen = new Set<string>()
) {
  if (seen.has(typeName)) {
    return [];
  }
  seen.add(typeName);
  const type = scopeTypes.get(typeName);
  if (!type) {
    return [];
  }
  return uniqueSorted([
    ...type.bases.flatMap((base) =>
      typeScriptFieldNames(scopeTypes, base, seen)
    ),
    ...type.fields,
  ]);
}

function serviceRequestFieldNames(
  scopeTypes: Map<string, TypeScriptType>,
  contract: Contract
) {
  return uniqueSorted([
    ...(contract.requestInputTypes || []).flatMap((typeName) =>
      typeScriptFieldNames(scopeTypes, typeName)
    ),
    ...(contract.requestInputInlineFields || []),
  ]);
}

function topLevelSemicolon(source: string, start: number) {
  let braceDepth = 0;
  let bracketDepth = 0;
  let parenDepth = 0;
  let angleDepth = 0;
  for (let index = start; index < source.length; index += 1) {
    const character = source[index];
    if (character === "{") {
      braceDepth += 1;
    } else if (character === "}") {
      braceDepth -= 1;
    } else if (character === "[") {
      bracketDepth += 1;
    } else if (character === "]") {
      bracketDepth -= 1;
    } else if (character === "(") {
      parenDepth += 1;
    } else if (character === ")") {
      parenDepth -= 1;
    } else if (character === "<") {
      angleDepth += 1;
    } else if (character === ">") {
      angleDepth = Math.max(0, angleDepth - 1);
    } else if (
      character === ";" &&
      braceDepth === 0 &&
      bracketDepth === 0 &&
      parenDepth === 0 &&
      angleDepth === 0
    ) {
      return index;
    }
  }
  return -1;
}

function topLevelComma(source: string) {
  let braceDepth = 0;
  let bracketDepth = 0;
  let parenDepth = 0;
  let angleDepth = 0;
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index];
    if (character === "{") {
      braceDepth += 1;
    } else if (character === "}") {
      braceDepth -= 1;
    } else if (character === "[") {
      bracketDepth += 1;
    } else if (character === "]") {
      bracketDepth -= 1;
    } else if (character === "(") {
      parenDepth += 1;
    } else if (character === ")") {
      parenDepth -= 1;
    } else if (character === "<") {
      angleDepth += 1;
    } else if (character === ">") {
      angleDepth = Math.max(0, angleDepth - 1);
    } else if (
      character === "," &&
      braceDepth === 0 &&
      bracketDepth === 0 &&
      parenDepth === 0 &&
      angleDepth === 0
    ) {
      return index;
    }
  }
  return -1;
}

function matchingAngleEnd(source: string, openAngle: number) {
  let depth = 0;
  for (let index = openAngle; index < source.length; index += 1) {
    const character = source[index];
    if (character === "<") {
      depth += 1;
    } else if (character === ">") {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }
  return -1;
}

function stripTypeScriptComments(source: string) {
  return source.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
}

function serviceScope(file: string) {
  return (
    file.match(/frontend\/packages\/ui\/src\/services\/([^/]+)/)?.[1] || ""
  );
}

function compareFieldNames(
  context: string,
  backendFields: string[],
  frontendFields: string[],
  errors: string[]
) {
  const missing = backendFields.filter(
    (field) => !frontendFields.includes(field)
  );
  const extra = frontendFields.filter(
    (field) => !backendFields.includes(field)
  );
  if (missing.length || extra.length) {
    errors.push(
      `${context} fields mismatch: missing [${missing.join(", ")}], extra [${extra.join(", ")}]`
    );
  }
}

function uniqueSorted(values: string[]) {
  return [...new Set(values)].sort();
}

function normalizeRoute(route: string) {
  return route.replace(/\/+/g, "/").replace(/^([^/])/, "/$1");
}

function walkFiles(root: string, predicate: (path: string) => boolean) {
  const files: string[] = [];
  walk(root, predicate, files);
  return files;
}

function walk(
  root: string,
  predicate: (path: string) => boolean,
  files: string[]
) {
  for (const entry of readdirSync(root, { withFileTypes: true })) {
    if (entry.name === "node_modules" || entry.name === ".git") {
      continue;
    }
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      walk(path, predicate, files);
    } else if (predicate(path)) {
      files.push(path);
    }
  }
}
