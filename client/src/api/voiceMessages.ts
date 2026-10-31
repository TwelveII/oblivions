/**
 * Voice channel ephemeral chat API.
 *
 * - GET    /api/voice-channels/{channelId}/messages
 * - POST   /api/voice-channels/{channelId}/messages   (JSON or multipart for files)
 * - PATCH  /api/voice-channels/{channelId}/messages/{messageId}
 * - DELETE /api/voice-channels/{channelId}/messages/{messageId}
 *
 * Server-side membership check: caller must currently be in the target voice channel.
 */

import { apiClient, uploadRequest, type UploadOptions } from "./client";
import type { VoiceMessage } from "../types";

export async function listVoiceMessages(channelId: string) {
  return apiClient<VoiceMessage[]>(`/voice-channels/${channelId}/messages`);
}

export async function sendVoiceMessage(
  channelId: string,
  ciphertext: string,
  senderDeviceId: string,
  metadata: string,
  files?: File[],
  upload?: UploadOptions
) {
  if (files && files.length > 0) {
    const formData = new FormData();
    formData.append("encryption_version", "1");
    formData.append("ciphertext", ciphertext);
    formData.append("sender_device_id", senderDeviceId);
    formData.append("e2ee_metadata", metadata);
    for (const file of files) formData.append("files", file);
    return uploadRequest<VoiceMessage>(`/voice-channels/${channelId}/messages`, formData, upload);
  }
  return apiClient<VoiceMessage>(`/voice-channels/${channelId}/messages`, {
    method: "POST",
    body: { encryption_version: 1, ciphertext, sender_device_id: senderDeviceId, e2ee_metadata: metadata },
    signal: upload?.signal,
  });
}
export async function editVoiceMessage(
  channelId: string,
  messageId: string,
  ciphertext: string,
  senderDeviceId: string,
  metadata: string
) {
  return apiClient<VoiceMessage>(`/voice-channels/${channelId}/messages/${messageId}`, {
    method: "PATCH",
    body: { encryption_version: 1, ciphertext, sender_device_id: senderDeviceId, e2ee_metadata: metadata },
  });
}
export async function deleteVoiceMessage(channelId: string, messageId: string) {
  return apiClient<void>(`/voice-channels/${channelId}/messages/${messageId}`, {
    method: "DELETE",
  });
}
