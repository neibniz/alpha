// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Update Ads PUT /v1/admin/ads/ */
export async function updateAds(
  body: API.UpdateAdsRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/ads/", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create Ads POST /v1/admin/ads/ */
export async function createAds(
  body: API.CreateAdsRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/ads/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete Ads DELETE /v1/admin/ads/ */
export async function deleteAds(
  body: API.DeleteAdsRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/ads/", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get Ads Detail GET /v1/admin/ads/detail */
export async function getAdsDetail(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetAdsDetailParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.Ads }>("/v1/admin/ads/detail", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Get Ads List GET /v1/admin/ads/list */
export async function getAdsList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetAdsListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetAdsListResponse }>(
    "/v1/admin/ads/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
