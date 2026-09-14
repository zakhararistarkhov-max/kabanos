"use client";

import { useEffect, useRef, useState } from "react";

interface ScannerControls {
  stop: () => void;
}

// BarcodeScanner streams the back camera and reports the first decoded barcode.
// The ZXing reader is imported dynamically so it only loads (and only touches
// browser APIs) on the client, when scanning actually starts. A manual-entry
// field is always available as a fallback (no camera / permission denied / iOS).
export function BarcodeScanner({ onDetected }: { onDetected: (code: string) => void }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const controlsRef = useRef<ScannerControls | null>(null);
  const detectedRef = useRef(onDetected);
  detectedRef.current = onDetected;

  const [error, setError] = useState<string | null>(null);
  const [manual, setManual] = useState("");

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const { BrowserMultiFormatReader } = await import("@zxing/browser");
        const reader = new BrowserMultiFormatReader();
        const controls = await reader.decodeFromConstraints(
          { video: { facingMode: "environment" } },
          videoRef.current!,
          (result) => {
            if (!result || cancelled) return;
            const text = result.getText().replace(/\D/g, "");
            if (text.length < 8) return;
            controlsRef.current?.stop();
            detectedRef.current(text);
          },
        );
        if (cancelled) {
          controls.stop();
          return;
        }
        controlsRef.current = controls;
      } catch {
        setError("Камера недоступна — введите штрихкод вручную.");
      }
    })();

    return () => {
      cancelled = true;
      try {
        controlsRef.current?.stop();
      } catch {
        /* ignore */
      }
    };
  }, []);

  function submitManual() {
    const code = manual.replace(/\D/g, "");
    if (code.length >= 8) onDetected(code);
  }

  return (
    <div className="space-y-3">
      {!error ? (
        <div className="relative overflow-hidden rounded-xl bg-black">
          {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
          <video ref={videoRef} className="mx-auto max-h-64 w-full object-cover" muted playsInline />
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <div className="h-24 w-4/5 rounded-lg border-2 border-brand/80 shadow-[0_0_0_9999px_rgba(0,0,0,0.35)]" />
          </div>
        </div>
      ) : (
        <p className="rounded-lg bg-ink-800/50 px-3 py-2 text-sm text-ink-400">{error}</p>
      )}

      <p className="text-center text-xs text-ink-500">Наведите камеру на штрихкод — или введите цифры вручную.</p>

      <div className="flex gap-2">
        <input
          className="input"
          inputMode="numeric"
          placeholder="Штрихкод (напр. 4600000000000)"
          value={manual}
          onChange={(e) => setManual(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              submitManual();
            }
          }}
        />
        <button onClick={submitManual} disabled={manual.replace(/\D/g, "").length < 8} className="btn-ghost shrink-0">
          Найти
        </button>
      </div>
    </div>
  );
}
