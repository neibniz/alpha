// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Get membership plan GET /v1/admin/membership/plan */
export async function queryMembershipPlan(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.MembershipPlan }>(
    "/v1/admin/membership/plan",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update membership plan PUT /v1/admin/membership/plan */
export async function updateMembershipPlan(
  body: API.UpdateMembershipPlanRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.MembershipPlan }>(
    "/v1/admin/membership/plan",
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
