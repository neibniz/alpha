// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Create order POST /v1/admin/order/ */
export async function createOrder(
  body: API.CreateOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/order/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get order list GET /v1/admin/order/list */
export async function getOrderList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetOrderListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetOrderListResponse }>(
    "/v1/admin/order/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get order detail GET /v1/admin/order/detail */
export async function getOrderDetail(
  params: API.GetOrderDetailParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.OrderDetail }>(
    "/v1/admin/order/detail",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Update order status PUT /v1/admin/order/status */
export async function updateOrderStatus(
  body: API.UpdateOrderStatusRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/order/status", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}
