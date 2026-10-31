export const APP_STORE_ID = "";
export const APP_STORE_URL = "";
export const PLAY_STORE_URL = "";

import { isNativeApp } from "./constants";

export type OSInfo = {
  os: "windows" | "macos" | "linux" | "ios" | "android" | "mobile";
  url: string;
  i18nKey: string;
};

export function detectOS(): OSInfo {
  const ua = navigator.userAgent.toLowerCase();

  if (/iphone|ipad|ipod/.test(ua) || (ua.includes("mac") && navigator.maxTouchPoints > 1)) {
    return { os: "ios", url: "", i18nKey: "hero_download_ios" };
  }
  if (ua.includes("android")) {
    return { os: "android", url: "", i18nKey: "hero_download_android" };
  }
  if (/mobile/.test(ua)) {
    return { os: "mobile", url: "", i18nKey: "hero_download_desktop" };
  }
  if (ua.includes("mac")) {
    return { os: "macos", url: "", i18nKey: "hero_download_macos" };
  }
  if (ua.includes("linux")) {
    return { os: "linux", url: "", i18nKey: "hero_download_linux" };
  }
  return { os: "windows", url: "", i18nKey: "hero_download_windows" };
}

export function shouldShowDownloadPrompt(): boolean {
  if (isNativeApp()) return false;
  return false;
}

export function isDownloadPromptPending(
  user: { has_seen_download_prompt: boolean } | null,
): boolean {
  return !!user && !user.has_seen_download_prompt && shouldShowDownloadPrompt();
}
