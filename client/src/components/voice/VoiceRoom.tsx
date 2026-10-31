/**
 * VoiceRoom — Visual component for the voice channel.
 *
 * Layout:
 * - Default: voice content fills the area (participant grid + screen share).
 * - Chat toggle (top-right) opens an ephemeral chat panel side-by-side on the right.
 *
 * Chat reuses Message/MessageInput/MessageList via VoiceChatProvider — no
 * duplicated UI. Voice-specific features (reactions/reply/pin/mentions) are
 * gated off in shared components via mode === "voice".
 *
 * LiveKit connection is managed by VoiceProvider at AppLayout level.
 */

import { useState } from "react";
import { useVoiceStore } from "../../stores/voiceStore";
import { useChannelStore } from "../../stores/channelStore";
import { useTranslation } from "react-i18next";
import VoiceParticipantGrid from "./VoiceParticipantGrid";
import VoiceConnectionStatus from "./VoiceConnectionStatus";
import ScreenShareView from "./ScreenShareView";
import VoiceChatProvider from "./VoiceChatProvider";
import MessageList from "../chat/MessageList";
import MessageInput from "../chat/MessageInput";

function VoiceRoom() {
  const { t } = useTranslation("voice");
  const livekitUrl = useVoiceStore((s) => s.livekitUrl);
  const livekitToken = useVoiceStore((s) => s.livekitToken);
  const currentVoiceChannelId = useVoiceStore((s) => s.currentVoiceChannelId);
  const channelName = useChannelStore((s) => {
    if (!currentVoiceChannelId) return "";
    for (const cg of s.categories) {
      const ch = cg.channels.find((c) => c.id === currentVoiceChannelId);
      if (ch) return ch.name;
    }
    return "";
  });
  const [chatOpen, setChatOpen] = useState(false);

  if (!livekitUrl || !livekitToken) {
    return (
      <div className="voice-room-loading">
        <p>{t("connectingToVoice")}</p>
      </div>
    );
  }

  // Apply split only when we'll actually render the chat panel — otherwise the
  // split CSS leaves a 50% empty area next to the voice grid.
  const chatVisible = chatOpen && !!currentVoiceChannelId;

  return (
    <div className={`voice-room${chatVisible ? " split" : ""}`}>
      <div className="voice-room-content">
        <button
          className={`vc-chat-toggle${chatVisible ? " open" : ""}`}
          onClick={() => setChatOpen((v) => !v)}
          title={chatVisible ? t("chatHide") : t("chatShow")}
          aria-label={chatVisible ? t("chatHide") : t("chatShow")}
        >
          <svg viewBox="0 0 24 24" width="40" height="40" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
          </svg>
        </button>
        <VoiceConnectionStatus />
        <ScreenShareView />
        <VoiceParticipantGrid />
      </div>
      {chatVisible && currentVoiceChannelId && (
        <div className="voice-room-chat">
          <VoiceChatProvider channelId={currentVoiceChannelId} channelName={channelName}>
            <MessageList />
            <MessageInput openSearch={() => {}} />
          </VoiceChatProvider>
        </div>
      )}
    </div>
  );
}

export default VoiceRoom;
