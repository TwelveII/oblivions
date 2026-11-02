/**
 * ReleaseNotes — collapsible list of releases (newest first), latest expanded.
 * Shared by the in-app info modal and the public landing page; renders the
 * structured data from data/releaseNotes (no markdown dependency).
 */

import { useState } from "react";
import { releaseNotes } from "../../data/releaseNotes";

function formatDate(d: string | null): string {
  if (!d) return "";
  const date = new Date(`${d}T00:00:00`);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" });
}

function ReleaseNotes() {
  // Latest release expanded by default; each release toggles independently.
  const [open, setOpen] = useState<Set<string>>(
    () => new Set(releaseNotes[0] ? [releaseNotes[0].version] : []),
  );

  const toggle = (version: string) =>
    setOpen((prev) => {
      const next = new Set(prev);
      if (next.has(version)) next.delete(version);
      else next.add(version);
      return next;
    });

  return (
    <div className="release-notes">
      {releaseNotes.map((release) => {
        const isOpen = open.has(release.version);
        return (
          <div key={release.version} className={`rn-release${isOpen ? " open" : ""}`}>
            <button
              type="button"
              className="rn-release-header"
              onClick={() => toggle(release.version)}
              aria-expanded={isOpen}
            >
              <span className="rn-chevron" aria-hidden="true">{isOpen ? "▾" : "▸"}</span>
              <span className="rn-version">{release.version}</span>
              {release.date && <span className="rn-date">{formatDate(release.date)}</span>}
            </button>

            {isOpen && (
              <div className="rn-body">
                {release.sections.map((section, si) => (
                  <div key={si} className="rn-section">
                    {section.heading && <h4 className="rn-section-heading">{section.heading}</h4>}
                    <ul className="rn-items">
                      {section.items.map((item, ii) => (
                        <li key={ii} className="rn-item">
                          {item.scope && <span className="rn-scope">{item.scope}</span>}
                          <span className="rn-text">{item.text}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

export default ReleaseNotes;
