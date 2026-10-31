/**
 * useIdleDetection — User inactivity detection hook.
 *
 * Electron: polls OS-level idle time (via powerMonitor) so activity in other
 * apps/games keeps the user online.
 * Web: falls back to DOM activity events (mouse, keyboard) — only sees in-tab input.
 *
 * Skipped when manualStatus is not "online" (DND, Idle, Invisible).
 * Skipped when user is in a voice channel.
 *
 * Singleton — called once in AppLayout.
 */

import { useEffect, useRef } from "react";
import { IDLE_TIMEOUT, ACTIVITY_EVENTS, isElectron } from "../utils/constants";
import { useAuthStore } from "../stores/authStore";
import { useVoiceStore } from "../stores/voiceStore";
import type { UserStatus } from "../types";

type UseIdleDetectionParams = {
  sendPresenceUpdate: (status: UserStatus, isAuto?: boolean) => void;
};

const ELECTRON_POLL_MS = 30_000;

export function useIdleDetection({ sendPresenceUpdate }: UseIdleDetectionParams) {
  /** useRef instead of useState — only read in event handlers, no re-render needed */
  const isIdleRef = useRef(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    // ─── Electron: poll OS-level idle time ───
    const electronGetIdle = isElectron() ? window.electronAPI?.getSystemIdleTime : undefined;
    if (electronGetIdle) {
      // Re-capture in a non-nullable const so the async closure keeps the narrowed type.
      const getIdle = electronGetIdle;
      let stopped = false;

      async function tick() {
        if (stopped) return;
        if (useAuthStore.getState().manualStatus !== "online") return;

        let idleSec: number;
        try {
          idleSec = await getIdle();
        } catch {
          return;
        }
        if (stopped) return; // re-check: component may have unmounted during await

        const inVoice = useVoiceStore.getState().currentVoiceChannelId !== null;
        const shouldBeIdle = !inVoice && idleSec * 1000 >= IDLE_TIMEOUT;

        if (shouldBeIdle && !isIdleRef.current) {
          isIdleRef.current = true;
          sendPresenceUpdate("idle", true);
        } else if (!shouldBeIdle && isIdleRef.current) {
          isIdleRef.current = false;
          sendPresenceUpdate("online", true);
        }
      }

      // Window focus → immediate check so coming back is snappy
      function onFocus() { void tick(); }
      window.addEventListener("focus", onFocus);

      void tick();
      const interval = setInterval(tick, ELECTRON_POLL_MS);

      return () => {
        stopped = true;
        clearInterval(interval);
        window.removeEventListener("focus", onFocus);
      };
    }

    // ─── Web fallback: DOM activity events (only see in-tab input) ───
    function resetTimer() {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }

      const manual = useAuthStore.getState().manualStatus;
      if (manual !== "online") {
        return;
      }

      if (isIdleRef.current) {
        isIdleRef.current = false;
        sendPresenceUpdate("online", true);
      }

      timerRef.current = setTimeout(function idleCheck() {
        const currentManual = useAuthStore.getState().manualStatus;
        if (currentManual !== "online") {
          return;
        }

        const inVoice = useVoiceStore.getState().currentVoiceChannelId !== null;
        if (inVoice) {
          timerRef.current = setTimeout(idleCheck, IDLE_TIMEOUT);
          return;
        }

        isIdleRef.current = true;
        sendPresenceUpdate("idle", true);
      }, IDLE_TIMEOUT);
    }

    for (const event of ACTIVITY_EVENTS) {
      window.addEventListener(event, resetTimer, { passive: true });
    }

    resetTimer();

    return () => {
      for (const event of ACTIVITY_EVENTS) {
        window.removeEventListener(event, resetTimer);
      }
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [sendPresenceUpdate]);
}
