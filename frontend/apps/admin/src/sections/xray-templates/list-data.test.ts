import { describe, expect, test } from "vitest";
import {
  buildXrayTemplateSummary,
  filterXrayTemplates,
  getXrayTemplateCapabilities,
  getXrayTemplateUseState,
  normalizeImportedXrayTemplates,
  xrayTemplateDetailToCreateRequest,
} from "./list-data";

function template(
  overrides: Partial<API.XrayTemplateListItem> = {}
): API.XrayTemplateListItem {
  return {
    id: 1,
    name: "VLESS Reality TCP",
    type: "inbound",
    description: "Production Reality inbound",
    enabled: true,
    binding_count: 2,
    subscription_binding_count: 1,
    updated_at: 1_700_000_000_000,
    ...overrides,
  };
}

describe("xray template list data", () => {
  test("summarizes active, unused, binding and subscription counts", () => {
    const result = buildXrayTemplateSummary([
      template(),
      template({ id: 2, binding_count: 0, subscription_binding_count: 0 }),
      template({ id: 3, enabled: false, binding_count: 1 }),
    ]);

    expect(result).toEqual({
      total: 3,
      enabled: 2,
      attention: 1,
      bindings: 3,
      subscriptions: 2,
    });
  });

  test("filters by search, type, state and binding status", () => {
    const templates = [
      template(),
      template({
        id: 2,
        name: "Default DNS",
        type: "dns",
        description: "DNS template",
        binding_count: 0,
      }),
      template({ id: 3, name: "Disabled", enabled: false }),
    ];

    expect(
      filterXrayTemplates(templates, {
        search: "dns",
        type: "dns",
        state: "enabled",
        binding: "unbound",
      }).map((item) => item.id)
    ).toEqual([2]);
  });

  test("derives compact capabilities and deterministic use state", () => {
    expect(getXrayTemplateCapabilities(template())).toEqual([
      "VLESS",
      "Reality",
      "TCP",
    ]);
    expect(getXrayTemplateUseState(template())).toBe("ready");
    expect(getXrayTemplateUseState(template({ binding_count: 0 }))).toBe(
      "unused"
    );
    expect(getXrayTemplateUseState(template({ enabled: false }))).toBe(
      "disabled"
    );
  });

  test("normalizes exported templates and defaults imports to disabled", () => {
    const [result] = normalizeImportedXrayTemplates({
      id: 7,
      name: " Imported ",
      type: "routing",
      config: { rules: [] },
      description: "Rules",
    });

    expect(result).toEqual({
      name: "Imported",
      type: "routing",
      config: { rules: [] },
      config_template: undefined,
      variables_schema: undefined,
      default_variables: undefined,
      subscription_meta: undefined,
      description: "Rules",
      enabled: false,
    });
  });

  test("rejects invalid import shapes", () => {
    expect(() =>
      normalizeImportedXrayTemplates({ name: "Bad", type: "other" })
    ).toThrow("类型无效");
    expect(() =>
      normalizeImportedXrayTemplates({ name: "Bad", type: "dns", config: [] })
    ).toThrow("config 必须是 JSON 对象");
  });

  test("creates a disabled duplicate payload without persistence fields", () => {
    const result = xrayTemplateDetailToCreateRequest(
      {
        id: 5,
        name: "DNS",
        type: "dns",
        config: { servers: ["1.1.1.1"] },
        description: "DNS template",
        enabled: true,
        created_at: 1,
        updated_at: 2,
      },
      { name: "DNS copy", enabled: false }
    );

    expect(result.name).toBe("DNS copy");
    expect(result.enabled).toBe(false);
    expect(result.config).toEqual({ servers: ["1.1.1.1"] });
    expect(result).not.toHaveProperty("id");
  });
});
