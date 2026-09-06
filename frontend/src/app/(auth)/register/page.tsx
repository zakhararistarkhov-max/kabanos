"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Field } from "@/components/Field";
import { authApi, ApiRequestError } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState({ displayName: "", email: "", password: "" });
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function update(key: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.target.value });
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setFields({});
    try {
      await authApi("/register", { method: "POST", body: JSON.stringify(form) });
      router.push("/dashboard");
      router.refresh();
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message);
        if (err.fields) setFields(err.fields);
      } else {
        setError("Не удалось зарегистрироваться");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="card animate-rise">
      <h2 className="mb-5 text-xl font-bold">Регистрация</h2>
      <form onSubmit={onSubmit} className="space-y-4">
        <Field label="Имя" name="displayName" value={form.displayName} onChange={update("displayName")} error={fields.displayName} />
        <Field label="Email" name="email" type="email" autoComplete="email" required value={form.email} onChange={update("email")} error={fields.email} />
        <Field
          label="Пароль"
          name="password"
          type="password"
          autoComplete="new-password"
          required
          value={form.password}
          onChange={update("password")}
          error={fields.password}
          hint="Минимум 8 символов"
        />
        {error ? <p className="field-error">{error}</p> : null}
        <button type="submit" disabled={loading} className="btn-primary w-full">
          {loading ? "Создаём…" : "Создать аккаунт"}
        </button>
      </form>
      <p className="mt-4 text-center text-sm text-ink-500">
        Уже есть аккаунт?{" "}
        <Link href="/login" className="text-brand hover:text-brand-soft">
          Войти
        </Link>
      </p>
    </div>
  );
}
