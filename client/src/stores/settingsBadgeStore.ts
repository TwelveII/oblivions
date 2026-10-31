/**
 * settingsBadgeStore — dot indicator state for Settings menu items.
 * Admin: new feedback / new reports panels. User: own feedback panel when
 * an admin has replied to one of their tickets. Cleared on panel mount.
 */

import { create } from "zustand";
import { getAdminBadges, markFeedbackSeen, markReportsSeen } from "../api/admin";
import { getMyFeedbackBadge, markMyFeedbackSeen } from "../api/feedback";

type SettingsBadgeState = {
  hasNewFeedback: boolean;
  hasNewReports: boolean;
  hasNewMyFeedbackReply: boolean;
  refreshAdmin: () => Promise<void>;
  refreshMyFeedback: () => Promise<void>;
  clearFeedback: () => Promise<void>;
  clearReports: () => Promise<void>;
  clearMyFeedback: () => Promise<void>;
};

export const useSettingsBadgeStore = create<SettingsBadgeState>((set) => ({
  hasNewFeedback: false,
  hasNewReports: false,
  hasNewMyFeedbackReply: false,

  refreshAdmin: async () => {
    const res = await getAdminBadges();
    if (res.success && res.data) {
      set({
        hasNewFeedback: res.data.has_new_feedback,
        hasNewReports: res.data.has_new_reports,
      });
    }
  },

  refreshMyFeedback: async () => {
    const res = await getMyFeedbackBadge();
    if (res.success && res.data) {
      set({ hasNewMyFeedbackReply: res.data.has_new_replies });
    }
  },

  clearFeedback: async () => {
    set({ hasNewFeedback: false });
    const res = await markFeedbackSeen();
    if (!res.success) set({ hasNewFeedback: true });
  },

  clearReports: async () => {
    set({ hasNewReports: false });
    const res = await markReportsSeen();
    if (!res.success) set({ hasNewReports: true });
  },

  clearMyFeedback: async () => {
    set({ hasNewMyFeedbackReply: false });
    const res = await markMyFeedbackSeen();
    if (!res.success) set({ hasNewMyFeedbackReply: true });
  },
}));
