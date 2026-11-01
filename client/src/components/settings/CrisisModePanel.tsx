/** Platform admin controls for the isolated conversation-retention worker. */

import { useCallback, useEffect, useState } from "react";
import { useToastStore } from "../../stores/toastStore";
import { useConfirm } from "../../hooks/useConfirm";
import {
  cleanConversationsNow,
  getCrisisModeState,
  setCrisisModeEnabled,
  type CrisisModeCleanResult,
  type CrisisModeState,
} from "../../api/admin";

function CrisisModePanel() {
  const addToast = useToastStore((s) => s.addToast);
  const confirm = useConfirm();
  const [state, setState] = useState<CrisisModeState | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    const res = await getCrisisModeState();
    if (res.success && res.data) setState(res.data);
    else addToast("error", res.error ?? "Could not load Crisis Mode");
    setLoading(false);
  }, [addToast]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function handleToggle(enabled: boolean) {
    setBusy(true);
    try {
      const res = await setCrisisModeEnabled(enabled);
      if (res.success && res.data) {
        setState(res.data);
        addToast("success", enabled ? "Crisis Mode enabled" : "Crisis Mode disabled");
      } else addToast("error", res.error ?? "Could not update Crisis Mode");
    } catch {
      addToast("error", "Could not update Crisis Mode");
    } finally {
      setBusy(false);
    }
  }

  async function handleCleanNow() {
    const ok = await confirm({
      message: "Permanently delete all server-channel, DM, and voice messages and their attachments? Report records and report evidence will be retained. This cannot be undone.",
      confirmLabel: "Clean now",
      danger: true,
    });
    if (!ok) return;

    setBusy(true);
    try {
      const res = await cleanConversationsNow();
      if (res.success && res.data) {
        const counts: CrisisModeCleanResult = res.data;
        addToast(
          "success",
          `Conversation cleanup complete: ${counts.channel_messages} channel, ${counts.dm_messages} DM, ${counts.voice_messages} voice messages deleted.`
        );
        await refresh();
      } else addToast("error", res.error ?? "Conversation cleanup failed");
    } catch {
      addToast("error", "Conversation cleanup failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="settings-section">
      <div className="settings-section-header">
        <h2>Crisis Mode</h2>
        <p>Separate conversation-retention controls. Report records and report evidence are not affected.</p>
      </div>

      <div className="settings-card">
        {loading ? (
          <p>Loading Crisis Mode status…</p>
        ) : (
          <>
            <label className="settings-toggle-row">
              <span>
                <strong>Automatic 24-hour cleanup</strong>
                <small>When enabled, all channel, DM, and voice messages and their attachments are deleted every 24 hours.</small>
              </span>
              <input
                type="checkbox"
                checked={state?.enabled ?? false}
                disabled={busy || !state}
                onChange={(e) => void handleToggle(e.target.checked)}
              />
            </label>
            <p className="settings-hint">
              Last cleanup: {state?.last_run_at ? new Date(state.last_run_at).toLocaleString() : "Never"}
            </p>
          </>
        )}
        <div className="settings-actions">
          <button
            type="button"
            className="settings-btn settings-btn-danger"
            disabled={busy || loading}
            onClick={() => void handleCleanNow()}
          >
            {busy ? "Working…" : "Clean now"}
          </button>
        </div>
      </div>
    </section>
  );
}

export default CrisisModePanel;
