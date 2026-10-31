/**
 * Calls API — ICE servers (STUN + TURN relay) for P2P calls.
 */

import { apiClient } from "./client";

// STUN-only fallback. Used when the backend fetch fails (offline, 403, 429) so a
// call still attempts to connect — it just loses the relay fallback.
const FALLBACK_ICE_SERVERS: RTCIceServer[] = [
  { urls: "stun:stun.l.google.com:19302" },
  { urls: "stun:stun1.l.google.com:19302" },
];

// Bound the wait so a hung server/proxy can't block the call forever. The timer
// aborts the actual request (not just our wait), so a timed-out fetch doesn't keep
// running in the background and mint an unneeded credential / consume rate limit.
const ICE_FETCH_TIMEOUT_MS = 4000;

// Fetches the backend ICE list, or null on any failure/timeout. Never throws and
// never blocks beyond ICE_FETCH_TIMEOUT_MS. Callers decide the failure policy.
async function fetchIceServersOrNull(): Promise<RTCIceServer[] | null> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), ICE_FETCH_TIMEOUT_MS);
  try {
    const res = await apiClient<{ ice_servers: RTCIceServer[] }>("/calls/ice-servers", {
      signal: controller.signal,
    });
    if (res.success && res.data?.ice_servers?.length) {
      return res.data.ice_servers;
    }
  } catch (err) {
    console.warn("[p2p] ICE server fetch threw:", err);
  } finally {
    clearTimeout(timer);
  }
  return null;
}

/**
 * Fetches the ICE server list (STUN + TURN with fresh short-lived credentials)
 * for the start of a P2P call. Must be called once the call is "active" — the
 * backend gates this on an accepted call. On any failure/timeout it returns
 * STUN-only so the call is not held up by TURN.
 */
export async function fetchIceServers(): Promise<RTCIceServer[]> {
  const servers = await fetchIceServersOrNull();
  if (servers) return servers;
  console.warn("[p2p] ICE server fetch failed/timed out, falling back to STUN-only");
  return FALLBACK_ICE_SERVERS;
}

/**
 * Recovery variant: returns null on failure instead of STUN-only. During an
 * ICE-restart the caller must NOT downgrade its config to STUN-only — that would
 * strip the TURN servers at the exact moment a relayed reconnect is needed. On
 * null, keep the existing configuration.
 */
export async function fetchIceServersForRecovery(): Promise<RTCIceServer[] | null> {
  return fetchIceServersOrNull();
}
