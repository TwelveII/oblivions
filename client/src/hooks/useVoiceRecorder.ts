/**
 * useVoiceRecorder — records a mic clip via MediaRecorder and returns it as a File.
 *
 * Format is picked per-platform: opus/webm on Chromium/Android, mp4/aac on iOS —
 * all accepted by the server MIME whitelist. The produced File is staged in the
 * composer (not auto-sent); the user presses Enter / send to actually send it.
 */

import { useRef, useState, useCallback, useEffect } from "react";
import { ensureMicPermission } from "../utils/devicePermissions";

type RecorderState = "idle" | "recording";

function pickMimeType(): string {
  const candidates = [
    "audio/webm;codecs=opus",
    "audio/webm",
    "audio/mp4",
    "audio/aac",
    "audio/ogg",
  ];
  if (typeof MediaRecorder === "undefined") return "";
  for (const c of candidates) {
    if (MediaRecorder.isTypeSupported(c)) return c;
  }
  return "";
}

function extForMime(type: string): string {
  if (type.includes("mp4") || type.includes("aac") || type.includes("m4a")) return "m4a";
  if (type.includes("ogg")) return "ogg";
  return "webm";
}

export function useVoiceRecorder() {
  const [state, setState] = useState<RecorderState>("idle");
  const [elapsedMs, setElapsedMs] = useState(0);

  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const startedAtRef = useRef(0);
  const timerRef = useRef<number | null>(null);
  const resolveRef = useRef<((f: File | null) => void) | null>(null);
  const cancelledRef = useRef(false);
  // Abort token — bumped by cancel/stop/unmount. If it changes during start()'s
  // async permission/getUserMedia gap, that start was aborted and must not go live.
  const startSeqRef = useRef(0);
  // Single-flight guard: re-entrant stop() calls (auto-finish + double "Done")
  // return the same in-flight promise so chunks aren't cleared mid-finalize.
  const stopPromiseRef = useRef<Promise<File | null> | null>(null);

  const cleanup = useCallback(() => {
    if (timerRef.current !== null) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    recorderRef.current = null;
    chunksRef.current = [];
    stopPromiseRef.current = null;
    setState("idle");
    setElapsedMs(0);
  }, []);

  // Stop the mic + timer if the component unmounts mid-recording (e.g. the user
  // switches channel/DM while a locked recording is running). Without this the
  // mic stream stays live — a privacy leak. No setState here (already unmounting).
  useEffect(() => {
    return () => {
      startSeqRef.current++; // abort any in-flight start
      if (timerRef.current !== null) clearInterval(timerRef.current);
      const rec = recorderRef.current;
      if (rec && rec.state !== "inactive") {
        rec.onstop = null;
        try {
          rec.stop();
        } catch {
          /* ignore */
        }
      }
      streamRef.current?.getTracks().forEach((t) => t.stop());
    };
  }, []);

  const start = useCallback(async (): Promise<boolean> => {
    if (recorderRef.current) return false;

    const seq = ++startSeqRef.current;

    const granted = await ensureMicPermission();
    if (!granted || seq !== startSeqRef.current) return false;

    let stream: MediaStream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      return false;
    }
    // Aborted (cancel/stop/unmount) during the async gap — free the mic, don't go live.
    if (seq !== startSeqRef.current) {
      stream.getTracks().forEach((t) => t.stop());
      return false;
    }

    const mime = pickMimeType();
    let rec: MediaRecorder;
    try {
      rec = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream);
    } catch {
      stream.getTracks().forEach((t) => t.stop());
      return false;
    }
    if (seq !== startSeqRef.current) {
      stream.getTracks().forEach((t) => t.stop());
      return false;
    }

    streamRef.current = stream;
    recorderRef.current = rec;
    chunksRef.current = [];
    cancelledRef.current = false;
    startedAtRef.current = Date.now();

    rec.ondataavailable = (e) => {
      if (e.data.size > 0) chunksRef.current.push(e.data);
    };
    rec.onstop = () => {
      const resolve = resolveRef.current;
      resolveRef.current = null;
      if (cancelledRef.current) {
        cleanup();
        resolve?.(null);
        return;
      }
      const type = rec.mimeType || mime || "audio/webm";
      const blob = new Blob(chunksRef.current, { type });
      const file = new File([blob], `voice-message-${startedAtRef.current}.${extForMime(type)}`, { type });
      cleanup();
      resolve?.(blob.size > 0 ? file : null);
    };

    rec.start();
    setState("recording");
    setElapsedMs(0);
    timerRef.current = window.setInterval(() => {
      setElapsedMs(Date.now() - startedAtRef.current);
    }, 100);
    return true;
  }, [cleanup]);

  // stop finishes the recording and resolves with the produced File (or null).
  const stop = useCallback((): Promise<File | null> => {
    // Re-entrant calls (auto-finish tick + double "Done") get the same promise —
    // never a second rec.stop()/cleanup() that would wipe chunks mid-finalize.
    if (stopPromiseRef.current) return stopPromiseRef.current;

    const rec = recorderRef.current;
    if (!rec || rec.state === "inactive") {
      cleanup();
      return Promise.resolve(null);
    }

    startSeqRef.current++; // abort any in-flight start
    cancelledRef.current = false;
    const p = new Promise<File | null>((resolve) => {
      resolveRef.current = resolve;
      rec.stop();
    });
    stopPromiseRef.current = p;
    void p.finally(() => {
      stopPromiseRef.current = null;
    });
    return p;
  }, [cleanup]);

  // cancel discards the recording (no file produced).
  const cancel = useCallback(() => {
    startSeqRef.current++; // abort any in-flight start
    const rec = recorderRef.current;
    cancelledRef.current = true;
    if (rec && rec.state !== "inactive") {
      resolveRef.current = null;
      rec.stop(); // onstop sees cancelledRef and discards
    } else {
      cleanup();
    }
  }, [cleanup]);

  return { state, elapsedMs, start, stop, cancel };
}
