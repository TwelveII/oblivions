/**
 * VoiceChannelDuration — shows live call duration ("0:14", "1:23:45") next to
 * a voice channel in the sidebar. Self-ticking 1s interval so re-renders are
 * scoped to this leaf and don't repaint the whole channel tree.
 *
 * Elapsed time is computed from Date.now() at render time (not from stored
 * state) so any re-render between ticks always shows the true elapsed —
 * a stale state value can't cause the display to jump or lag.
 */

import { useEffect, useState } from "react";
import { useVoiceStore } from "../../stores/voiceStore";

type Props = { channelId: string };

function formatDuration(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const ss = s.toString().padStart(2, "0");
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, "0")}:${ss}`;
  }
  return `${m}:${ss}`;
}

function VoiceChannelDuration({ channelId }: Props) {
  const startedAt = useVoiceStore((s) => s.channelTimers[channelId]);
  // Tick counter — value is unused, it just triggers re-renders every second.
  const [, setTick] = useState(0);

  useEffect(() => {
    if (!startedAt) return;
    // Force an immediate re-render so the first frame after startedAt arrives
    // is computed against current Date.now(), not whatever value happened to
    // be in scope from the previous render.
    setTick((t) => t + 1);
    const id = setInterval(() => setTick((t) => t + 1), 1000);
    return () => clearInterval(id);
  }, [startedAt]);

  if (!startedAt) return null;

  return <span className="ch-tree-voice-duration">{formatDuration(Date.now() - startedAt)}</span>;
}

export default VoiceChannelDuration;
