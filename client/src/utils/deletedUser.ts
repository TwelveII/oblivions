/**
 * Helpers for rendering soft-deleted/tombstone users consistently across surfaces.
 *
 * The backend keeps the user row (tombstone semantics) so messages.user_id remains
 * valid. Anywhere we render a user's identity (search results, pin lists, reply
 * preview, DM list, etc.) must check `deleted_at` and substitute the placeholder
 * label / drop the avatar — otherwise the original username/avatar leaks.
 */

import i18n from "../i18n";

export type DeletableAuthor = {
  display_name?: string | null;
  username?: string | null;
  avatar_url?: string | null;
  deleted_at?: string | null;
};

/** True when the author should render as "[deleted user]". */
export function isAuthorDeleted(author: DeletableAuthor | null | undefined): boolean {
  return !!author?.deleted_at;
}

/** Display name to show, with deleted users replaced by the localised placeholder. */
export function authorDisplayName(
  author: DeletableAuthor | null | undefined,
  fallback = "Unknown",
): string {
  if (!author) return fallback;
  if (author.deleted_at) {
    return i18n.t("deletedUser", { ns: "common", defaultValue: "Deleted user" });
  }
  return author.display_name ?? author.username ?? fallback;
}

/** Avatar URL, returning undefined for deleted users so the avatar component falls back. */
export function authorAvatarURL(
  author: DeletableAuthor | null | undefined,
): string | undefined {
  if (!author || author.deleted_at) return undefined;
  return author.avatar_url ?? undefined;
}
