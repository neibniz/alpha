// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Create Server POST /v1/admin/server/create */
export async function createServer(
  body: API.CreateServerRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/server/create", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete Server POST /v1/admin/server/delete */
export async function deleteServer(
  body: API.DeleteServerRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/server/delete", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Filter Server List GET /v1/admin/server/list */
export async function filterServerList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterServerListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterServerListResponse }>(
    "/v1/admin/server/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Query per-server network rate history GET /v1/admin/server/network-rate/history */
export async function queryServerNetworkRateHistory(
  params: API.QueryServerNetworkRateHistoryParams,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.QueryServerNetworkRateHistoryResponse }
  >("/v1/admin/server/network-rate/history", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Query the authentication token bound to one server GET /v1/admin/server/node-token */
export async function queryServerNodeToken(
  params: API.QueryServerNodeTokenParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.QueryServerNodeTokenResponse }>(
    "/v1/admin/server/node-token",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Create Node POST /v1/admin/server/node/create */
export async function createNode(
  body: API.CreateNodeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/node/create",
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

/** Delete Node POST /v1/admin/server/node/delete */
export async function deleteNode(
  body: API.DeleteNodeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/node/delete",
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

/** Filter Node List GET /v1/admin/server/node/list */
export async function filterNodeList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterNodeListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterNodeListResponse }>(
    "/v1/admin/server/node/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Reset node sort POST /v1/admin/server/node/sort */
export async function resetSortWithNode(
  body: API.ResetSortRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/node/sort",
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

/** Toggle Node Status POST /v1/admin/server/node/status/toggle */
export async function toggleNodeStatus(
  body: API.ToggleNodeStatusRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/node/status/toggle",
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

/** Query all node tags GET /v1/admin/server/node/tags */
export async function queryNodeTag(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.QueryNodeTagResponse }>(
    "/v1/admin/server/node/tags",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update Node POST /v1/admin/server/node/update */
export async function updateNode(
  body: API.UpdateNodeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/node/update",
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

/** Reset server sort POST /v1/admin/server/server/sort */
export async function resetSortWithServer(
  body: API.ResetSortRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/server/sort",
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

/** Update Server POST /v1/admin/server/update */
export async function updateServer(
  body: API.UpdateServerRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/server/update", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create Xray Template POST /v1/admin/server/xray-template/create */
export async function createXrayTemplate(
  body: API.CreateXrayTemplateRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/xray-template/create",
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

/** Update Xray Template POST /v1/admin/server/xray-template/update */
export async function updateXrayTemplate(
  body: API.UpdateXrayTemplateRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/xray-template/update",
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

/** Delete Xray Template POST /v1/admin/server/xray-template/delete */
export async function deleteXrayTemplate(
  body: API.DeleteXrayTemplateRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/xray-template/delete",
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

/** Get Xray Template Detail GET /v1/admin/server/xray-template/detail */
export async function getXrayTemplateDetail(
  params: API.GetXrayTemplateDetailParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.XrayTemplate }>(
    "/v1/admin/server/xray-template/detail",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter Xray Template List GET /v1/admin/server/xray-template/list */
export async function filterXrayTemplateList(
  params: API.FilterXrayTemplateListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterXrayTemplateListResponse }>(
    "/v1/admin/server/xray-template/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Query xray template catalog GET /v1/admin/server/xray-template/catalog */
export async function queryXrayTemplateCatalog(
  params: API.QueryXrayTemplateCatalogParams,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.QueryXrayTemplateCatalogResponse }
  >("/v1/admin/server/xray-template/catalog", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Bind Xray Templates To Server POST /v1/admin/server/xray-template/bind */
export async function bindServerXrayTemplates(
  body: API.BindServerXrayTemplatesRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/server/xray-template/bind",
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

/** Preview Server Xray Config POST /v1/admin/server/xray-template/preview */
export async function previewServerXrayConfig(
  body: API.PreviewServerXrayConfigRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PreviewServerXrayConfigResponse }>(
    "/v1/admin/server/xray-template/preview",
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

/** Query Server Xray Templates GET /v1/admin/server/xray-template/server */
export async function queryServerXrayTemplateList(
  params: API.QueryServerXrayTemplateListParams,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.QueryServerXrayTemplateListResponse }
  >("/v1/admin/server/xray-template/server", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}
