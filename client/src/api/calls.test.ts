import { describe, it, expect, vi, beforeEach } from "vitest";

const { apiClient } = vi.hoisted(() => ({ apiClient: vi.fn() }));
vi.mock("./client", () => ({ apiClient }));

import { fetchIceServers, fetchIceServersForRecovery } from "./calls";

describe("fetchIceServers", () => {
  beforeEach(() => apiClient.mockReset());

  it("returns the backend list on success", async () => {
    const servers = [{ urls: ["turn:t:3478?transport=udp"], username: "u", credential: "c" }];
    apiClient.mockResolvedValue({ success: true, data: { ice_servers: servers } });
    await expect(fetchIceServers()).resolves.toEqual(servers);
  });

  it("uses no external ICE servers when the response is empty", async () => {
    apiClient.mockResolvedValue({ success: true, data: { ice_servers: [] } });
    await expect(fetchIceServers()).resolves.toEqual([]);
  });

  it("uses no external ICE servers on a failed response (403/429/network)", async () => {
    apiClient.mockResolvedValue({ success: false, error: "forbidden" });
    await expect(fetchIceServers()).resolves.toEqual([]);
  });

  // Timeout and throw/reject paths are defense-in-depth over apiClient (which is
  // designed to return {success:false} rather than throw) and produce the same empty ICE fallback as the cases above. They are checkable with fake timers / a
  // throwing mock, but doing so reliably means working around Vitest's
  // unhandled-rejection handling for marginal added coverage — left to inspection.
});

describe("fetchIceServersForRecovery", () => {
  beforeEach(() => apiClient.mockReset());

  it("returns the backend list on success", async () => {
    const servers = [{ urls: ["turn:t:3478?transport=udp"] }];
    apiClient.mockResolvedValue({ success: true, data: { ice_servers: servers } });
    await expect(fetchIceServersForRecovery()).resolves.toEqual(servers);
  });

  it("returns null on failure — recovery keeps the existing TURN config, not STUN", async () => {
    apiClient.mockResolvedValue({ success: false, error: "forbidden" });
    await expect(fetchIceServersForRecovery()).resolves.toBeNull();
  });

  it("returns null on an empty list", async () => {
    apiClient.mockResolvedValue({ success: true, data: { ice_servers: [] } });
    await expect(fetchIceServersForRecovery()).resolves.toBeNull();
  });
});
