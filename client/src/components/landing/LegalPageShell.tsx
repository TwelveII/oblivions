/**
 * LegalPageShell — Shared wrapper for public legal pages (privacy, terms).
 * Mirrors the redesigned landing structure: content must live inside
 * .lp-content (z-index above the fixed blurred ::before background).
 */

import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import "../../styles/landing.css";

type LegalPageShellProps = {
  /** i18n namespace providing footerCopy */
  ns: "privacy" | "terms";
  children: ReactNode;
};

function LegalPageShell({ ns, children }: LegalPageShellProps) {
  const { t } = useTranslation(ns);
  const navigate = useNavigate();

  return (
    <div className="landing-page">
      <div className="lp-content">
        <nav className="lp-nav">
          <img
            src="/oblivions-icon.svg"
            alt="Oblivions"
            className="lp-nav-logo-img lp-nav-home"
            onClick={() => navigate("/")}
          />
          <span className="lp-nav-brand lp-nav-home" onClick={() => navigate("/")}>
            Oblivions
          </span>

        </nav>

        <main className="privacy-content">{children}</main>

        <footer className="lp-footer">
          <div className="lp-footer-left">
            <img src="/oblivions-icon.svg" alt="Oblivions" className="lp-footer-logo-img" />
            <span className="lp-footer-copy">{t("footerCopy")}</span>
          </div>
          <div className="lp-footer-links">
            <a
              href="https://github.com/akinalpfdn/Mqvi"
              target="_blank"
              rel="noopener noreferrer"
              className="lp-footer-link"
            >
              GitHub
            </a>
          </div>
        </footer>
      </div>
    </div>
  );
}

export default LegalPageShell;
