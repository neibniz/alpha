// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Query revenue statistics GET /v1/admin/console/revenue */
export async function queryRevenueStatistics(
  params?: API.StatisticsPeriodParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.RevenueStatisticsResponse }>(
    "/v1/admin/console/revenue",
    {
      method: "GET",
      params,
      ...(options || {}),
    }
  );
}

/** Query user statistics GET /v1/admin/console/user */
export async function queryUserStatistics(
  params?: API.StatisticsPeriodParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.UserStatisticsResponse }>(
    "/v1/admin/console/user",
    {
      method: "GET",
      params,
      ...(options || {}),
    }
  );
}
