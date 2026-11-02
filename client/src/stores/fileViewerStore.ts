/**
 * fileViewerStore — global state for the in-app file viewer overlay.
 * Callers (plaintext + E2EE attachment renderers) push an item to open;
 * the overlay reads `item` and dispatches by mime type.
 *
 * URL lifecycle: the overlay does NOT own the `src` URL. The opener is
 * responsible for keeping any blob URL alive while the overlay is open
 * (typically by keeping the source component mounted) and revoking it
 * on its own unmount. Plaintext callers pass plain asset URLs which
 * need no revoke.
 */

import { create } from "zustand";

export type FileViewerItem = {
  /** Source URL: either an http(s) asset URL or a blob: URL from E2EE decrypt. */
  src: string;
  filename: string;
  /** MIME type used for viewer dispatch. */
  mime: string;
  /** Plaintext byte size, or null when unknown. */
  size: number | null;
  /** Optional override for download href; defaults to `src`. */
  downloadHref?: string;
};

type FileViewerState = {
  item: FileViewerItem | null;
  open: (item: FileViewerItem) => void;
  close: () => void;
};

export const useFileViewerStore = create<FileViewerState>((set) => ({
  item: null,
  open: (item) => set({ item }),
  close: () => set({ item: null }),
}));
