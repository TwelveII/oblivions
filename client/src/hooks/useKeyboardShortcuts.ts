/**
 * useKeyboardShortcuts — Global keyboard shortcuts hook.
 *
 * Mute / deafen bindings are read from voiceStore so the user can rebind
 * them in Settings → Voice. Ctrl+K (quick switcher) stays hardcoded.
 *
 * Mute / deafen need both paths in Electron, split by focus:
 * - Native global hook (uIOhook): fires when the app is unfocused (e.g. in a
 *   game). It's unreliable while the app owns the foreground — events go
 *   missing and its modifier mask can stick (a missed Ctrl/Shift keyup makes a
 *   bare key match a Ctrl/Shift binding). So native is ignored while focused.
 * - Document keydown: handles the focused case with reliable browser modifiers.
 * The two are mutually exclusive by focus, so no double-toggle.
 *
 * Singleton — called once in AppLayout.
 */

import { useEffect } from "react";
import { useUIStore } from "../stores/uiStore";
import { useVoiceStore } from "../stores/voiceStore";
import { isMouseBinding, type ShortcutBinding } from "../stores/slices/voiceSettingsSlice";
import { isElectron } from "../utils/constants";

type KeyboardShortcutActions = {
  toggleMute: () => void;
  toggleDeafen: () => void;
};

function matchesBinding(e: KeyboardEvent, binding: ShortcutBinding): boolean {
  return (
    e.code === binding.code &&
    e.ctrlKey === binding.ctrl &&
    e.shiftKey === binding.shift &&
    e.altKey === binding.alt
  );
}

export function useKeyboardShortcuts({ toggleMute, toggleDeafen }: KeyboardShortcutActions) {
  const muteShortcut = useVoiceStore((s) => s.muteShortcut);
  const deafenShortcut = useVoiceStore((s) => s.deafenShortcut);

  // ─── Document-level: Ctrl+K always; mute/deafen when focused ───
  // Runs in Electron too — covers the focused case the native hook misses.
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const target = e.target as HTMLElement;
      const isInputFocused =
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.isContentEditable;

      // Ctrl+K — Quick Switcher (works in input too)
      if (e.ctrlKey && !e.shiftKey && !e.altKey && e.code === "KeyK") {
        e.preventDefault();
        useUIStore.getState().toggleQuickSwitcher();
        return;
      }

      if (isInputFocused) return;

      const { muteShortcut: mute, deafenShortcut: deafen } = useVoiceStore.getState();
      if (matchesBinding(e, mute)) {
        e.preventDefault();
        toggleMute();
        return;
      }
      if (matchesBinding(e, deafen)) {
        e.preventDefault();
        toggleDeafen();
        return;
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [toggleMute, toggleDeafen]);

  // ─── Electron: native global mute/deafen via uIOhook IPC ───
  useEffect(() => {
    if (!isElectron()) return;
    const api = window.electronAPI!;

    api.removeMuteListeners();
    api.removeDeafenListeners();
    // Keyboard: ignore native toggles while focused — the document path handles
    // that case with reliable modifiers (native's mask can be stale in the
    // foreground). Mouse: always native (DOM mouse needs the pointer over the
    // window), so don't focus-gate it.
    api.onMuteGlobalToggle(() => {
      if (isMouseBinding(muteShortcut.code) || !document.hasFocus()) toggleMute();
    });
    api.onDeafenGlobalToggle(() => {
      if (isMouseBinding(deafenShortcut.code) || !document.hasFocus()) toggleDeafen();
    });

    api.registerMuteShortcut(muteShortcut);
    api.registerDeafenShortcut(deafenShortcut);

    return () => {
      api.unregisterMuteShortcut();
      api.unregisterDeafenShortcut();
      api.removeMuteListeners();
      api.removeDeafenListeners();
    };
  }, [toggleMute, toggleDeafen, muteShortcut, deafenShortcut]);
}
