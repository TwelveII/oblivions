/**
 * Help article body loader — lazy, per-language, with English fallback.
 *
 * Globs every language's markdown but as lazy loaders, so only the chunk for the
 * slug actually opened (in the active language) is fetched. If the active language
 * has no translation for a slug, the English body is loaded instead.
 */

const loaders = import.meta.glob("./content/*/*.md", {
  query: "?raw",
  import: "default",
}) as Record<string, () => Promise<string>>;

const keyFor = (lang: string, slug: string) => `./content/${lang}/${slug}.md`;

// Eagerly map every bundled help image (basename → hashed URL). Markdown references
// images as `assets/<file>`; the renderer resolves them through this map. Bundling
// (not server-hosting) keeps help offline-capable and versioned with the app.
const assetUrls = import.meta.glob(["./content/assets/*", "!./content/assets/*.md"], {
  eager: true,
  query: "?url",
  import: "default",
}) as Record<string, string>;

const assetByName = new Map<string, string>(
  Object.entries(assetUrls).map(([path, url]) => [path.split("/").pop() as string, url]),
);

/** Resolves a markdown image src ("assets/foo.webp") to its bundled URL, or null if absent. */
export function resolveAsset(src: string | undefined): string | null {
  if (!src) return null;
  const name = src.replace(/^\.?\/?assets\//, "").split("/").pop();
  return name ? assetByName.get(name) ?? null : null;
}

export type LoadedArticle = { slug: string; body: string; translated: boolean };

/** Loads a slug's markdown body for `lang`, falling back to English. Null if neither exists. */
export async function loadArticleBody(lang: string, slug: string): Promise<LoadedArticle | null> {
  const active = loaders[keyFor(lang, slug)];
  if (active) return { slug, body: await active(), translated: true };

  const english = loaders[keyFor("en", slug)];
  if (english) return { slug, body: await english(), translated: lang === "en" };

  return null;
}

// Cache the full body set per language so search loads each language's bodies once.
const bodyCache = new Map<string, Map<string, string>>();

/** Loads every article body for `lang` (with English fallback), cached. Powers search. */
export async function loadAllBodies(lang: string, slugs: string[]): Promise<Map<string, string>> {
  const cached = bodyCache.get(lang);
  if (cached) return cached;

  const map = new Map<string, string>();
  await Promise.all(
    slugs.map(async (slug) => {
      const article = await loadArticleBody(lang, slug);
      if (article) map.set(slug, article.body);
    }),
  );
  bodyCache.set(lang, map);
  return map;
}
