/**
 * FriendsVoiceList — right-panel body shown when the Friends view is open.
 * Lists friends currently in a voice channel (across all servers the user
 * shares with them), with the server + channel they're in. Display only.
 */

import { useTranslation } from "react-i18next";
import { useFriendStore } from "../../stores/friendStore";
import { useVoiceStore } from "../../stores/voiceStore";
import { useServerStore } from "../../stores/serverStore";
import Avatar from "../shared/Avatar";
import type { VoiceState } from "../../types";

function FriendsVoiceList() {
  const { t } = useTranslation("common");
  const friends = useFriendStore((s) => s.friends);
  const voiceStates = useVoiceStore((s) => s.voiceStates);
  const servers = useServerStore((s) => s.servers);

  // Flatten voice presence into a user_id → location lookup.
  const voiceByUser = new Map<string, VoiceState>();
  for (const list of Object.values(voiceStates)) {
    for (const st of list) voiceByUser.set(st.user_id, st);
  }

  const inVoice = friends
    .map((f) => ({ friend: f, voice: voiceByUser.get(f.user_id) }))
    .filter((x): x is { friend: typeof x.friend; voice: VoiceState } => !!x.voice);

  if (inVoice.length === 0) {
    return <div className="fva-empty">{t("noFriendsInVoice")}</div>;
  }

  return (
    <div className="fva-list">
      {inVoice.map(({ friend, voice }) => {
        const serverName = servers.find((s) => s.id === voice.server_id)?.name ?? "";
        const name = friend.display_name ?? friend.username;
        return (
          <div key={friend.user_id} className="fva-row">
            <div className="fva-avatar">
              <Avatar name={name} avatarUrl={friend.avatar_url ?? undefined} size={36} isCircle />
            </div>
            <div className="fva-info">
              <span className="fva-name">{name}</span>
              <span className="fva-location">
                <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M15.536 8.464a5 5 0 010 7.072M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z"
                  />
                </svg>
                <span className="fva-location-text">
                  {voice.channel_name}
                  {serverName && <span className="fva-server"> · {serverName}</span>}
                </span>
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default FriendsVoiceList;
