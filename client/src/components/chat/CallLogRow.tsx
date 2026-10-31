/**
 * CallLogRow — WhatsApp-style call entry inside a DM conversation.
 * Renders a centered system row (not a message bubble) with a perspective-aware
 * label: the missed party sees "Missed call", the caller sees "Outgoing call".
 */

import { useTranslation } from "react-i18next";
import { useAuthStore } from "../../stores/authStore";
import { formatMessageTime } from "../../utils/dateFormat";
import type { CallMeta } from "../../types";

type CallLogRowProps = {
  meta: CallMeta;
  createdAt: string;
};

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

function CallLogRow({ meta, createdAt }: CallLogRowProps) {
  const { t, i18n } = useTranslation("chat");
  const myId = useAuthStore((s) => s.user?.id);

  const isVideo = meta.call_type === "video";
  const amCaller = myId === meta.caller_id;

  let label: string;
  let isMissed = false;

  if (meta.outcome === "completed") {
    label = `${isVideo ? t("callVideo") : t("callVoice")} · ${formatDuration(meta.duration_sec)}`;
  } else if (meta.outcome === "declined") {
    label = amCaller ? t("callDeclined") : t("callYouDeclined");
  } else {
    // missed (unanswered / cancelled)
    if (amCaller) {
      label = isVideo ? t("callOutgoingVideo") : t("callOutgoingVoice");
    } else {
      label = isVideo ? t("callMissedVideo") : t("callMissedVoice");
      isMissed = true;
    }
  }

  return (
    <div className={`call-log-row${isMissed ? " missed" : ""}`}>
      <span className="call-log-icon">
        {isVideo ? (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M23 7l-7 5 7 5V7z" />
            <rect x="1" y="5" width="15" height="14" rx="2" ry="2" />
          </svg>
        ) : (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.13.96.36 1.9.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.9.34 1.85.57 2.81.7A2 2 0 0 1 22 16.92z" />
          </svg>
        )}
      </span>
      <span className="call-log-text">{label}</span>
      <span className="call-log-time">{formatMessageTime(createdAt, i18n.language ?? "en", { yesterday: t("yesterday") })}</span>
    </div>
  );
}

export default CallLogRow;
