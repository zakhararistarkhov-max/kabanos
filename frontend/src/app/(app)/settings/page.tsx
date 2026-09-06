"use client";

import { useEffect, useState } from "react";
import { Field } from "@/components/Field";
import { api, ApiRequestError } from "@/lib/api";
import { useInvalidateSession, useSession } from "@/hooks/useSession";
import type { User } from "@/lib/types";

export default function SettingsPage() {
  const { data } = useSession();
  const invalidate = useInvalidateSession();
  const user = data?.user;

  const [displayName, setDisplayName] = useState("");
  const [heightCm, setHeightCm] = useState("");
  const [sex, setSex] = useState<"" | "male" | "female" | "other">("");
  const [telegram, setTelegram] = useState("");
  const [status, setStatus] = useState<null | "ok" | string>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!user) return;
    setDisplayName(user.displayName ?? "");
    setHeightCm(user.heightCm != null ? String(user.heightCm) : "");
    setSex((user.sex as typeof sex) ?? "");
    setTelegram(user.telegramUsername ?? "");
  }, [user]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setStatus(null);
    const body: Record<string, unknown> = { displayName };
    if (heightCm) body.heightCm = parseFloat(heightCm.replace(",", "."));
    if (sex) body.sex = sex;
    body.telegramUsername = telegram.replace(/^@/, "");
    try {
      await api<User>("/me", { method: "PATCH", body: JSON.stringify(body) });
      invalidate();
      setStatus("ok");
    } catch (err) {
      setStatus(err instanceof ApiRequestError ? err.message : "Не удалось сохранить");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Настройки</h1>
        <p className="text-sm text-ink-500">Профиль и параметры для расчётов.</p>
      </div>

      <form onSubmit={save} className="card space-y-4">
        <Field label="Имя" name="displayName" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
        <Field
          label="Рост, см"
          name="heightCm"
          inputMode="decimal"
          value={heightCm}
          onChange={(e) => setHeightCm(e.target.value)}
          hint="Используется для расчёта ИМТ"
        />
        <div>
          <label className="label" htmlFor="sex">
            Пол
          </label>
          <select id="sex" className="input" value={sex} onChange={(e) => setSex(e.target.value as typeof sex)}>
            <option value="">Не указан</option>
            <option value="male">Мужской</option>
            <option value="female">Женский</option>
            <option value="other">Другой</option>
          </select>
        </div>
        <Field
          label="Telegram"
          name="telegram"
          placeholder="username"
          value={telegram}
          onChange={(e) => setTelegram(e.target.value)}
          hint="Для утреннего дайджеста (скоро)"
        />
        <div className="flex items-center gap-3">
          <button type="submit" disabled={saving} className="btn-primary">
            {saving ? "Сохраняем…" : "Сохранить"}
          </button>
          {status === "ok" ? <span className="text-sm text-good">Сохранено ✓</span> : null}
          {status && status !== "ok" ? <span className="text-sm text-bad">{status}</span> : null}
        </div>
      </form>

      <div className="card text-sm text-ink-500">
        <div className="text-ink-300">Аккаунт</div>
        <div className="mt-1">{user?.email}</div>
        <div className="mt-1">{user?.emailVerified ? "Email подтверждён ✓" : "Email не подтверждён"}</div>
      </div>
    </div>
  );
}
