/**
 * HelpCenter — the help content itself (Features guide + Release Notes), without
 * a modal wrapper. Reused by:
 *  - InfoModal (desktop title-bar icon, settings) → view="tabs" (both, tabbed)
 *  - Settings "Help" panel → view="tabs" inline
 *  - Landing "Help" / "Release Notes" links → view="features" / "releaseNotes" (single, no tabs)
 */

import { useState } from "react";
import { useTranslation } from "react-i18next";
import ReleaseNotes from "./ReleaseNotes";
import FeatureGuide from "../help/FeatureGuide";

export type HelpView = "tabs" | "features" | "releaseNotes";

function HelpCenter({ view = "tabs" }: { view?: HelpView }) {
  const { t } = useTranslation("common");
  const [tab, setTab] = useState<"features" | "releaseNotes">("features");

  if (view === "features") {
    return (
      <div className="info-modal">
        <div className="info-modal-body">
          <FeatureGuide />
        </div>
      </div>
    );
  }

  if (view === "releaseNotes") {
    return (
      <div className="info-modal">
        <div className="info-modal-body">
          <ReleaseNotes />
        </div>
      </div>
    );
  }

  return (
    <div className="info-modal">
      <div className="info-modal-tabs">
        <button
          type="button"
          className={`info-modal-tab${tab === "features" ? " active" : ""}`}
          onClick={() => setTab("features")}
        >
          {t("features")}
        </button>
        <button
          type="button"
          className={`info-modal-tab${tab === "releaseNotes" ? " active" : ""}`}
          onClick={() => setTab("releaseNotes")}
        >
          {t("releaseNotes")}
        </button>
      </div>

      <div className="info-modal-body">
        {tab === "features" ? <FeatureGuide /> : <ReleaseNotes />}
      </div>
    </div>
  );
}

export default HelpCenter;
