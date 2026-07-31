// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Update announcement PUT /v1/admin/announcement/ */
export async function updateAnnouncement(
  body: API.UpdateAnnouncementRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/announcement/", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create announcement POST /v1/admin/announcement/ */
export async function createAnnouncement(
  body: API.CreateAnnouncementRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/announcement/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete announcement DELETE /v1/admin/announcement/ */
export async function deleteAnnouncement(
  body: API.DeleteAnnouncementRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/announcement/", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get announcement GET /v1/admin/announcement/detail */
export async function getAnnouncement(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetAnnouncementParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.Announcement }>(
    "/v1/admin/announcement/detail",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get announcement list GET /v1/admin/announcement/list */
export async function getAnnouncementList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetAnnouncementListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetAnnouncementListResponse }>(
    "/v1/admin/announcement/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
