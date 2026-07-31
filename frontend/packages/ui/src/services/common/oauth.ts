// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** OAuth login POST /v1/auth/oauth/login */
export async function oAuthLogin(
  body: API.OAthLoginRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.OAuthLoginResponse }>(
    "/v1/auth/oauth/login",
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

/** OAuth login get token POST /v1/auth/oauth/login/token */
export async function oAuthLoginGetToken(
  body: API.OAuthLoginGetTokenRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.LoginResponse }>(
    "/v1/auth/oauth/login/token",
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
