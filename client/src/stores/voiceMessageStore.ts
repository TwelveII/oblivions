/**
 * voiceMessageStore — ephemeral voice channel chat messages, scoped per channel.
 *
 * Lifecycle:
 *  - load via listVoiceMessages on chat panel open
 *  - the sender applies its own send/edit/delete here; other clients get it from the WS handlers
 *  - wipeChannel called when voice_channel_timer_stop arrives (last person left)
 */

import { create } from "zustand";
import i18n from "../i18n";
import * as voiceMessageApi from "../api/voiceMessages";
import type { UploadOptions } from "../api/client";
import type { VoiceMessage, Message } from "../types";
import { useE2EEStore, canEncrypt } from "./e2eeStore";
import { useAuthStore } from "./authStore";
import { useToastStore } from "./toastStore";
import { encryptChannelMessage, decryptChannelMessages } from "../crypto/channelEncryption";
import { encodePayload } from "../crypto/e2eePayload";
import { encryptFilesForE2EE, encryptedEditPayload } from "./shared/messageUtils";

type VoiceMessageState = {
  /** channelId -> messages (chronological asc) */
  messagesByChannel: Record<string, VoiceMessage[]>;

  /** Replace the entire list for a channel (used after listVoiceMessages). */
  setForChannel: (channelId: string, messages: VoiceMessage[]) => Promise<void>;
  /** Append a single message from WS create event (dedup by id). */
  append: (message: VoiceMessage) => Promise<void>;
  /** Replace an existing message in place after edit. */
  update: (message: VoiceMessage) => Promise<void>;
  /** Remove a deleted message. */
  remove: (channelId: string, messageId: string) => void;
  /** Wipe all messages for a channel (fired when channel goes empty). */
  wipeChannel: (channelId: string) => void;

  // ─── Mutations ───
  // The sender's own message lands here instead of waiting for voice_message_create to come back.
  // Each applies its result through the handler the echo uses, so the two paths cannot drift, and
  // all three of those dedup by id — the echo replaying this is a no-op.
  send: (
    channelId: string,
    content: string,
    files?: File[],
    upload?: UploadOptions
  ) => Promise<boolean>;
  edit: (channelId: string, messageId: string, content: string) => Promise<boolean>;
  del: (channelId: string, messageId: string) => Promise<boolean>;
};

async function decryptVoiceMessage(message: VoiceMessage): Promise<VoiceMessage> {
  const asMessage: Message = {
    id: message.id,
    channel_id: message.channel_id,
    user_id: message.user_id,
    content: message.content,
    edited_at: message.edited_at,
    created_at: message.created_at,
    reply_to_id: null,
    referenced_message: null,
    author: message.author,
    attachments: (message.attachments ?? []).map((a) => ({
      id: a.id, message_id: a.voice_message_id, filename: a.filename,
      file_url: a.file_url, file_size: a.file_size, mime_type: a.mime_type,
    })),
    mentions: [],
    role_mentions: [],
    reactions: [],
    encryption_version: message.encryption_version ?? 0,
    ciphertext: message.ciphertext,
    sender_device_id: message.sender_device_id,
    e2ee_metadata: message.e2ee_metadata,
  };
  const [decrypted] = await decryptChannelMessages([asMessage]);
  return { ...message, content: decrypted.content, e2ee_file_keys: decrypted.e2ee_file_keys };
}

export const useVoiceMessageStore = create<VoiceMessageState>((set, get) => ({
  messagesByChannel: {},

  setForChannel: async (channelId, messages) => {
    const decrypted = await Promise.all(messages.map(decryptVoiceMessage));
    set((state) => ({
      messagesByChannel: { ...state.messagesByChannel, [channelId]: decrypted },
    }));
  },

  append: async (message) => {
    const decrypted = await decryptVoiceMessage(message);
    set((state) => {
      const existing = state.messagesByChannel[decrypted.channel_id] ?? [];
      if (existing.some((m) => m.id === decrypted.id)) return state;
      return {
        messagesByChannel: {
          ...state.messagesByChannel,
          [decrypted.channel_id]: [...existing, decrypted],
        },
      };
    });
  },

  update: async (message) => {
    const decrypted = await decryptVoiceMessage(message);
    set((state) => {
      const list = state.messagesByChannel[decrypted.channel_id];
      if (!list) return state;
      let changed = false;
      const next = list.map((m) => {
        if (m.id === decrypted.id) {
          changed = true;
          return decrypted;
        }
        return m;
      });
      if (!changed) return state;
      return {
        messagesByChannel: { ...state.messagesByChannel, [decrypted.channel_id]: next },
      };
    });
  },

  remove: (channelId, messageId) => {
    set((state) => {
      const list = state.messagesByChannel[channelId];
      if (!list) return state;
      const next = list.filter((m) => m.id !== messageId);
      if (next.length === list.length) return state;
      return {
        messagesByChannel: { ...state.messagesByChannel, [channelId]: next },
      };
    });
  },

  wipeChannel: (channelId) => {
    set((state) => {
      if (!(channelId in state.messagesByChannel)) return state;
      const next = { ...state.messagesByChannel };
      delete next[channelId];
      return { messagesByChannel: next };
    });
  },

  // ─── Mutations ───

  send: async (channelId, content, files, upload) => {
    const e2ee = useE2EEStore.getState();
    const userId = useAuthStore.getState().user?.id;
    if (!canEncrypt(e2ee) || !userId || !e2ee.localDeviceId) {
      useToastStore.getState().addToast("error", i18n.t("chat:encryptionRequired"));
      return false;
    }
    try {
      const encrypted = files?.length ? await encryptFilesForE2EE(files, upload?.signal) : undefined;
      const payload = encodePayload(content, encrypted?.metas);
      const senderKeyMessage = await encryptChannelMessage(channelId, userId, e2ee.localDeviceId, payload);
      const res = await voiceMessageApi.sendVoiceMessage(
        channelId, JSON.stringify(senderKeyMessage), e2ee.localDeviceId, JSON.stringify({}), encrypted?.files, upload
      );
      if (!res.success || !res.data) return false;
      await get().append({ ...res.data, content, e2ee_file_keys: encrypted?.metas });
      return true;
    } catch (error) {
      console.error("[voiceMessageStore] encrypted send failed:", error);
      useToastStore.getState().addToast("error", i18n.t("e2ee:encryptionFailed"));
      return false;
    }
  },

  edit: async (channelId, messageId, content) => {
    const e2ee = useE2EEStore.getState();
    const userId = useAuthStore.getState().user?.id;
    if (!canEncrypt(e2ee) || !userId || !e2ee.localDeviceId) {
      useToastStore.getState().addToast("error", "Encryption is still setting up. Try again shortly.");
      return false;
    }
    const held = (get().messagesByChannel[channelId] ?? []).find((message) => message.id === messageId);
    if (!held) return false;
    const payload = encryptedEditPayload(content, held);
    if (payload === null) {
      useToastStore.getState().addToast("error", i18n.t("e2ee:editAttachmentKeysMissing"));
      return false;
    }
    try {
      const senderKeyMessage = await encryptChannelMessage(channelId, userId, e2ee.localDeviceId, payload);
      const res = await voiceMessageApi.editVoiceMessage(
        channelId, messageId, JSON.stringify(senderKeyMessage), e2ee.localDeviceId, JSON.stringify({})
      );
      if (!res.success || !res.data) return false;
      await get().update({ ...res.data, content, e2ee_file_keys: held.e2ee_file_keys });
      return true;
    } catch (error) {
      console.error("[voiceMessageStore] encrypted edit failed:", error);
      useToastStore.getState().addToast("error", i18n.t("e2ee:encryptionFailed"));
      return false;
    }
  },

  del: async (channelId, messageId) => {
    const res = await voiceMessageApi.deleteVoiceMessage(channelId, messageId);
    if (!res.success) return false;

    get().remove(channelId, messageId);
    return true;
  },
}));
