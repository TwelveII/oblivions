/**
 * ConnectionsModal — wraps the backend-connection manager (ConnectionsSettings) in a
 * modal so a self-hoster can switch servers both from settings AND from the pre-auth
 * login/register pages. The pre-auth entry matters: connection is picked before login,
 * and without it a down/unreachable default server would lock native apps out entirely.
 * Native apps (Electron + mobile) only.
 */

import { useTranslation } from "react-i18next";
import Modal from "../shared/Modal";
import ConnectionsSettings from "./ConnectionsSettings";

type ConnectionsModalProps = {
  isOpen: boolean;
  onClose: () => void;
};

function ConnectionsModal({ isOpen, onClose }: ConnectionsModalProps) {
  const { t } = useTranslation("settings");
  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t("connectionsTitle")}>
      <ConnectionsSettings />
    </Modal>
  );
}

export default ConnectionsModal;
