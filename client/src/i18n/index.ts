/**
 * i18n configuration — i18next + react-i18next setup.
 *
 * Namespaces: common, auth, channels, chat, settings, voice, landing, privacy, servers, dm, e2ee, guide.
 * Supported languages: EN only.
 */

import i18n from "i18next";
import { initReactI18next } from "react-i18next";

// ─── EN Translation Files ───
import enCommon from "./locales/en/common.json";
import enAuth from "./locales/en/auth.json";
import enChannels from "./locales/en/channels.json";
import enChat from "./locales/en/chat.json";
import enSettings from "./locales/en/settings.json";
import enVoice from "./locales/en/voice.json";
import enLanding from "./locales/en/landing.json";
import enServers from "./locales/en/servers.json";
import enDM from "./locales/en/dm.json";
import enE2EE from "./locales/en/e2ee.json";
import enSoundboard from "./locales/en/soundboard.json";
import enPrivacy from "./locales/en/privacy.json";
import enTerms from "./locales/en/terms.json";
import enViewer from "./locales/en/viewer.json";
import enHelp from "./locales/en/help.json";
import enDiscovery from "./locales/en/discovery.json";
import enGuide from "./locales/en/guide.json";

export const SUPPORTED_LANGUAGES = { en: "English" } as const;

export type Language = keyof typeof SUPPORTED_LANGUAGES;

export const DEFAULT_LANGUAGE: Language = "en";

/**
 * The application ships English only; all browser and account locales resolve to English.
 */
export function resolveLanguage(_raw?: string | null): Language {
  return "en";
}

i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        common: enCommon,
        auth: enAuth,
        channels: enChannels,
        chat: enChat,
        settings: enSettings,
        voice: enVoice,
        landing: enLanding,
        servers: enServers,
        dm: enDM,
        e2ee: enE2EE,
        soundboard: enSoundboard,
        privacy: enPrivacy,
        terms: enTerms,
        viewer: enViewer,
        help: enHelp,
        discovery: enDiscovery,
        guide: enGuide,
      },
    },

    fallbackLng: DEFAULT_LANGUAGE,
    supportedLngs: ["en"],
    load: "languageOnly",

    defaultNS: "common",
    ns: ["common", "auth", "channels", "chat", "settings", "voice", "landing", "privacy", "terms", "servers", "dm", "e2ee", "soundboard", "viewer", "help", "discovery", "guide"],

    interpolation: {
      // React already handles XSS protection
      escapeValue: false,
    },

  });

/** Changes language and persists to localStorage. */
export function changeLanguage(lng: Language): void {
  i18n.changeLanguage(lng);
  localStorage.setItem("language", lng);
}

export default i18n;
