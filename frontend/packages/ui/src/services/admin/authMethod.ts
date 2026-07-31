// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Get auth method config GET /v1/admin/auth-method/config */
export async function getAuthMethodConfig(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.GetAuthMethodConfigParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.AuthMethodConfig }>(
    "/v1/admin/auth-method/config",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** Update auth method config PUT /v1/admin/auth-method/config */
export async function updateAuthMethodConfig(
  body: API.UpdateAuthMethodConfigRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.AuthMethodConfig }>(
    "/v1/admin/auth-method/config",
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

/** Get email support platform GET /v1/admin/auth-method/email_platform */
export async function getEmailPlatform(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.PlatformResponse }>(
    "/v1/admin/auth-method/email_platform",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Get auth method list GET /v1/admin/auth-method/list */
export async function getAuthMethodList(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.GetAuthMethodListResponse }>(
    "/v1/admin/auth-method/list",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Test email send POST /v1/admin/auth-method/test_email_send */
export async function testEmailSend(
  body: API.TestEmailSendRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/auth-method/test_email_send",
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
