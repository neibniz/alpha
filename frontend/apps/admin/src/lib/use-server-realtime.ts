import { getAuthorizationToken } from "@workspace/ui/lib/auth-token";
import { getApiBaseURL, getApiPrefix } from "@workspace/ui/lib/runtime-config";
import { useEffect, useRef } from "react";
import {
  normalizeRealtimeServerIDs,
  parseRealtimeServerMessage,
} from "./server-realtime-data";

const RECONNECT_DELAY_MS = 3000;

function buildRealtimeWsUrl(serverIDs: readonly number[]) {
  const base = getApiBaseURL() || window.location.origin;
  const prefix = getApiPrefix();
  const url = new URL(`${prefix}/v1/admin/server/realtime/ws`, base);
  if (serverIDs.length > 0) {
    url.searchParams.set("server_ids", serverIDs.join(","));
  }
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

export function useServerRealtime(
  serverIDs: readonly number[],
  onStatus: (serverId: number, status: Partial<API.ServerStatus>) => void,
  enabled = true
) {
  const onStatusRef = useRef(onStatus);
  onStatusRef.current = onStatus;
  const subscriptionKey = normalizeRealtimeServerIDs(serverIDs).join(",");

  useEffect(() => {
    if (!(enabled && getAuthorizationToken())) return;
    const subscribedServerIDs = subscriptionKey
      ? subscriptionKey.split(",").map(Number)
      : [];
    let closed = false;
    let socket: WebSocket | undefined;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;

    const connect = () => {
      const token = getAuthorizationToken();
      if (closed || !token) return;
      socket = new WebSocket(buildRealtimeWsUrl(subscribedServerIDs), [
        "alpha-auth",
        token,
      ]);
      socket.onmessage = (event) => {
        for (const update of parseRealtimeServerMessage(event.data)) {
          onStatusRef.current(update.serverId, update.status);
        }
      };
      socket.onclose = () => {
        if (!closed) reconnectTimer = setTimeout(connect, RECONNECT_DELAY_MS);
      };
      socket.onerror = () => socket?.close();
    };

    connect();
    return () => {
      closed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [enabled, subscriptionKey]);
}
