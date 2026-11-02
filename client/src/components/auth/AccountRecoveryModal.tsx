/**
 * AccountRecoveryModal — shown when login is attempted on a soft-deleted account.
 * Offers to restore the account before the permanent-delete deadline.
 */

import { useTranslation } from "react-i18next";
import { useAuthStore } from "../../stores/authStore";

function formatDate(iso: string, locale: string): string {
  try {
    return new Date(iso).toLocaleString(locale, {
      dateStyle: "medium",
      timeStyle: "short",
    });
  } catch {
    return iso;
  }
}

function AccountRecoveryModal() {
  const { t, i18n } = useTranslation("auth");
  const accountDeleted = useAuthStore((s) => s.accountDeleted);
  const isLoading = useAuthStore((s) => s.isLoading);
  const restoreAccount = useAuthStore((s) => s.restoreAccount);
  const cancelAccountDeleted = useAuthStore((s) => s.cancelAccountDeleted);

  if (!accountDeleted) return null;

  return (
    <div className="modal-backdrop" onClick={cancelAccountDeleted}>
      <div
        className="modal-card"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">{t("accountDeletedTitle")}</h2>
        </div>
        <p className="modal-text">
          {t("accountDeletedBody", {
            permanentDeleteAt: formatDate(accountDeleted.permanentDeleteAt, i18n.language),
          })}
        </p>
        <div className="modal-actions">
          <button
            type="button"
            className="settings-btn settings-btn-secondary"
            onClick={cancelAccountDeleted}
            disabled={isLoading}
          >
            {t("accountDeletedCancel")}
          </button>
          <button
            type="button"
            className="settings-btn"
            onClick={() => void restoreAccount()}
            disabled={isLoading}
          >
            {t("accountDeletedRestoreButton")}
          </button>
        </div>
      </div>
    </div>
  );
}

export default AccountRecoveryModal;
