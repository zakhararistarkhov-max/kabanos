"use client";

import Link from "next/link";
import { useState } from "react";
import { Field } from "@/components/Field";
import { authApi } from "@/lib/api";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    // The API always responds success to avoid leaking which emails exist.
    await authApi("/forgot-password", { method: "POST", body: JSON.stringify({ email }) }).catch(() => undefined);
    setSent(true);
    setLoading(false);
  }

  return (
    <div className="card animate-rise">
      <h2 className="mb-5 text-xl font-bold">Восстановление пароля</h2>
      {sent ? (
        <p className="text-ink-300">
          Если аккаунт с адресом <span className="text-ink-100">{email}</span> существует, мы отправили на него ссылку для
          сброса пароля.
        </p>
      ) : (
        <form onSubmit={onSubmit} className="space-y-4">
          <Field label="Email" name="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
          <button type="submit" disabled={loading} className="btn-primary w-full">
            {loading ? "Отправляем…" : "Отправить ссылку"}
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
