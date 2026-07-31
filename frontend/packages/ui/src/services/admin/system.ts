// @ts-nocheck
/* eslint-disable */
import request from "@workspace/ui/lib/request";

/** Get Currency Config GET /v1/admin/system/currency_config */
export async function getCurrencyConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.CurrencyConfig }>(
    "/v1/admin/system/currency_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update Currency Config PUT /v1/admin/system/currency_config */
export async function updateCurrencyConfig(
  body: API.CurrencyConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/currency_config",
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

/** Get Node Multiplier GET /v1/admin/system/get_node_multiplier */
export async function getNodeMultiplier(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.GetNodeMultiplierResponse }>(
    "/v1/admin/system/get_node_multiplier",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Get invite config GET /v1/admin/system/invite_config */
export async function getInviteConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.InviteConfig }>(
    "/v1/admin/system/invite_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update invite config PUT /v1/admin/system/invite_config */
export async function updateInviteConfig(
  body: API.InviteConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/invite_config",
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

/** Get node config GET /v1/admin/system/node_config */
export async function getNodeConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.NodeConfig }>(
    "/v1/admin/system/node_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update node config PUT /v1/admin/system/node_config */
export async function updateNodeConfig(
  body: API.NodeConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/node_config",
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

/** PreView Node Multiplier GET /v1/admin/system/node_multiplier/preview */
export async function preViewNodeMultiplier(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.PreViewNodeMultiplierResponse }>(
    "/v1/admin/system/node_multiplier/preview",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** get Privacy Policy Config GET /v1/admin/system/privacy */
export async function getPrivacyPolicyConfig(
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.PrivacyPolicyConfig }>(
    "/v1/admin/system/privacy",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update Privacy Policy Config PUT /v1/admin/system/privacy */
export async function updatePrivacyPolicyConfig(
  body: API.PrivacyPolicyConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>("/v1/admin/system/privacy", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** Get register config GET /v1/admin/system/register_config */
export async function getRegisterConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.RegisterConfig }>(
    "/v1/admin/system/register_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update register config PUT /v1/admin/system/register_config */
export async function updateRegisterConfig(
  body: API.RegisterConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/register_config",
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

/** Set Node Multiplier POST /v1/admin/system/set_node_multiplier */
export async function setNodeMultiplier(
  body: API.SetNodeMultiplierRequest,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/set_node_multiplier",
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

/** setting telegram bot POST /v1/admin/system/setting_telegram_bot */
export async function settingTelegramBot(options?: Record<string, unknown>) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/setting_telegram_bot",
    {
      method: "POST",
      ...(options || {}),
    }
  );
}

/** Get site config GET /v1/admin/system/site_config */
export async function getSiteConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.SiteConfig }>(
    "/v1/admin/system/site_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update site config PUT /v1/admin/system/site_config */
export async function updateSiteConfig(
  body: API.SiteConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/site_config",
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

/** Get subscribe config GET /v1/admin/system/subscribe_config */
export async function getSubscribeConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.SubscribeConfig }>(
    "/v1/admin/system/subscribe_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update subscribe config PUT /v1/admin/system/subscribe_config */
export async function updateSubscribeConfig(
  body: API.SubscribeConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/subscribe_config",
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

/** Get Team of Service Config GET /v1/admin/system/tos_config */
export async function getTosConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.TosConfig }>(
    "/v1/admin/system/tos_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update Team of Service Config PUT /v1/admin/system/tos_config */
export async function updateTosConfig(
  body: API.TosConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/tos_config",
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

/** Get Verify Code Config GET /v1/admin/system/verify_code_config */
export async function getVerifyCodeConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.VerifyCodeConfig }>(
    "/v1/admin/system/verify_code_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update Verify Code Config PUT /v1/admin/system/verify_code_config */
export async function updateVerifyCodeConfig(
  body: API.VerifyCodeConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/verify_code_config",
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

/** Get verify config GET /v1/admin/system/verify_config */
export async function getVerifyConfig(options?: Record<string, unknown>) {
  return request<API.Response & { data?: API.VerifyConfig }>(
    "/v1/admin/system/verify_config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** Update verify config PUT /v1/admin/system/verify_config */
export async function updateVerifyConfig(
  body: API.VerifyConfig,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: never }>(
    "/v1/admin/system/verify_config",
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
