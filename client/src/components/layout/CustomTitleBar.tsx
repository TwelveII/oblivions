/**
 * CustomTitleBar — Electron frameless window controls.
 *
 * Provides minimize, maximize/restore, close buttons when frame:false.
 * Draggable via -webkit-app-region: drag on the bar itself.
 * Close button sends to tray (isQuitting=false → window hides).
 * Shows "Update" button when a new version is downloaded and ready.
 */

import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAuthStore } from "../../stores/authStore";
import { useSettingsStore } from "../../stores/settingsStore";
import InfoModal from "../shared/InfoModal";

function CustomTitleBar() {
  const { t } = useTranslation("settings");
  const { t: tc } = useTranslation("common");
  const [isMaximized, setIsMaximized] = useState(false);
  const [updateReady, setUpdateReady] = useState(false);
  const [infoOpen, setInfoOpen] = useState(false);
  const user = useAuthStore((s) => s.user);
  const openSettings = useSettingsStore((s) => s.openSettings);

  useEffect(() => {
    const api = window.electronAPI;
    if (!api) return;

    api.onMaximizedChange((val) => setIsMaximized(val));
    api.onUpdateDownloaded(() => setUpdateReady(true));

    return () => {
      api.removeMaximizedListener();
    };
  }, []);

  function handleMinimize() {
    window.electronAPI?.minimizeWindow();
  }

  function handleMaximize() {
    window.electronAPI?.maximizeWindow();
  }

  function handleClose() {
    window.electronAPI?.closeWindow();
  }

  function handleUpdate() {
    window.electronAPI?.installUpdate();
  }

  return (
    <div className="custom-titlebar">
      <div className="titlebar-drag-region" />

      {updateReady && (
        <button className="titlebar-update-btn" onClick={handleUpdate}>
          {t("updateRestart")}
        </button>
      )}

      <div className="titlebar-controls">
        <button
          className="titlebar-icon-btn"
          onClick={() => setInfoOpen(true)}
          title={tc("appInfo")}
          aria-label={tc("appInfo")}
        >
          <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
          </svg>
        </button>

        {user && (
          <button
            className="titlebar-text-btn"
            onClick={() => openSettings("feedback")}
          >
            {t("feedback")}
          </button>
        )}

        <button className="titlebar-btn" onClick={handleMinimize}>
          <svg width="10" height="1" viewBox="0 0 10 1">
            <rect width="10" height="1" fill="currentColor" />
          </svg>
        </button>

        <button className="titlebar-btn" onClick={handleMaximize}>
          {isMaximized ? (
            // Restore icon (overlapping squares)
            <svg width="10" height="10" viewBox="0 0 10 10">
              <path
                fill="none"
                stroke="currentColor"
                strokeWidth="1"
                d="M3 1h6v6h-1M1 3h6v6H1z"
              />
            </svg>
          ) : (
            <svg width="10" height="10" viewBox="0 0 10 10">
              <rect
                x="0.5"
                y="0.5"
                width="9"
                height="9"
                fill="none"
                stroke="currentColor"
                strokeWidth="1"
              />
            </svg>
          )}
        </button>

        {/* Close → sends to tray */}
        <button className="titlebar-btn close" onClick={handleClose}>
          <svg width="10" height="10" viewBox="0 0 10 10">
            <path
              stroke="currentColor"
              strokeWidth="1.2"
              d="M1 1l8 8M9 1l-8 8"
            />
          </svg>
        </button>
      </div>

      <InfoModal isOpen={infoOpen} onClose={() => setInfoOpen(false)} />
    </div>
  );
}

export default CustomTitleBar;
