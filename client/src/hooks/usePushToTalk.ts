/**
 * usePushToTalk — Push-to-talk key listener.
 *
 * Electron needs BOTH paths because the native global hook only covers the
 * unfocused case:
 * - uIOhook (native): fires when the app is in the background (e.g. in a game).
 *   Windows does not surface low-level hook events to the app's own foreground
 *   window, so this path goes silent while the app is focused.
 * - document keydown/keyup: covers the focused case — the key reaches the
 *   renderer as a normal DOM event only while the window has focus.
 * The two overlap during focus transitions; isPressedRef dedupes them.
 *
 * Browser: only the document path (no native hook available).
 *
 * Document path guards:
 * - Focus guard: disabled when typing in input/textarea/contentEditable
 * - Repeat filter: ignores e.repeat (browser auto-repeat on key hold)
 * - Mode guard: no-op if inputMode !== "push_to_talk"
 * - Connection guard: no-op if not in a voice channel
 * - Blur guard: releases mic on window blur (alt-tab) — the native path resumes
 */

import { useEffect, useRef } from "react";
import { useVoiceStore } from "../stores/voiceStore";
import { isMouseBinding } from "../stores/slices/voiceSettingsSlice";
import { isElectron } from "../utils/constants";

type UsePushToTalkParams = {
  setMicEnabled: (enabled: boolean) => void;
};

export function usePushToTalk({ setMicEnabled }: UsePushToTalkParams): void {
  const inputMode = useVoiceStore((s) => s.inputMode);
  const pttKey = useVoiceStore((s) => s.pttKey);
  const currentVoiceChannelId = useVoiceStore((s) => s.currentVoiceChannelId);

  // Ref — no re-render needed, side-effect only
  const isPressedRef = useRef(false);

  // ─── Electron: global PTT via uIOhook IPC ───
  useEffect(() => {
    if (!isElectron()) return;
    if (inputMode !== "push_to_talk" || !currentVoiceChannelId) return;

    const api = window.electronAPI!;

    // Remove stale listeners from previous sessions
    api.removePTTListeners();

    api.onPTTGlobalDown(() => {
      // Keyboard: focused → the document path handles it (native keyboard hook
      // is unreliable in the foreground). Mouse: always native — DOM mouse
      // events need the pointer over the window, so the native hook is the
      // reliable source in both focus states.
      if (!isMouseBinding(pttKey) && document.hasFocus()) return;
      if (isPressedRef.current) return;
      const { isMuted, isServerMuted } = useVoiceStore.getState();
      if (isMuted || isServerMuted) return;
      isPressedRef.current = true;
      setMicEnabled(true);
    });

    // Up is never focus-gated — always release so the mic can't get stuck on.
    api.onPTTGlobalUp(() => {
      if (!isPressedRef.current) return;
      isPressedRef.current = false;
      setMicEnabled(false);
    });

    // Register the key with the main process
    api.registerPTTShortcut(pttKey);

    return () => {
      api.unregisterPTTShortcut();
      api.removePTTListeners();

      if (isPressedRef.current) {
        isPressedRef.current = false;
        setMicEnabled(false);
      }
    };
  }, [inputMode, pttKey, currentVoiceChannelId, setMicEnabled]);

  // ─── Document-level keydown/keyup (focus required) ───
  // Runs in Electron too: covers the focused case the native hook misses.
  // Mouse bindings are handled solely by the native path above.
  useEffect(() => {
    if (inputMode !== "push_to_talk" || !currentVoiceChannelId) return;
    if (isMouseBinding(pttKey)) return;

    function isTextInput(el: Element | null): boolean {
      if (!el) return false;
      const tag = el.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA") return true;
      if ((el as HTMLElement).isContentEditable) return true;
      return false;
    }

    function handleKeyDown(e: KeyboardEvent) {
      if (e.repeat) return;
      if (e.code !== pttKey) return;
      if (isTextInput(document.activeElement)) return;
      if (isPressedRef.current) return;
      const { isMuted, isServerMuted } = useVoiceStore.getState();
      if (isMuted || isServerMuted) return;

      isPressedRef.current = true;
      setMicEnabled(true);
    }

    function handleKeyUp(e: KeyboardEvent) {
      if (e.code !== pttKey) return;
      if (!isPressedRef.current) return;

      isPressedRef.current = false;
      setMicEnabled(false);
    }

    // Release mic on window blur (e.g. alt-tab)
    function handleBlur() {
      if (isPressedRef.current) {
        isPressedRef.current = false;
        setMicEnabled(false);
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    document.addEventListener("keyup", handleKeyUp);
    window.addEventListener("blur", handleBlur);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("keyup", handleKeyUp);
      window.removeEventListener("blur", handleBlur);

      // Ensure mic is off when exiting PTT mode
      if (isPressedRef.current) {
        isPressedRef.current = false;
        setMicEnabled(false);
      }
    };
  }, [inputMode, pttKey, currentVoiceChannelId, setMicEnabled]);
}
