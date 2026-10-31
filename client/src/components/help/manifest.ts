/**
 * Help center manifest — the language-agnostic structure (single source of truth).
 *
 * Categories and article order live here; translations live elsewhere:
 *  - article/category TITLES → i18n "help" namespace (loaded with the app, used in the nav)
 *  - article BODIES → help/content/{lang}/{slug}.md (lazy-loaded on open, EN fallback)
 *
 * Adding a language never touches this file. Adding an article = add a slug here +
 * a title key in help.json + an EN markdown file (translations follow over time).
 */

export type HelpIcon =
  | "rocket" | "chat" | "voice" | "friends" | "servers"
  | "roles" | "lock" | "user" | "sliders" | "server" | "sparkle";

export type HelpArticleRef = {
  /** Matches help/content/{lang}/<slug>.md */
  slug: string;
  /** i18n key in the "help" namespace */
  titleKey: string;
};

export type HelpCategory = {
  id: string;
  /** i18n key in the "help" namespace */
  labelKey: string;
  icon: HelpIcon;
  articles: HelpArticleRef[];
};

export const HELP_CATEGORIES: HelpCategory[] = [
  {
    id: "getting-started",
    labelKey: "cat_gettingStarted",
    icon: "rocket",
    articles: [
      { slug: "welcome", titleKey: "art_welcome" },
      { slug: "register-login", titleKey: "art_register-login" },
      { slug: "download-app", titleKey: "art_download-app" },
      { slug: "join-invite", titleKey: "art_join-invite" },
      { slug: "interface-tour", titleKey: "art_interface-tour" },
      { slug: "first-voice-chat", titleKey: "art_first-voice-chat" },
    ],
  },
  {
    id: "messaging",
    labelKey: "cat_messaging",
    icon: "chat",
    articles: [
      { slug: "send-messages", titleKey: "art_send-messages" },
      { slug: "commands", titleKey: "art_commands" },
      { slug: "reactions-replies", titleKey: "art_reactions-replies" },
      { slug: "mentions", titleKey: "art_mentions" },
      { slug: "pins", titleKey: "art_pins" },
      { slug: "attachments", titleKey: "art_attachments" },
      { slug: "search-messages", titleKey: "art_search-messages" },
    ],
  },
  {
    id: "voice",
    labelKey: "cat_voice",
    icon: "voice",
    articles: [
      { slug: "audio-setup", titleKey: "art_audio-setup" },
      { slug: "voice-activity-ptt", titleKey: "art_voice-activity-ptt" },
      { slug: "in-call-controls", titleKey: "art_in-call-controls" },
      { slug: "screen-share", titleKey: "art_screen-share" },
      { slug: "soundboard", titleKey: "art_soundboard" },
    ],
  },
  {
    id: "friends",
    labelKey: "cat_friends",
    icon: "friends",
    articles: [
      { slug: "add-friends", titleKey: "art_add-friends" },
      { slug: "direct-messages", titleKey: "art_direct-messages" },
      { slug: "status-presence", titleKey: "art_status-presence" },
      { slug: "calls", titleKey: "art_calls" },
      { slug: "blocking-reporting", titleKey: "art_blocking-reporting" },
    ],
  },
  {
    id: "servers",
    labelKey: "cat_servers",
    icon: "servers",
    articles: [
      { slug: "create-server", titleKey: "art_create-server" },
      { slug: "channels-categories", titleKey: "art_channels-categories" },
      { slug: "invites", titleKey: "art_invites" },
    ],
  },
  {
    id: "roles",
    labelKey: "cat_roles",
    icon: "roles",
    articles: [
      { slug: "roles", titleKey: "art_roles" },
      { slug: "permissions", titleKey: "art_permissions" },
      { slug: "moderation", titleKey: "art_moderation" },
    ],
  },
  {
    id: "privacy",
    labelKey: "cat_privacy",
    icon: "lock",
    articles: [
      { slug: "encryption-overview", titleKey: "art_encryption-overview" },
      { slug: "devices-keys", titleKey: "art_devices-keys" },
      { slug: "recovery", titleKey: "art_recovery" },
    ],
  },
  {
    id: "account",
    labelKey: "cat_account",
    icon: "user",
    articles: [
      { slug: "profile", titleKey: "art_profile" },
      { slug: "security", titleKey: "art_security" },
      { slug: "manage-account", titleKey: "art_manage-account" },
    ],
  },
  {
    id: "app",
    labelKey: "cat_app",
    icon: "sliders",
    articles: [
      { slug: "themes-appearance", titleKey: "art_themes-appearance" },
      { slug: "notifications", titleKey: "art_notifications" },
      { slug: "desktop-app", titleKey: "art_desktop-app" },
    ],
  },
  {
    id: "selfhost",
    labelKey: "cat_selfhost",
    icon: "server",
    articles: [
      { slug: "self-host-overview", titleKey: "art_self-host-overview" },
      { slug: "voice-livekit", titleKey: "art_voice-livekit" },
      { slug: "full-server", titleKey: "art_full-server" },
      { slug: "connections", titleKey: "art_connections" },
    ],
  },
  {
    id: "nice-to-know",
    labelKey: "cat_niceToKnow",
    icon: "sparkle",
    articles: [{ slug: "split-view", titleKey: "art_split-view" }],
  },
];
