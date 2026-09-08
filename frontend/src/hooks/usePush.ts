"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

// Converts a base64url VAPID public key into the Uint8Array the Push API wants.
function urlB64ToUint8Array(base64String: string): Uint8Array {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(base64);
  const out = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

export interface PushState {
  supported: boolean; // browser has SW + Push API
  secure: boolean; // secure context (HTTPS or localhost)
  configured: boolean; // server has VAPID keys
  checkFailed: boolean; // couldn't reach the server to check (≠ not configured)
  subscribed: boolean;
  loading: boolean;
  error: string | null;
}

export function usePush() {
  const [state, setState] = useState<PushState>({
    supported: false,
    secure: true,
    configured: true,
    checkFailed: false,
    subscribed: false,
    loading: true,
    error: null,
  });

  const refresh = useCallback(async () => {
    const secure = typeof window !== "undefined" && window.isSecureContext;
    const supported =
      typeof window !== "undefined" &&
      "serviceWorker" in navigator &&
      "PushManager" in window &&
      "Notification" in window;
    if (!supported) {
      setState((s) => ({ ...s, supported: false, secure, loading: false }));
      return;
    }
    let subscribed = false;
    try {
      const reg = await navigator.serviceWorker.getRegistration();
      if (reg) subscribed = Boolean(await reg.pushManager.getSubscription());
    } catch {
      /* ignore */
    }
    // Separate "server says push is off" from "we couldn't ask the server": a
    // transient error (e.g. an auth blip) must not be reported as missing VAPID.
    let configured = false;
    let checkFailed = false;
    try {
      const k = await api<{ enabled: boolean }>("/push/key");
      configured = k.enabled;
    } catch {
      checkFailed = true;
    }
    setState({ supported: true, secure, configured, checkFailed, subscribed, loading: false, error: null });
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const enable = useCallback(async () => {
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const { enabled, key } = await api<{ enabled: boolean; key: string }>("/push/key");
      if (!enabled || !key) throw new Error("Push-уведомления не настроены на сервере");
      const perm = await Notification.requestPermission();
      if (perm !== "granted") throw new Error("Вы не разрешили уведомления в браузере");
      const reg = await navigator.serviceWorker.register("/sw.js");
      await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlB64ToUint8Array(key),
      });
      const json = sub.toJSON();
      await api("/push/subscribe", {
        method: "POST",
        body: JSON.stringify({ endpoint: json.endpoint, keys: json.keys }),
      });
      setState((s) => ({ ...s, subscribed: true, loading: false }));
    } catch (e) {
      setState((s) => ({ ...s, loading: false, error: e instanceof Error ? e.message : "Не удалось включить уведомления" }));
    }
  }, []);

  const disable = useCallback(async () => {
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const reg = await navigator.serviceWorker.getRegistration();
      const sub = reg ? await reg.pushManager.getSubscription() : null;
      if (sub) {
        await api("/push/unsubscribe", { method: "POST", body: JSON.stringify({ endpoint: sub.endpoint }) }).catch(() => undefined);
        await sub.unsubscribe().catch(() => undefined);
      }
      setState((s) => ({ ...s, subscribed: false, loading: false }));
    } catch {
      setState((s) => ({ ...s, loading: false, error: "Не удалось отключить уведомления" }));
    }
  }, []);

  const test = useCallback(async () => {
    try {
      await api("/push/test", { method: "POST" });
    } catch {
      /* ignore */
    }
  }, []);

  return { ...state, enable, disable, test, refresh };
}
