// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";
import { withRequestId } from "@workspace/ui/utils/request-id";

/** Close order POST /v1/public/order/close */
export async function closeOrder(
  body: API.CloseOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/public/order/close", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get order GET /v1/public/order/detail */
export async function queryOrderDetail(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.QueryOrderDetailParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.OrderDetail }>(
    "/v1/public/order/detail",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get order list GET /v1/public/order/list */
export async function queryOrderList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.QueryOrderListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.QueryOrderListResponse }>(
    "/v1/public/order/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Pre create order POST /v1/public/order/pre */
export async function preCreateOrder(
  body: API.PurchaseOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PreOrderResponse }>(
    "/v1/public/order/pre",
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

/** purchase Subscription POST /v1/public/order/purchase */
export async function purchase(
  body: API.PurchaseOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PurchaseOrderResponse }>(
    "/v1/public/order/purchase",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: withRequestId(body),
      ...(options || {}),
    }
  );
}

/** Get membership plan GET /v1/public/order/membership/plan */
export async function queryMembershipPlan(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.MembershipPlan }>(
    "/v1/public/order/membership/plan",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Get membership card GET /v1/public/order/membership/card */
export async function queryMembershipCard(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.MembershipCardResponse }>(
    "/v1/public/order/membership/card",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Purchase membership card POST /v1/public/order/membership */
export async function purchaseMembership(
  body: API.PurchaseMembershipOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PurchaseMembershipOrderResponse }>(
    "/v1/public/order/membership",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: withRequestId(body),
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /v1/public/order/recharge */
export async function recharge(
  body: API.RechargeOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.RechargeOrderResponse }>(
    "/v1/public/order/recharge",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: withRequestId(body),
      ...(options || {}),
    }
  );
}

/** Renewal Subscription POST /v1/public/order/renewal */
export async function renewal(
  body: API.RenewalOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.RenewalOrderResponse }>(
    "/v1/public/order/renewal",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: withRequestId(body),
      ...(options || {}),
    }
  );
}

/** Quote a renewal using the current product price and rules. POST /v1/public/order/renewal/pre */
export async function preRenewal(
  body: API.PreRenewalOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PreRenewalOrderResponse }>(
    "/v1/public/order/renewal/pre",
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

/** Reset traffic POST /v1/public/order/reset */
export async function resetTraffic(
  body: API.ResetTrafficOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.ResetTrafficOrderResponse }>(
    "/v1/public/order/reset",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: withRequestId(body),
      ...(options || {}),
    }
  );
}

/** Preview a traffic reset with current product pricing. POST /v1/public/order/reset/pre */
export async function preResetTraffic(
  body: API.PreResetTrafficOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PreResetTrafficOrderResponse }>(
    "/v1/public/order/reset/pre",
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
