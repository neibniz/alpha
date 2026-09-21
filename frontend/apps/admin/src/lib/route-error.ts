function record(value: unknown): Record<string, unknown> {
  return value !== null && typeof value === "object"
    ? (value as Record<string, unknown>)
    : {};
}

export function classifyRouteError(error: unknown) {
  const details = record(error);
  const message = typeof details.message === "string" ? details.message : "";
  if (
    /Failed to fetch dynamically imported module|error loading dynamically imported module|Importing a module script failed|Loading (?:CSS )?chunk .+ failed|Unable to preload CSS/i.test(
      message
    )
  ) {
    return "resource";
  }
  const response = record(details.response);
  const status = response.status ?? details.status;
  if (typeof status === "number" && status >= 500) return "server";
  if (status === 401 || status === 403) return "session";
  if (
    details.code === "ERR_NETWORK" ||
    details.code === "ECONNABORTED" ||
    details.code === "ETIMEDOUT" ||
    /^(?:Failed to fetch|Network Error|Load failed)$|NetworkError when attempting to fetch resource|timeout of \d+ms exceeded/i.test(
      message
    )
  ) {
    return "network";
  }
  return "application";
}

export const routeErrorDescriptions = {
  resource:
    "页面资源加载失败，可能是发布后旧资源失效或资源请求被阻止。请刷新页面重新加载。",
  server: "服务端暂时无法处理请求，请稍后重试。",
  session: "无法验证管理员会话或权限，请刷新页面重新检查登录状态。",
  network: "请求未能完成，请检查网络、API 地址及跨域配置后重试。",
  application:
    "页面运行或数据处理发生异常，不一定是网络问题。若重试后仍然出现，请记录页面地址和触发操作。",
} as const;
