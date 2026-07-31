// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Purchase Checkout POST /v1/public/portal/order/checkout */
export async function purchaseCheckout(
  body: API.CheckoutOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.CheckoutOrderResponse }>(
    "/v1/public/portal/order/checkout",
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

/** Query Purchase Order GET /v1/public/portal/order/status */
export async function queryPurchaseOrder(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.QueryPurchaseOrderParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.QueryPurchaseOrderResponse }>(
    "/v1/public/portal/order/status",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get available payment methods GET /v1/public/portal/payment-method */
export async function getAvailablePaymentMethods(options?: {
  [key: string]: unknown;
}) {
  return request<
    API.Response & { data?: API.GetAvailablePaymentMethodsResponse }
  >("/v1/public/portal/payment-method", {
    method: "GET",
    ...(options || {}),
  });
}

/** Pre Purchase Order POST /v1/public/portal/pre */
export async function prePurchaseOrder(
  body: API.PrePurchaseOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PrePurchaseOrderResponse }>(
    "/v1/public/portal/pre",
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

/** Purchase subscription POST /v1/public/portal/purchase */
export async function purchase(
  body: API.PortalPurchaseRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PortalPurchaseResponse }>(
    "/v1/public/portal/purchase",
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

/** Get Subscription GET /v1/public/portal/subscribe */
export async function getSubscription(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetSubscriptionParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetSubscriptionResponse }>(
    "/v1/public/portal/subscribe",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
