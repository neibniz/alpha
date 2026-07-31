// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Update document PUT /v1/admin/document/ */
export async function updateDocument(
  body: API.UpdateDocumentRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/document/", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create document POST /v1/admin/document/ */
export async function createDocument(
  body: API.CreateDocumentRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/document/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete document DELETE /v1/admin/document/ */
export async function deleteDocument(
  body: API.DeleteDocumentRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/document/", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Batch delete document DELETE /v1/admin/document/batch */
export async function batchDeleteDocument(
  body: API.BatchDeleteDocumentRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/document/batch", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get document detail GET /v1/admin/document/detail */
export async function getDocumentDetail(
  params: API.GetDocumentDetailParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.Document }>(
    "/v1/admin/document/detail",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get document list GET /v1/admin/document/list */
export async function getDocumentList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetDocumentListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetDocumentListResponse }>(
    "/v1/admin/document/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
