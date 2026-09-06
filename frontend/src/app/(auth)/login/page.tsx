"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Field } from "@/components/Field";
import { authApi, ApiRequestError } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await authApi("/login", { method: "POST", body: JSON.stringify({ email, password }) });
      const next = new URLSearchParams(window.location.search).get("next");
      router.push(next || "/dashboard");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Не удалось войти");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="card animate-rise">
      <h2 className="mb-5 text-xl font-bold">Вход</h2>
      <form onSubmit={onSubmit} className="space-y-4">
        <Field
          label="Email"
          name="email"
          type="email"
          autoComplete="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <Field
          label="Пароль"
          name="password"
          type="password"
          autoComplete="current-password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit" disabled={loading} className="btn-primary w-full">
          {loading ? "Входим…" : "Войти"}
        </button>
      </form>
      <div className="mt-4 flex items-center justify-between text-sm text-ink-500">
        <Link href="/forgot-password" className="hover:text-ink-100">
          Забыли пароль?
        </Link>
        <Link href="/register" className="hover:text-ink-100">
          Создать аккаунт
        </Link>
      </div>
    </div>
  );
}
