// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Create user POST /v1/admin/user/ */
export async function createUser(
  body: API.CreateUserRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete user DELETE /v1/admin/user/ */
export async function deleteUser(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.DeleteUserParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/", {
    method: "DELETE",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Get user auth method GET /v1/admin/user/auth_method */
export async function getUserAuthMethod(
  params: API.GetUserAuthMethodParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetUserAuthMethodResponse }>(
    "/v1/admin/user/auth_method",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Update user auth method PUT /v1/admin/user/auth_method */
export async function updateUserAuthMethod(
  body: API.UpdateUserAuthMethodRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/user/auth_method",
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

/** Create user auth method POST /v1/admin/user/auth_method */
export async function createUserAuthMethod(
  body: API.CreateUserAuthMethodRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/user/auth_method",
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

/** Delete user auth method DELETE /v1/admin/user/auth_method */
export async function deleteUserAuthMethod(
  body: API.DeleteUserAuthMethodRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/user/auth_method",
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

/** Update user basic info PUT /v1/admin/user/basic */
export async function updateUserBasicInfo(
  body: API.UpdateUserBasiceInfoRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/basic", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Update user membership PUT /v1/admin/user/membership */
export async function updateUserMembership(
  body: API.UpdateUserMembershipRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/membership", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Batch delete user DELETE /v1/admin/user/batch */
export async function batchDeleteUser(
  body: API.BatchDeleteUserRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/batch", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Current user GET /v1/admin/user/current */
export async function currentUser(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.User }>("/v1/admin/user/current", {
    method: "GET",
    ...(options || {}),
  });
}

/** Get user detail GET /v1/admin/user/detail */
export async function getUserDetail(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserDetailParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.User }>("/v1/admin/user/detail", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** User device PUT /v1/admin/user/device */
export async function updateUserDevice(
  body: API.UserDevice,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/device", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Delete user device DELETE /v1/admin/user/device */
export async function deleteUserDevice(
  body: API.DeleteUserDeivceRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/device", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** kick offline user device PUT /v1/admin/user/device/kick_offline */
export async function kickOfflineByUserDevice(
  body: API.KickOfflineRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/user/device/kick_offline",
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

/** Get user list GET /v1/admin/user/list */
export async function getUserList(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetUserListResponse }>(
    "/v1/admin/user/list",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get user login logs GET /v1/admin/user/login/logs */
export async function getUserLoginLogs(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserLoginLogsParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetUserLoginLogsResponse }>(
    "/v1/admin/user/login/logs",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Update user notify setting PUT /v1/admin/user/notify */
export async function updateUserNotifySetting(
  body: API.UpdateUserNotifySettingRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/notify", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get user subcribe GET /v1/admin/user/subscribe */
export async function getUserSubscribe(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserSubscribeParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetUserSubscribeListResponse }>(
    "/v1/admin/user/subscribe",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get user subcribe access GET /v1/admin/user/subscribe/access */
export async function getUserSubscribeAccess(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserSubscribeAccessParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.UserSubscribeAccess }>(
    "/v1/admin/user/subscribe/access",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Update user subcribe PUT /v1/admin/user/subscribe */
export async function updateUserSubscribe(
  body: API.UpdateUserSubscribeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/subscribe", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Create user subcribe POST /v1/admin/user/subscribe */
export async function createUserSubscribe(
  body: API.CreateUserSubscribeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/subscribe", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Preview an administrator-issued subscription order POST /v1/admin/user/subscribe/issue/quote */
export async function quoteUserSubscribeOrder(
  body: API.IssueUserSubscribeOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.IssueUserSubscribeOrderQuote }>(
    "/v1/admin/user/subscribe/issue/quote",
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

/** Issue a paid order and queue subscription activation POST /v1/admin/user/subscribe/issue */
export async function issueUserSubscribeOrder(
  body: API.IssueUserSubscribeOrderRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.IssueUserSubscribeOrderResponse }>(
    "/v1/admin/user/subscribe/issue",
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

/** Delete user subcribe DELETE /v1/admin/user/subscribe */
export async function deleteUserSubscribe(
  body: API.DeleteUserSubscribeRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/user/subscribe", {
    method: "DELETE",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get user subcribe by id GET /v1/admin/user/subscribe/detail */
export async function getUserSubscribeById(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserSubscribeByIdParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.UserSubscribeDetail }>(
    "/v1/admin/user/subscribe/detail",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get user subcribe devices GET /v1/admin/user/subscribe/device */
export async function getUserSubscribeDevices(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserSubscribeDevicesParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetUserSubscribeDevicesResponse }>(
    "/v1/admin/user/subscribe/device",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get user subcribe logs GET /v1/admin/user/subscribe/logs */
export async function getUserSubscribeLogs(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserSubscribeLogsParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.GetUserSubscribeLogsResponse }>(
    "/v1/admin/user/subscribe/logs",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Get user subcribe reset traffic logs GET /v1/admin/user/subscribe/reset/logs */
export async function getUserSubscribeResetTrafficLogs(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserSubscribeResetTrafficLogsParams,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.GetUserSubscribeResetTrafficLogsResponse }
  >("/v1/admin/user/subscribe/reset/logs", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Reset user subscribe token POST /v1/admin/user/subscribe/reset/token */
export async function resetUserSubscribeToken(
  body: API.ResetUserSubscribeTokenRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/user/subscribe/reset/token",
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

/** Reset user subscribe traffic POST /v1/admin/user/subscribe/reset/traffic */
export async function resetUserSubscribeTraffic(
  body: API.ResetUserSubscribeTrafficRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/user/subscribe/reset/traffic",
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

/** Reset traffic for all, selected users, or selected subscriptions POST /v1/admin/user/subscribe/reset/traffic/batch */
export async function batchResetUserSubscribeTraffic(
  body: API.BatchResetUserSubscribeTrafficRequest,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.BatchResetUserSubscribeTrafficResponse }
  >("/v1/admin/user/subscribe/reset/traffic/batch", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Stop user subscribe POST /v1/admin/user/subscribe/toggle */
export async function toggleUserSubscribeStatus(
  body: API.ToggleUserSubscribeStatusRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/user/subscribe/toggle",
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

/** Get user subcribe traffic logs GET /v1/admin/user/subscribe/traffic_logs */
export async function getUserSubscribeTrafficLogs(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetUserSubscribeTrafficLogsParams,
  options?: Record<string, unknown>
) {
  return request<
    API.Response & { data?: API.GetUserSubscribeTrafficLogsResponse }
  >("/v1/admin/user/subscribe/traffic_logs", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** Get subscription traffic series GET /v1/admin/user/subscribe/traffic_series */
export async function getUserSubscribeTrafficSeries(
  params: API.GetUserSubscribeTrafficSeriesRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.TrafficSeriesResponse }>(
    "/v1/admin/user/subscribe/traffic_series",
    {
      method: "GET",
      params: { ...params },
      ...(options || {}),
    }
  );
}
