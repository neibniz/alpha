import { describe, expect, it } from "vitest";
import {
  buildBindingByTemplateId,
  findBindingIndexByTemplateId,
  groupRenderedTemplates,
  selectEnabledTemplateBindings,
} from "./server-bind-form-data";

describe("server bind form data helpers", () => {
  it("indexes bindings by template id and keeps latest row", () => {
    const first = { template_id: 7, alias: "first" };
    const second = { template_id: 8, alias: "second" };
    const updated = { template_id: 7, alias: "updated" };

    const result = buildBindingByTemplateId([first, second, updated]);

    expect([...result.entries()]).toEqual([
      [7, updated],
      [8, second],
    ]);
    expect(result.get(7)).toBe(updated);
  });

  it("keeps an empty map shape for empty bindings", () => {
    const result = buildBindingByTemplateId([]);

    expect(result).toBeInstanceOf(Map);
    expect(result.size).toBe(0);
  });

  it("finds binding indexes without allocating a map for one-off updates", () => {
    const bindings = [
      { template_id: 4, alias: "dns" },
      { template_id: 7, alias: "inbound" },
    ];

    expect(findBindingIndexByTemplateId(bindings, 7)).toBe(1);
    expect(findBindingIndexByTemplateId(bindings, 9)).toBe(-1);
    expect(findBindingIndexByTemplateId([], 7)).toBe(-1);
  });

  it("selects enabled template bindings through the template index before sorting", () => {
    const bindings = [
      { template_id: 1, enabled: true, sort: 20, alias: "late" },
      { template_id: 2, enabled: false, sort: 1, alias: "disabled" },
      { template_id: 3, enabled: true, sort: 5, alias: "early" },
      { template_id: 4, enabled: true, sort: 0, alias: "missing-template" },
    ];
    const templates = new Map([
      [1, { id: 1, name: "Late" }],
      [3, { id: 3, name: "Early" }],
    ]);

    const result = selectEnabledTemplateBindings(bindings, templates);

    expect(result.map((item) => item.binding.alias)).toEqual(["early", "late"]);
    expect(result.map((item) => item.template.name)).toEqual(["Early", "Late"]);
  });

  it("groups rendered template configs and errors in one pass", () => {
    const inbound = { tag: "in-a" };
    const outbound = { tag: "out-a" };
    const dns = { servers: ["1.1.1.1"] };
    const routing = { rules: [{ outboundTag: "out-a" }] };
    const geodata = { geoip: "geoip.dat" };
    const broken = {
      id: 6,
      type: "routing",
      error: "JSON parse error",
    };

    const result = groupRenderedTemplates([
      { id: 1, type: "inbound", config: inbound },
      { id: 2, type: "outbound", config: outbound },
      { id: 3, type: "dns", config: dns },
      { id: 4, type: "routing", config: routing },
      { id: 5, type: "geodata", config: geodata },
      broken,
      { id: 7, type: "unknown", config: { ignored: true } },
    ]);

    expect(result.inbounds).toEqual([inbound]);
    expect(result.outbounds).toEqual([outbound]);
    expect(result.dnsItems).toEqual([dns]);
    expect(result.routingItems).toEqual([routing]);
    expect(result.geodataItems).toEqual([geodata]);
    expect(result.errors).toEqual([broken]);
  });
});
