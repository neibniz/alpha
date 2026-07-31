// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Update Payment Method PUT /v1/admin/payment/ */
export async function updatePaymentMethod(
  body: API.UpdatePaymentMethodRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PaymentConfig }>(
    "/v1/admin/payment/",
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

/** Create Payment Method POST /v1/admin/payment/ */
export async function createPaymentMethod(
  body: API.CreatePaymentMethodRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PaymentConfig }>(
    "/v1/admin/payment/",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

/** Delete Payment Method DELETE /v1/admin/payment/ */
export async function deletePaymentMethod(
  body: API.DeletePaymentMethodRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/payment/", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get Payment Method GET /v1/admin/payment/detail */
export async function getPaymentMethod(
  params: API.GetPaymentMethodParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PaymentConfig }>(
    "/v1/admin/payment/detail",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get Payment Method List GET /v1/admin/payment/list */
export async function getPaymentMethodList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetPaymentMethodListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetPaymentMethodListResponse }>(
    "/v1/admin/payment/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get supported payment platform GET /v1/admin/payment/platform */
export async function getPaymentPlatform(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.PlatformResponse }>(
    "/v1/admin/payment/platform",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}
