"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Field } from "@/components/Field";
import { authApi, ApiRequestError } from "@/lib/api";

export default function ResetPasswordPage() {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setToken(new URLSearchParams(window.location.search).get("token") ?? "");
  }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await authApi("/reset-password", { method: "POST", body: JSON.stringify({ token, password }) });
      setDone(true);
      setTimeout(() => router.push("/login"), 1500);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Не удалось сбросить пароль");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="card animate-rise">
      <h2 className="mb-5 text-xl font-bold">Новый пароль</h2>
      {done ? (
        <p className="text-good">Пароль обновлён. Перенаправляем на страницу входа…</p>
      ) : (
        <form onSubmit={onSubmit} className="space-y-4">
          <Field
            label="Новый пароль"
            name="password"
            type="password"
            autoComplete="new-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            hint="Минимум 8 символов"
          />
          {!token ? <p className="field-error">Ссылка недействительна: отсутствует токен.</p> : null}
          {error ? <p className="field-error">{error}</p> : null}
          <button type="submit" disabled={loading || !token} className="btn-primary w-full">
            {loading ? "Сохраняем…" : "Сохранить пароль"}
          </button>
        </form>
      )}
      <p className="mt-4 text-center text-sm text-ink-500">
        <Link href="/login" className="hover:text-ink-100">
          Вернуться ко входу
        </Link>
      </p>
    </div>
  );
}
