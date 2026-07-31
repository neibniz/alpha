const ADMIN_ORDER_STATUS_TARGETS: Readonly<Record<number, readonly number[]>> =
  {
    1: [2, 3, 4],
    3: [1],
    4: [1],
  };

export function adminOrderStatusTargets(status: number): number[] {
  return [...(ADMIN_ORDER_STATUS_TARGETS[status] || [])];
}

export type AdminOrderListFilter = {
  search?: string;
  status?: string;
  subscribe_id?: string;
  user_id?: string;
};

function positiveInteger(value: string | undefined): number | undefined {
  if (!value?.trim()) return undefined;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
}

export function toAdminOrderListRequest(
  pagination: { page: number; size: number },
  filter: AdminOrderListFilter
): API.GetOrderListParams {
  return {
    ...pagination,
    search: filter.search?.trim() || undefined,
    status: positiveInteger(filter.status),
    subscribe_id: positiveInteger(filter.subscribe_id),
    user_id: positiveInteger(filter.user_id),
  };
}
