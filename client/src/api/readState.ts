/**
 * Read State API — unread message tracking, server-scoped.
 */

import { apiClient } from "./client";

export type UnreadInfo = {
  channel_id: string;
  unread_count: number;
  last_mention_seen_at: string | null;
  last_mention_seen_message_id: string | null;
};

/** Marks a channel as read up to the given message ID. */
export function markRead(serverId: string, channelId: string, messageId: string) {
  return apiClient<{ message: string }>(`/servers/${serverId}/channels/${channelId}/read`, {
    method: "POST",
    body: { message_id: messageId },
  });
}

/** Returns unread counts for all channels in a server (channels with unread > 0 OR mention watermark set). */
export function getUnreadCounts(serverId: string) {
  return apiClient<UnreadInfo[]>(`/servers/${serverId}/channels/unread`);
}

/** Marks all channels in a server as read. */
export function markAllRead(serverId: string) {
  return apiClient<{ message: string }>(`/servers/${serverId}/channels/read-all`, {
    method: "POST",
  });
}

/** Advances the per-channel mention-seen watermark to the given mention message. */
export function markMentionSeen(serverId: string, channelId: string, mentionMessageId: string) {
  return apiClient<{ message: string }>(`/servers/${serverId}/channels/${channelId}/read/mentions`, {
    method: "POST",
    body: { mention_message_id: mentionMessageId },
  });
}
