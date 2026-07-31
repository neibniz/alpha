// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Get available payment methods GET /v1/public/payment/methods */
export async function getAvailablePaymentMethods(options?: {
  [key: string]: unknown;
}) {
  return request<
    API.Response & { data?: API.GetAvailablePaymentMethodsResponse }
  >("/v1/public/payment/methods", {
    method: "GET",
    ...(options || {}),
  });
}
