"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { authApi, ApiRequestError } from "@/lib/api";

type Status = "verifying" | "ok" | "error";

export default function VerifyEmailPage() {
  const [status, setStatus] = useState<Status>("verifying");
  const [message, setMessage] = useState("");

  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get("token");
    if (!token) {
      setStatus("error");
      setMessage("Ссылка недействительна: отсутствует токен.");
      return;
    }
    authApi("/verify-email", { method: "POST", body: JSON.stringify({ token }) })
      .then(() => setStatus("ok"))
      .catch((err) => {
        setStatus("error");
        setMessage(err instanceof ApiRequestError ? err.message : "Не удалось подтвердить email.");
      });
  }, []);

  return (
    <div className="card animate-rise text-center">
      {status === "verifying" && <p className="text-ink-300">Подтверждаем ваш email…</p>}
      {status === "ok" && (
        <>
          <div className="mb-3 text-4xl">✅</div>
          <h2 className="mb-2 text-xl font-bold">Email подтверждён</h2>
          <p className="mb-5 text-ink-500">Спасибо! Теперь ваш аккаунт полностью активен.</p>
          <Link href="/dashboard" className="btn-primary w-full">
            Перейти в приложение
          </Link>
        </>
      )}
      {status === "error" && (
        <>
          <div className="mb-3 text-4xl">⚠️</div>
          <h2 className="mb-2 text-xl font-bold">Не получилось</h2>
          <p className="mb-5 text-ink-500">{message}</p>
          <Link href="/dashboard" className="btn-ghost w-full">
            В приложение
          </Link>
        </>
      )}
    </div>
  );
}
