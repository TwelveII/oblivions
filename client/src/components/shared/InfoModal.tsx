/**
 * InfoModal — modal wrapper around HelpCenter.
 * view="tabs" (default) shows Features + Release Notes tabbed (desktop info icon).
 * view="features" / "releaseNotes" show a single section, no tabs (landing links).
 */

import { useTranslation } from "react-i18next";
import Modal from "./Modal";
import HelpCenter, { type HelpView } from "./HelpCenter";

type InfoModalProps = {
  isOpen: boolean;
  onClose: () => void;
  view?: HelpView;
};

function InfoModal({ isOpen, onClose, view = "tabs" }: InfoModalProps) {
  const { t } = useTranslation("common");
  const title =
    view === "features" ? t("features") : view === "releaseNotes" ? t("releaseNotes") : "Oblivions";

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title}>
      <HelpCenter view={view} />
    </Modal>
  );
}

export default InfoModal;
