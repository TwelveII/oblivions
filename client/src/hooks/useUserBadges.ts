/**
 * useUserBadges — Returns cached badges for a user, fetching if not yet loaded.
 *
 * Reads from badgeStore.userBadgesMap. On first call for a given userId,
 * triggers a fetch. Subsequent renders return the cached value instantly.
 *
 * Pass `enabled: false` to skip the fetch entirely (e.g. for deleted users
 * whose badges are not rendered anyway). Hooks must be called unconditionally,
 * so we can't drop the call — but we can drop the fetch.
 */

import { useEffect } from "react";
import { useBadgeStore } from "../stores/badgeStore";
import type { UserBadge } from "../types";

const EMPTY: UserBadge[] = [];

/** Set of user IDs currently being fetched (prevents duplicate requests). */
const fetchingSet = new Set<string>();

export function useUserBadges(
  userId: string | undefined,
  enabled: boolean = true,
): UserBadge[] {
  const cached = useBadgeStore((s) =>
    userId ? s.userBadgesMap[userId] : undefined
  );

  useEffect(() => {
    if (!enabled || !userId || cached !== undefined || fetchingSet.has(userId)) return;
    fetchingSet.add(userId);
    useBadgeStore
      .getState()
      .fetchUserBadges(userId)
      .finally(() => fetchingSet.delete(userId));
  }, [userId, cached, enabled]);

  return cached ?? EMPTY;
}
