import request from "@workspace/ui/lib/request";

export async function listForwarders(
  params?: API.ForwarderListParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.ForwarderListResponse }>(
    "/v1/admin/forwarder/list",
    { method: "GET", params: params || {}, ...(options || {}) }
  );
}

export async function getForwarderConfig(
  params: API.ForwarderConfigParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.ForwarderConfig }>(
    "/v1/admin/forwarder/config",
    { method: "GET", params, ...(options || {}) }
  );
}

export async function listForwarderConnections(
  params: API.ForwarderConnectionParams,
  options?: Record<string, unknown>
) {
  return request<API.Response & { data?: API.ForwarderConnectionPage }>(
    "/v1/admin/forwarder/connections",
    { method: "GET", params, ...(options || {}) }
  );
}
