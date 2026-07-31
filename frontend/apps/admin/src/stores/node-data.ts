const SMALL_NODE_TAG_LIST_LIMIT = 16;
const SMALL_NODE_LOOKUP_LIMIT = 16;

export type CompactLookup<T> = {
  has: (value: T) => boolean;
};

export function buildNodeIndexes(nodes: API.Node[]) {
  const nodeById = new Map<number, API.Node>();
  const nodesByTag = new Map<string, API.Node[]>();
  const nodesWithoutTags: API.Node[] = [];
  const nodeTags: string[] = [];
  let globalTagSeen: Set<string> | undefined;
  const currentNodeTags: string[] = [];
  let currentNodeTagSeen: Set<string> | undefined;
  const inboundUsage = newCompactLookupBuilder<string>();
  const serverIds = newCompactLookupBuilder<number>();

  for (const node of nodes) {
    nodeById.set(node.id, node);
    serverIds.add(node.server_id);

    const inboundAlias = node.inbound_alias || node.protocol;
    if (inboundAlias) {
      inboundUsage.add(nodeInboundKey(node.server_id, inboundAlias));
    }

    const tags = Array.isArray(node.tags) ? node.tags : [];
    let hasTags = false;
    currentNodeTags.length = 0;
    currentNodeTagSeen?.clear();
    for (const tag of tags) {
      const normalizedTag = normalizeNodeTag(tag);
      if (!normalizedTag) {
        continue;
      }
      if (currentNodeTagSeen) {
        if (currentNodeTagSeen.has(normalizedTag)) {
          continue;
        }
        currentNodeTagSeen.add(normalizedTag);
      } else {
        if (currentNodeTags.includes(normalizedTag)) {
          continue;
        }
        currentNodeTags.push(normalizedTag);
        if (currentNodeTags.length > SMALL_NODE_TAG_LIST_LIMIT) {
          currentNodeTagSeen = new Set(currentNodeTags);
        }
      }
      hasTags = true;
      globalTagSeen = appendUniqueTag(nodeTags, normalizedTag, globalTagSeen);
      const group = nodesByTag.get(normalizedTag);
      if (group) {
        group.push(node);
      } else {
        nodesByTag.set(normalizedTag, [node]);
      }
    }

    if (!hasTags) {
      nodesWithoutTags.push(node);
    }
  }

  return {
    nodeById,
    nodesByTag,
    nodesWithoutTags,
    nodeTags,
    inboundUsage: inboundUsage.lookup(),
    serverIds: serverIds.lookup(),
  };
}

export function mergeAvailableNodeTags(
  apiTags: string[] | undefined,
  nodeTags: string[] | undefined
) {
  const result: string[] = [];
  const seen = appendTagList(result, apiTags, undefined);
  appendTagList(result, nodeTags, seen);
  return result;
}

function appendTagList(
  result: string[],
  source: string[] | undefined,
  seen: Set<string> | undefined
) {
  if (!source?.length) {
    return seen;
  }
  let currentSeen = seen;
  for (const tag of source) {
    currentSeen = appendUniqueTag(result, tag, currentSeen);
  }
  return currentSeen;
}

function appendUniqueTag(
  tags: string[],
  tag: string | undefined,
  seen: Set<string> | undefined
) {
  const normalizedTag = normalizeNodeTag(tag);
  if (!normalizedTag) {
    return seen;
  }
  if (seen) {
    if (!seen.has(normalizedTag)) {
      seen.add(normalizedTag);
      tags.push(normalizedTag);
    }
    return seen;
  }
  if (tags.includes(normalizedTag)) {
    return seen;
  }
  tags.push(normalizedTag);
  if (tags.length > SMALL_NODE_TAG_LIST_LIMIT) {
    return new Set(tags);
  }
  return seen;
}

function normalizeNodeTag(tag: string | undefined) {
  return tag?.trim() || "";
}

type CompactLookupBuilder<T> = {
  add: (value: T) => void;
  lookup: () => CompactLookup<T>;
};

export function newCompactLookupBuilder<T>(): CompactLookupBuilder<T> {
  const values: T[] = [];
  let set: Set<T> | undefined;
  return {
    add(value) {
      if (set) {
        set.add(value);
        return;
      }
      if (values.includes(value)) {
        return;
      }
      values.push(value);
      if (values.length > SMALL_NODE_LOOKUP_LIMIT) {
        set = new Set(values);
      }
    },
    lookup() {
      if (set) {
        return set;
      }
      return {
        has(value) {
          return values.includes(value);
        },
      };
    },
  };
}

export function nodeInboundKey(serverId: number, inboundAlias: string) {
  return `${serverId}:${inboundAlias}`;
}
