import type { InputHTMLAttributes, ReactNode } from "react";

interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: ReactNode;
}

// A labelled input with inline validation messaging, used across all forms.
export function Field({ label, error, hint, id, ...rest }: FieldProps) {
  const inputId = id ?? rest.name;
  return (
    <div>
      <label className="label" htmlFor={inputId}>
        {label}
      </label>
      <input id={inputId} className="input" aria-invalid={Boolean(error)} {...rest} />
      {error ? <p className="field-error">{error}</p> : null}
      {hint ? <p className="mt-1 text-sm text-ink-500">{hint}</p> : null}
    </div>
  );
}
