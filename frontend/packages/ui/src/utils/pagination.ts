import type { ColumnFiltersState } from "@tanstack/react-table";

export type PageRequest = {
  page: number;
  size: number;
};

export type PageResult<TData> = {
  list: TData[];
  total: number;
};

export type ListResult<TData> = {
  list?: TData[];
};

export type FilterSnapshot<TFilter extends Record<string, unknown>> = {
  values: TFilter | Record<string, never>;
  signature: string;
};

const EMPTY_FILTER_VALUES: Record<string, never> = Object.freeze({});
const DEFAULT_FETCH_ALL_PAGES_CONCURRENCY = 4;

type PaginatedResponse<TData> = {
  data?: {
    data?: Partial<PageResult<TData>>;
  };
};

type PageFetcher<TData> = (
  params: PageRequest
) => Promise<PaginatedResponse<TData>>;

export function createPageRequest(pageIndex: number, pageSize: number) {
  return {
    page: pageIndex + 1,
    size: pageSize,
  };
}

export function createFilterSnapshot<TFilter extends Record<string, unknown>>(
  filters: ColumnFiltersState
): FilterSnapshot<TFilter> {
  if (filters.length === 0) {
    return {
      values: EMPTY_FILTER_VALUES,
      signature: "",
    };
  }

  const values: Record<string, unknown> = {};
  let signature = "";

  for (const filter of filters) {
    values[filter.id] = filter.value;
    if (signature) {
      signature += "|";
    }
    signature += JSON.stringify([filter.id, filter.value]);
  }

  return {
    values: values as TFilter,
    signature,
  };
}

export function normalizePageResult<TData>(
  response: Partial<PageResult<TData>> | null | undefined
): PageResult<TData> {
  const list = normalizeListResult(response);
  const total = Number(response?.total);
  return {
    list,
    total: Number.isFinite(total) && total > 0 ? total : list.length,
  };
}

export function normalizeListResult<TData>(
  response: Partial<ListResult<TData>> | null | undefined
) {
  return Array.isArray(response?.list) ? response.list : [];
}

export async function fetchAllPages<TData>(
  fetchPage: PageFetcher<TData>,
  pageSize = 100,
  maxConcurrency = DEFAULT_FETCH_ALL_PAGES_CONCURRENCY
) {
  const size = normalizePageSize(pageSize);
  const concurrency = normalizeConcurrency(maxConcurrency);
  const firstResponse = await fetchPage({ page: 1, size });
  const firstPage = normalizePageResult(firstResponse.data?.data);

  if (firstPage.list.length === 0) {
    return [];
  }

  const items: TData[] = [];
  const total = firstPage.total;
  appendPageItems(items, firstPage.list, total);
  if (items.length >= total || firstPage.list.length < size) {
    return items;
  }

  const pageCount = Math.ceil(total / size);
  for (let pageStart = 2; pageStart <= pageCount; pageStart += concurrency) {
    const pageEnd = Math.min(pageCount, pageStart + concurrency - 1);
    const responses = await Promise.all(
      createPageRequests(fetchPage, pageStart, pageEnd, size)
    );

    for (const response of responses) {
      const { list } = normalizePageResult(response.data?.data);
      appendPageItems(items, list, total);
      if (items.length >= total) {
        return items;
      }
    }
  }

  return items;
}

function normalizePageSize(value: number) {
  return Number.isFinite(value) && value > 0 ? Math.floor(value) : 100;
}

function normalizeConcurrency(value: number) {
  return Number.isFinite(value) && value > 0
    ? Math.max(1, Math.floor(value))
    : 1;
}

function createPageRequests<TData>(
  fetchPage: PageFetcher<TData>,
  start: number,
  end: number,
  size: number
) {
  const pages = new Array<Promise<PaginatedResponse<TData>>>(end - start + 1);
  let index = 0;
  for (let page = start; page <= end; page += 1) {
    pages[index] = fetchPage({ page, size });
    index += 1;
  }
  return pages;
}

function appendPageItems<TData>(
  items: TData[],
  pageItems: TData[],
  total: number
) {
  const remaining = total - items.length;
  if (remaining <= 0 || pageItems.length === 0) {
    return;
  }

  const count = Math.min(pageItems.length, remaining);
  for (let index = 0; index < count; index += 1) {
    items.push(pageItems[index]!);
  }
}
