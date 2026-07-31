// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Update subscribe PUT /v1/admin/subscribe/ */
export async function updateSubscribe(
  body: API.UpdateSubscribeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create subscribe POST /v1/admin/subscribe/ */
export async function createSubscribe(
  body: API.CreateSubscribeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete subscribe DELETE /v1/admin/subscribe/ */
export async function deleteSubscribe(
  body: API.DeleteSubscribeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Batch delete subscribe DELETE /v1/admin/subscribe/batch */
export async function batchDeleteSubscribe(
  body: API.BatchDeleteSubscribeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/batch", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get subscribe details GET /v1/admin/subscribe/details */
export async function getSubscribeDetails(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetSubscribeDetailsParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.Subscribe }>(
    "/v1/admin/subscribe/details",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Update subscribe group PUT /v1/admin/subscribe/group */
export async function updateSubscribeGroup(
  body: API.UpdateSubscribeGroupRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/group", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create subscribe group POST /v1/admin/subscribe/group */
export async function createSubscribeGroup(
  body: API.CreateSubscribeGroupRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/group", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete subscribe group DELETE /v1/admin/subscribe/group */
export async function deleteSubscribeGroup(
  body: API.DeleteSubscribeGroupRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/group", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Batch delete subscribe group DELETE /v1/admin/subscribe/group/batch */
export async function batchDeleteSubscribeGroup(
  body: API.BatchDeleteSubscribeGroupRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/subscribe/group/batch",
    {
      method: "DELETE",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

/** Get subscribe group list GET /v1/admin/subscribe/group/list */
export async function getSubscribeGroupList(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.GetSubscribeGroupListResponse }>(
    "/v1/admin/subscribe/group/list",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Get subscribe list GET /v1/admin/subscribe/list */
export async function getSubscribeList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetSubscribeListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetSubscribeListResponse }>(
    "/v1/admin/subscribe/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Reset all subscribe tokens POST /v1/admin/subscribe/reset_all_token */
export async function resetAllSubscribeToken(
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.ResetAllSubscribeTokenResponse }>(
    "/v1/admin/subscribe/reset_all_token",
    {
      method: "POST",
      ...(options || {}),
    }
  );
}

/** Subscribe sort POST /v1/admin/subscribe/sort */
export async function subscribeSort(
  body: API.SubscribeSortRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/subscribe/sort", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}
