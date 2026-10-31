/**
 * Release notes data — parsed from the repo-root release-notes/*.md files.
 *
 * Single source of truth: each release is one markdown file (the same files the
 * release CI reads). New releases appear automatically — the glob picks them up.
 * Parsed at load time into a structured shape so the UI renders natively (no
 * markdown dependency); a malformed file degrades to an empty section, not a crash.
 *
 * File format (per release):
 *   <!-- date: YYYY-MM-DD -->   (optional)
 *   ## vX.Y.Z                   (title — version is taken from the filename)
 *   ### Section Heading
 *   - **scope**: description
 */

export type ReleaseItem = { scope: string | null; text: string };
export type ReleaseSection = { heading: string | null; items: ReleaseItem[] };
export type ReleaseNote = { version: string; date: string | null; sections: ReleaseSection[] };

const DATE_RE = /<!--\s*date:\s*([\d-]+)\s*-->/i;
const ITEM_RE = /^-\s+(?:\*\*(.+?)\*\*:\s*)?(.*)$/;

function parseRelease(version: string, raw: string): ReleaseNote {
  let date: string | null = null;
  const sections: ReleaseSection[] = [];
  let current: ReleaseSection | null = null;

  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    const dm = trimmed.match(DATE_RE);
    if (dm) {
      date = dm[1];
      continue;
    }
    if (trimmed.startsWith("## ")) continue; // version title — derived from filename

    if (trimmed.startsWith("### ")) {
      current = { heading: trimmed.slice(4).trim(), items: [] };
      sections.push(current);
      continue;
    }

    const im = trimmed.match(ITEM_RE);
    if (im) {
      if (!current) {
        // Bullets before any heading (e.g. backfilled tag notes) → headingless section.
        current = { heading: null, items: [] };
        sections.push(current);
      }
      current.items.push({ scope: im[1] ?? null, text: im[2].trim() });
    }
  }

  return { version, date, sections };
}

/** [major, minor, patch] for semver-descending sort. */
function parseVersion(v: string): [number, number, number] {
  const p = v.replace(/^v/, "").split(".").map((n) => parseInt(n, 10) || 0);
  return [p[0] ?? 0, p[1] ?? 0, p[2] ?? 0];
}

// Repo-root release-notes/ lives one level above client/ (see vite.config fs.allow).
const modules = import.meta.glob("../../../release-notes/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

/** All releases, newest first. */
export const releaseNotes: ReleaseNote[] = Object.entries(modules)
  .map(([path, raw]) => {
    const version = path.split("/").pop()!.replace(/\.md$/, "");
    return parseRelease(version, raw);
  })
  .sort((a, b) => {
    const av = parseVersion(a.version);
    const bv = parseVersion(b.version);
    for (let i = 0; i < 3; i++) {
      if (bv[i] !== av[i]) return bv[i] - av[i];
    }
    return 0;
  });
