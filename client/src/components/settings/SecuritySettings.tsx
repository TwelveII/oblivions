/** Security settings: signup invites, password, sessions, deleted servers, and account deletion. */

import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useToastStore } from "../../stores/toastStore";
import { useAuthStore } from "../../stores/authStore";
import * as authApi from "../../api/auth";
import * as serversApi from "../../api/servers";
import { PASSWORD_MIN_LENGTH, PASSWORD_MAX_BYTES, passwordByteLength } from "../../utils/constants";
import { passwordErrorKey } from "../../utils/passwordError";
import ActiveSessions from "./ActiveSessions";
import type { DeletedServerInfo } from "../../api/servers";

function formatDate(iso: string, locale: string): string {
  try {
    return new Date(iso).toLocaleString(locale, { dateStyle: "medium", timeStyle: "short" });
  } catch {
    return iso;
  }
}

function daysUntil(iso: string): number {
  try {
    const ms = new Date(iso).getTime() - Date.now();
    return Math.max(0, Math.ceil(ms / (1000 * 60 * 60 * 24)));
  } catch {
    return 0;
  }
}

function SecuritySettings() {
  const { t, i18n } = useTranslation("settings");
  const addToast = useToastStore((s) => s.addToast);
  const user = useAuthStore((s) => s.user);
  const replaceTokens = useAuthStore((s) => s.replaceTokens);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  // ─── Deleted Servers State ───
  const [deletedServers, setDeletedServers] = useState<DeletedServerInfo[]>([]);
  const [isLoadingDeleted, setIsLoadingDeleted] = useState(true);
  const [actioningServer, setActioningServer] = useState<string | null>(null);

  // ─── Delete Account State ───
  const [deletePassword, setDeletePassword] = useState("");
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [isDeletingAccount, setIsDeletingAccount] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const res = await serversApi.getDeletedServers();
      if (cancelled) return;
      if (res.success && res.data) {
        setDeletedServers(res.data);
      }
      setIsLoadingDeleted(false);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // ─── Signup Invite State ───
  const [signupInvites, setSignupInvites] = useState<authApi.SignupInvite[]>([]);
  const [isLoadingInvites, setIsLoadingInvites] = useState(true);
  const [inviteExpiry, setInviteExpiry] = useState(0);
  const [isCreatingInvite, setIsCreatingInvite] = useState(false);
  const [newlyCreatedCode, setNewlyCreatedCode] = useState("");

  useEffect(() => {
    let cancelled = false;
    void authApi.getSignupInvites().then((res) => {
      if (cancelled) return;
      if (res.success && res.data) setSignupInvites(res.data);
      setIsLoadingInvites(false);
    });
    return () => { cancelled = true; };
  }, []);

  // Refresh invite codes on focus and never render consumed codes.
  useEffect(() => {
    const refresh = () => {
      void authApi.getSignupInvites().then((res) => {
        if (res.success && res.data) setSignupInvites(res.data);
      });
    };
    window.addEventListener("focus", refresh);
    return () => window.removeEventListener("focus", refresh);
  }, []);

  // ─── Password State ───
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  // ─── Password Handlers ───
  const canSubmitPassword =
    currentPassword.length > 0 &&
    newPassword.length > 0 &&
    confirmPassword.length > 0 &&
    !isSaving;

  async function handlePasswordSubmit() {
    if (!canSubmitPassword) return;

    if (newPassword.length < PASSWORD_MIN_LENGTH) {
      addToast("error", t("passwordTooShort"));
      return;
    }

    if (passwordByteLength(newPassword) > PASSWORD_MAX_BYTES) {
      addToast("error", t("passwordTooLong"));
      return;
    }

    if (newPassword !== confirmPassword) {
      addToast("error", t("passwordMismatch"));
      return;
    }

    if (currentPassword === newPassword) {
      addToast("error", t("passwordSameAsOld"));
      return;
    }

    setIsSaving(true);
    try {
      const res = await authApi.changePassword(currentPassword, newPassword);
      if (res.success && res.data) {
        replaceTokens(res.data.access_token, res.data.refresh_token, res.data.file_token);
        addToast("success", t("passwordChanged"));
        setCurrentPassword("");
        setNewPassword("");
        setConfirmPassword("");
      } else {
        // Password rejections carry a code; the rest are still matched on English text.
        const passwordKey = passwordErrorKey(res.code);
        const errMsg = res.error ?? "";
        if (passwordKey) {
          addToast("error", i18n.t(passwordKey));
        } else if (errMsg.includes("incorrect") || errMsg.includes("unauthorized")) {
          addToast("error", t("wrongCurrentPassword"));
        } else if (errMsg.includes("different")) {
          addToast("error", t("passwordSameAsOld"));
        } else {
          addToast("error", t("passwordChangeError"));
        }
      }
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="settings-section">
      <h2 className="settings-section-title">{t("security")}</h2>

      {/* ═══ Signup Invite Codes ═══ */}
      <h3 className="settings-section-subtitle">Signup invite codes</h3>
      <p className="settings-hint">
        Create single-use codes to invite people to register.
        Codes count toward your lifetime creation limit, even after expiry or revocation.
      </p>
      <div className="settings-field">
        <p className="settings-value">
          Codes created: {user?.invite_codes_created ?? 0} / {user?.invite_limit ?? 1}
        </p>
      </div>
      <div className="settings-field">
        <label htmlFor="signupInviteExpiry" className="settings-label">Code expiration</label>
        <select
          id="signupInviteExpiry"
          className="settings-input"
          value={inviteExpiry}
          onChange={(e) => setInviteExpiry(Number(e.target.value))}
        >
          <option value={0}>Never</option>
          <option value={1}>1 day</option>
          <option value={7}>7 days</option>
          <option value={30}>30 days</option>
        </select>
      </div>
      <div className="settings-btn-row">
        <button
          className="settings-btn"
          disabled={isCreatingInvite || (user?.invite_codes_created ?? 0) >= (user?.invite_limit ?? 1)}
          onClick={async () => {
            setIsCreatingInvite(true);
            const res = await authApi.createSignupInvite(inviteExpiry);
            setIsCreatingInvite(false);
            if (res.success && res.data) {
              setNewlyCreatedCode(res.data.code);
              setSignupInvites((prev) => [res.data!, ...prev]);
              useAuthStore.setState((state) => ({
                user: state.user ? {
                  ...state.user,
                  invite_codes_created: (state.user.invite_codes_created ?? 0) + 1,
                } : null,
              }));
              addToast("success", "Signup invite created");
            } else {
              addToast("error", res.error ?? "Could not create signup invite");
            }
          }}
        >
          {isCreatingInvite ? "Creating..." : "Create invite code"}
        </button>
      </div>

      {newlyCreatedCode && (
        <div className="settings-field">
          <label className="settings-label">Your new code — copy it now</label>
          <div className="settings-btn-row">
            <code className="settings-value" style={{ overflowWrap: "anywhere" }}>{newlyCreatedCode}</code>
            <button className="settings-btn" onClick={async () => {
              await navigator.clipboard.writeText(newlyCreatedCode);
              addToast("success", "Invite code copied");
            }}>Copy</button>
          </div>
        </div>
      )}

      {isLoadingInvites ? (
        <p className="settings-value">Loading invite codes…</p>
      ) : signupInvites.filter((invite) => !invite.used_at).length === 0 ? (
        <p className="settings-value">You haven't created any signup codes.</p>
      ) : (
        <ul className="signup-invite-list">
          {signupInvites.filter((invite) => !invite.used_at).map((invite) => {
            const status = invite.revoked_at ? "Revoked"
              : invite.used_at ? "Used"
              : invite.expires_at && new Date(invite.expires_at).getTime() <= Date.now() ? "Expired"
              : "Available";
            return (
              <li key={invite.id} className="settings-field signup-invite-item" style={{
                display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12,
              }}>
                <div>
                  <div className="settings-value">{status}</div>
                  <div className="settings-hint">
                    Created {formatDate(invite.created_at, i18n.language)}
                    {invite.expires_at ? ` · Expires ${formatDate(invite.expires_at, i18n.language)}` : " · Never expires"}
                  </div>
                </div>
                <button
                  className="settings-btn settings-btn-danger"
                  disabled={status !== "Available"}
                  onClick={async () => {
                    const res = await authApi.revokeSignupInvite(invite.id);
                    if (res.success) {
                      setSignupInvites((prev) => prev.map((item) =>
                        item.id === invite.id ? { ...item, revoked_at: new Date().toISOString() } : item
                      ));
                      addToast("success", "Invite code revoked");
                    } else {
                      addToast("error", res.error ?? "Could not revoke invite code");
                    }
                  }}
                >Revoke</button>
              </li>
            );
          })}
        </ul>
      )}

      <div className="settings-divider" />

      {/* ═══ Password Section ═══ */}
      <h3 className="settings-section-subtitle">{t("changePassword")}</h3>

      {/* Current Password */}
      <div className="settings-field">
        <label htmlFor="currentPassword" className="settings-label">
          {t("currentPassword")}
        </label>
        <input
          id="currentPassword"
          type="password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          placeholder={t("currentPasswordPlaceholder")}
          className="settings-input"
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
        />
      </div>

      {/* New Password */}
      <div className="settings-field">
        <label htmlFor="newPassword" className="settings-label">
          {t("newPassword")}
        </label>
        <input
          id="newPassword"
          type="password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          placeholder={t("newPasswordPlaceholder")}
          className="settings-input"
          autoComplete="new-password"
        />
      </div>

      {/* Confirm New Password */}
      <div className="settings-field">
        <label htmlFor="confirmNewPassword" className="settings-label">
          {t("confirmNewPassword")}
        </label>
        <input
          id="confirmNewPassword"
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          placeholder={t("confirmNewPasswordPlaceholder")}
          className="settings-input"
          autoComplete="new-password"
        />
      </div>

      {/* Submit */}
      <div style={{ marginTop: 16 }}>
        <button
          onClick={handlePasswordSubmit}
          disabled={!canSubmitPassword}
          className="settings-btn"
        >
          {isSaving ? t("changePassword") + "..." : t("changePassword")}
        </button>
      </div>

      {/* ═══ Separator ═══ */}
      <div className="settings-divider" />

      {/* ═══ Active Devices Section ═══ */}
      <ActiveSessions />

      <div className="settings-divider" />

      {/* ═══ Deleted Servers Section ═══ */}
      <h3 className="settings-section-subtitle">{t("deletedServers")}</h3>
      <p className="settings-hint">{t("deletedServersDescription")}</p>

      {isLoadingDeleted ? (
        <p className="settings-value">…</p>
      ) : deletedServers.length === 0 ? (
        <p className="settings-value">{t("deletedServersEmpty")}</p>
      ) : (
        <ul className="deleted-server-list" style={{ listStyle: "none", padding: 0, margin: 0 }}>
          {deletedServers.map((srv) => {
            const remaining = daysUntil(srv.permanent_delete_at);
            const isAdminDeleted = srv.deleted_by_admin;
            const busy = actioningServer === srv.id;
            return (
              <li
                key={srv.id}
                className="settings-field"
                style={{ display: "flex", alignItems: "center", gap: 12 }}
              >
                {srv.icon_url ? (
                  <img
                    src={srv.icon_url}
                    alt=""
                    style={{ width: 40, height: 40, borderRadius: 8, objectFit: "cover" }}
                  />
                ) : (
                  <div
                    style={{
                      width: 40,
                      height: 40,
                      borderRadius: 8,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      background: "var(--color-background-tertiary)",
                    }}
                    aria-hidden
                  >
                    {srv.name.charAt(0).toUpperCase()}
                  </div>
                )}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontWeight: 600 }}>{srv.name}</div>
                  <div className="settings-hint" style={{ marginTop: 2 }}>
                    {isAdminDeleted ? (
                      <span>{t("deletedByAdminLabel")} · {formatDate(srv.deleted_at, i18n.language)}</span>
                    ) : (
                      <span>
                        {t("permanentDeleteIn", { days: remaining })} · {formatDate(srv.deleted_at, i18n.language)}
                      </span>
                    )}
                  </div>
                </div>
                <div style={{ display: "flex", gap: 8 }}>
                  {!isAdminDeleted && (
                    <button
                      className="settings-btn"
                      disabled={busy}
                      onClick={async () => {
                        setActioningServer(srv.id);
                        const res = await serversApi.restoreServer(srv.id);
                        setActioningServer(null);
                        if (res.success) {
                          setDeletedServers((prev) => prev.filter((s) => s.id !== srv.id));
                          addToast("success", t("restoreServerSuccess"));
                        } else {
                          addToast("error", res.error ?? t("restoreServerFailed"));
                        }
                      }}
                    >
                      {t("restore")}
                    </button>
                  )}
                  {isAdminDeleted && (
                    <span
                      className="settings-hint"
                      style={{ alignSelf: "center", maxWidth: 240, textAlign: "right" }}
                    >
                      {t("deletedByAdminCannotRestore")}
                    </span>
                  )}
                  <button
                    className="settings-btn settings-btn-danger"
                    disabled={busy || isAdminDeleted}
                    onClick={async () => {
                      if (!window.confirm(t("permanentDeleteServerConfirm", { name: srv.name }))) return;
                      setActioningServer(srv.id);
                      const res = await serversApi.hardDeleteServer(srv.id);
                      setActioningServer(null);
                      if (res.success) {
                        setDeletedServers((prev) => prev.filter((s) => s.id !== srv.id));
                        addToast("success", t("permanentDeleteServerSuccess"));
                      } else {
                        addToast("error", res.error ?? t("permanentDeleteServerFailed"));
                      }
                    }}
                  >
                    {t("permanentDelete")}
                  </button>
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {/* ═══ Separator ═══ */}
      <div className="settings-divider" />

      {/* ═══ Delete Account Section ═══ */}
      <h3 className="settings-section-subtitle">{t("deleteAccountSection")}</h3>
      <p className="settings-hint">{t("deleteAccountDescription")}</p>

      {!confirmingDelete ? (
        <div style={{ marginTop: 16 }}>
          <button
            className="settings-btn settings-btn-danger"
            onClick={() => setConfirmingDelete(true)}
          >
            {t("deleteAccount")}
          </button>
        </div>
      ) : (
        <div style={{ marginTop: 16 }}>
          <p className="settings-value" style={{ marginBottom: 12 }}>
            {t("deleteAccountConfirmBody")}
          </p>
          <div className="settings-field">
            <label htmlFor="deleteAccountPassword" className="settings-label">
              {t("deleteAccountPasswordLabel")}
            </label>
            <input
              id="deleteAccountPassword"
              type="password"
              value={deletePassword}
              onChange={(e) => setDeletePassword(e.target.value)}
              className="settings-input"
              autoComplete="off"
              data-1p-ignore
              data-lpignore="true"
            />
          </div>
          <div className="settings-btn-row">
            <button
              className="settings-btn"
              onClick={() => {
                setConfirmingDelete(false);
                setDeletePassword("");
              }}
              disabled={isDeletingAccount}
            >
              {t("cancel", { ns: "common", defaultValue: "Cancel" })}
            </button>
            <button
              className="settings-btn settings-btn-danger"
              disabled={isDeletingAccount || deletePassword.length === 0}
              onClick={async () => {
                setIsDeletingAccount(true);
                const res = await authApi.softDeleteSelf(deletePassword);
                setIsDeletingAccount(false);
                if (res.success) {
                  addToast("success", t("deleteAccountSuccess"));
                  await logout();
                  navigate("/login");
                } else {
                  addToast("error", res.error ?? t("deleteAccountFailed"));
                }
              }}
            >
              {t("deleteAccountConfirm")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default SecuritySettings;
