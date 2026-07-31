// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Update coupon PUT /v1/admin/coupon/ */
export async function updateCoupon(
  body: API.UpdateCouponRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/coupon/", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create coupon POST /v1/admin/coupon/ */
export async function createCoupon(
  body: API.CreateCouponRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/coupon/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete coupon DELETE /v1/admin/coupon/ */
export async function deleteCoupon(
  body: API.DeleteCouponRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/coupon/", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Batch delete coupon DELETE /v1/admin/coupon/batch */
export async function batchDeleteCoupon(
  body: API.BatchDeleteCouponRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/coupon/batch", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get coupon GET /v1/admin/coupon/ */
export async function getCoupon(
  params: API.GetCouponParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.Coupon }>("/v1/admin/coupon/", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Get coupon list GET /v1/admin/coupon/list */
export async function getCouponList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetCouponListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetCouponListResponse }>(
    "/v1/admin/coupon/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
