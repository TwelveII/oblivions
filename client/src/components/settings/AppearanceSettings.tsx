/** AppearanceSettings — Theme selection grid with color swatch previews. */

import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useSettingsStore } from "../../stores/settingsStore";
import { useToastStore } from "../../stores/toastStore";
import { THEMES, THEME_ORDER, type ThemeId } from "../../styles/themes";
import { isElectron, isCapacitor } from "../../utils/constants";

function AppearanceSettings() {
  const { t } = useTranslation("settings");
  const themeId = useSettingsStore((s) => s.themeId);
  const setTheme = useSettingsStore((s) => s.setTheme);
  const blurEnabled = useSettingsStore((s) => s.blurEnabled);
  const setBlurEnabled = useSettingsStore((s) => s.setBlurEnabled);
  const transparentBackground = useSettingsStore((s) => s.transparentBackground);
  const setTransparentBackground = useSettingsStore((s) => s.setTransparentBackground);

  // Track initial value to detect change — restart needed in either direction
  const initialTransparent = useMemo(() => transparentBackground, []);
  function handleSelectTheme(id: ThemeId) {
    setTheme(id);
  }

  return (
    <div>
      {/* Blur is force-disabled on mobile (too costly on mobile GPUs), so the toggle is hidden there. */}
      {!isCapacitor() && (
        <>
          <h2 className="settings-section-title">{t("blurTitle")}</h2>
          <p className="theme-section-desc">{t("blurDescription")}</p>
          <label className="settings-toggle-row">
            <span>{t("blurTitle")}</span>
            <button
              className={`ub-switch${blurEnabled ? " active" : ""}`}
              onClick={() => setBlurEnabled(!blurEnabled)}
              role="switch"
              aria-checked={blurEnabled}
              type="button"
            >
              <span className="ub-switch-thumb" />
            </button>
          </label>
        </>
      )}

      {isElectron() && (
        <>
          <h2 className="settings-section-title" style={{ marginTop: 24 }}>{t("transparentTitle")}</h2>
          <p className="theme-section-desc">{t("transparentDescription")}</p>
          <label className="settings-toggle-row">
            <span>{t("transparentEnable")}</span>
            <button
              className={`ub-switch${transparentBackground ? " active" : ""}`}
              onClick={() => setTransparentBackground(!transparentBackground)}
              role="switch"
              aria-checked={transparentBackground}
              type="button"
            >
              <span className="ub-switch-thumb" />
            </button>
          </label>
          {transparentBackground !== initialTransparent && (
            <p className="theme-section-desc" style={{ color: "var(--yellow)", marginTop: 4 }}>{t("transparentRestart")}</p>
          )}
        </>
      )}

      <h2 className="settings-section-title" style={{ marginTop: 24 }}>{t("themeTitle")}</h2>
      <p className="theme-section-desc">{t("themeDescription")}</p>

      <div className="theme-grid">
        {THEME_ORDER.map((id) => {
          const theme = THEMES[id];
          const isActive = id === themeId;

          return (
            <button
              key={id}
              className={`theme-card${isActive ? " theme-card-active" : ""}`}
              onClick={() => handleSelectTheme(id)}
              type="button"
            >
              {/* Color swatch preview */}
              <div className="theme-swatches">
                {theme.swatches.map((color, i) => (
                  <span
                    key={i}
                    className="theme-swatch"
                    style={{ background: color }}
                  />
                ))}
              </div>

              {/* Theme info */}
              <span className="theme-card-name">{t(theme.nameKey)}</span>
              <span className="theme-card-desc">{t(theme.descKey)}</span>

              {/* Active indicator */}
              {isActive && <span className="theme-card-check">&#10003;</span>}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export default AppearanceSettings;
