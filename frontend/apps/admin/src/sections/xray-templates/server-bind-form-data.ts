export type TemplateBindingKeyed = {
  template_id: number;
};

export type SortableTemplateBinding = TemplateBindingKeyed & {
  enabled?: boolean;
  sort: number;
};

type JsonObject = Record<string, unknown>;

type RenderedTemplateType =
  | "inbound"
  | "outbound"
  | "dns"
  | "routing"
  | "geodata";

export type RenderedTemplateLike = {
  type: RenderedTemplateType | string;
  config?: JsonObject;
  error?: string;
};

export type GroupedRenderedTemplates<T extends RenderedTemplateLike> = {
  inbounds: JsonObject[];
  outbounds: JsonObject[];
  dnsItems: JsonObject[];
  routingItems: JsonObject[];
  geodataItems: JsonObject[];
  errors: T[];
};

export function buildBindingByTemplateId<T extends TemplateBindingKeyed>(
  bindings: T[]
): Map<number, T> {
  const result = new Map<number, T>();
  for (const binding of bindings) {
    result.set(binding.template_id, binding);
  }
  return result;
}

export function findBindingIndexByTemplateId<T extends TemplateBindingKeyed>(
  bindings: T[],
  templateId: number
): number {
  for (let index = 0; index < bindings.length; index += 1) {
    if (bindings[index]?.template_id === templateId) {
      return index;
    }
  }
  return -1;
}

export function selectEnabledTemplateBindings<
  TBinding extends SortableTemplateBinding,
  TTemplate,
>(
  bindings: TBinding[],
  templateById: ReadonlyMap<number, TTemplate>
): Array<{ binding: TBinding; template: TTemplate }> {
  const selected: Array<{ binding: TBinding; template: TTemplate }> = [];
  for (const binding of bindings) {
    if (!binding.enabled) {
      continue;
    }
    const template = templateById.get(binding.template_id);
    if (template) {
      selected.push({ binding, template });
    }
  }
  selected.sort((a, b) => a.binding.sort - b.binding.sort);
  return selected;
}

export function groupRenderedTemplates<T extends RenderedTemplateLike>(
  rendered: T[]
): GroupedRenderedTemplates<T> {
  const groups: GroupedRenderedTemplates<T> = {
    inbounds: [],
    outbounds: [],
    dnsItems: [],
    routingItems: [],
    geodataItems: [],
    errors: [],
  };

  for (const item of rendered) {
    if (item.error) {
      groups.errors.push(item);
    }
    if (!item.config) {
      continue;
    }
    switch (item.type) {
      case "inbound":
        groups.inbounds.push(item.config);
        break;
      case "outbound":
        groups.outbounds.push(item.config);
        break;
      case "dns":
        groups.dnsItems.push(item.config);
        break;
      case "routing":
        groups.routingItems.push(item.config);
        break;
      case "geodata":
        groups.geodataItems.push(item.config);
        break;
    }
  }

  return groups;
}
