// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Filter balance log GET /v1/admin/log/balance/list */
export async function filterBalanceLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterBalanceLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterBalanceLogResponse }>(
    "/v1/admin/log/balance/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter commission log GET /v1/admin/log/commission/list */
export async function filterCommissionLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterCommissionLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterCommissionLogResponse }>(
    "/v1/admin/log/commission/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter email log GET /v1/admin/log/email/list */
export async function filterEmailLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterEmailLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterEmailLogResponse }>(
    "/v1/admin/log/email/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter gift log GET /v1/admin/log/gift/list */
export async function filterGiftLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterGiftLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterGiftLogResponse }>(
    "/v1/admin/log/gift/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter login log GET /v1/admin/log/login/list */
export async function filterLoginLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterLoginLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterLoginLogResponse }>(
    "/v1/admin/log/login/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get message log list GET /v1/admin/log/message/list */
export async function getMessageLogList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetMessageLogListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetMessageLogListResponse }>(
    "/v1/admin/log/message/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter Xray proxy access logs GET /v1/admin/log/proxy/access */
export async function filterProxyAccessLog(
  params: API.FilterProxyAccessLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterProxyAccessLogResponse }>(
    "/v1/admin/log/proxy/access",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter register log GET /v1/admin/log/register/list */
export async function filterRegisterLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterRegisterLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterRegisterLogResponse }>(
    "/v1/admin/log/register/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter server traffic log GET /v1/admin/log/server/traffic/list */
export async function filterServerTrafficLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterServerTrafficLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterServerTrafficLogResponse }>(
    "/v1/admin/log/server/traffic/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get log setting GET /v1/admin/log/setting */
export async function getLogSetting(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.LogSetting }>(
    "/v1/admin/log/setting",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update log setting POST /v1/admin/log/setting */
export async function updateLogSetting(
  body: API.LogSetting,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/log/setting", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Filter subscribe log GET /v1/admin/log/subscribe/list */
export async function filterSubscribeLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterSubscribeLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterSubscribeLogResponse }>(
    "/v1/admin/log/subscribe/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter reset subscribe log GET /v1/admin/log/subscribe/reset/list */
export async function filterResetSubscribeLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterResetSubscribeLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterResetSubscribeLogResponse }>(
    "/v1/admin/log/subscribe/reset/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter user subscribe traffic log GET /v1/admin/log/subscribe/traffic/list */
export async function filterUserSubscribeTrafficLog(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterUserSubscribeTrafficLogParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterSubscribeTrafficResponse }>(
    "/v1/admin/log/subscribe/traffic/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Filter traffic log details GET /v1/admin/log/traffic/details */
export async function filterTrafficLogDetails(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FilterTrafficLogDetailsParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.FilterTrafficLogDetailsResponse }>(
    "/v1/admin/log/traffic/details",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
