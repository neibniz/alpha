type NumericIdItem = {
  id?: number;
};

export type NumericIdListIndex<T extends NumericIdItem> = {
  byId: Map<number, T>;
  indexById: Map<number, number>;
};

export function buildByNumericId<T extends NumericIdItem>(items: readonly T[]) {
  const byId = new Map<number, T>();
  for (const item of items) {
    if (isPositiveNumericId(item.id)) {
      byId.set(item.id, item);
    }
  }
  return byId;
}

export function buildNumericIdListIndex<T extends NumericIdItem>(
  items: readonly T[]
): NumericIdListIndex<T> {
  const byId = new Map<number, T>();
  const indexById = new Map<number, number>();
  for (let index = 0; index < items.length; index += 1) {
    const item = items[index];
    if (isPositiveNumericId(item?.id)) {
      byId.set(item.id, item);
      indexById.set(item.id, index);
    }
  }
  return {
    byId,
    indexById,
  };
}

export function patchListItemById<T extends NumericIdItem>(
  items: readonly T[],
  byId: ReadonlyMap<number, T>,
  patch: Partial<T> & NumericIdItem,
  indexById?: ReadonlyMap<number, number>
) {
  const id = patch.id;
  if (!isPositiveNumericId(id)) {
    return;
  }

  const indexedPosition = indexById?.get(id);
  const hasCurrentPosition =
    indexedPosition !== undefined && items[indexedPosition]?.id === id;
  const index = hasCurrentPosition
    ? indexedPosition
    : findItemIndexById(items, id);

  if (index < 0) {
    return;
  }

  const item = items[index];
  if (!item) {
    return;
  }
  if (itemMatchesPatch(item, patch)) {
    return;
  }

  const nextItem = { ...item, ...patch } as T;
  const nextItems = items.slice();
  nextItems[index] = nextItem;
  const nextById = new Map(byId);
  nextById.set(id, nextItem);
  const nextIndexById = nextListIndex(indexById, id, index, hasCurrentPosition);
  return {
    items: nextItems,
    byId: nextById,
    indexById: nextIndexById,
  };
}

function isPositiveNumericId(id: number | undefined): id is number {
  return typeof id === "number" && Number.isInteger(id) && id > 0;
}

function itemMatchesPatch<T extends NumericIdItem>(
  item: T,
  patch: Partial<T> & NumericIdItem
) {
  for (const [key, value] of Object.entries(patch) as [keyof T, T[keyof T]][]) {
    if (item[key] !== value) {
      return false;
    }
  }
  return true;
}

function findItemIndexById<T extends NumericIdItem>(
  items: readonly T[],
  id: number
) {
  for (let index = 0; index < items.length; index += 1) {
    if (items[index]?.id === id) {
      return index;
    }
  }
  return -1;
}

function nextListIndex(
  indexById: ReadonlyMap<number, number> | undefined,
  id: number,
  index: number,
  hasCurrentPosition: boolean
) {
  if (!indexById) {
    return;
  }
  if (hasCurrentPosition && indexById instanceof Map) {
    return indexById;
  }
  const nextIndexById = new Map(indexById);
  nextIndexById.set(id, index);
  return nextIndexById;
}
