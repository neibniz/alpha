// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Create subscribe application POST /v1/admin/application/ */
export async function createSubscribeApplication(
  body: API.CreateSubscribeApplicationRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.SubscribeApplication }>(
    "/v1/admin/application/",
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

/** Preview Template GET /v1/admin/application/preview */
export async function previewSubscribeTemplate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PreviewSubscribeTemplateParams,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.PreviewSubscribeTemplateResponse }
  >("/v1/admin/application/preview", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Update subscribe application PUT /v1/admin/application/subscribe_application */
export async function updateSubscribeApplication(
  body: API.UpdateSubscribeApplicationRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.SubscribeApplication }>(
    "/v1/admin/application/subscribe_application",
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

/** Get subscribe application GET /v1/admin/application/subscribe_application */
export async function getSubscribeApplication(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetSubscribeApplicationParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.SubscribeApplication }>(
    "/v1/admin/application/subscribe_application",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Delete subscribe application DELETE /v1/admin/application/subscribe_application */
export async function deleteSubscribeApplication(
  body: API.DeleteSubscribeApplicationRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/application/subscribe_application",
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

/** Get subscribe application list GET /v1/admin/application/subscribe_application_list */
export async function getSubscribeApplicationList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetSubscribeApplicationListParams,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.GetSubscribeApplicationListResponse }
  >("/v1/admin/application/subscribe_application_list", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}
