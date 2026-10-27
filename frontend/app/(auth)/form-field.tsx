"use client";

import { useState } from "react";
import { CheckIcon, DotIcon, EyeIcon, EyeOffIcon, LockIcon } from "./icons";

type FieldProps = React.InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  name: string;
  error?: string;
  icon?: React.ReactNode;
  aside?: React.ReactNode;
  trailing?: React.ReactNode;
  children?: React.ReactNode;
};

export function Field({
  label,
  name,
  error,
  icon,
  aside,
  trailing,
  children,
  className,
  ...input
}: FieldProps) {
  const id = `field-${name}`;

  return (
    <div className={`space-y-1 ${className ?? ""}`}>
      <div className="flex items-center justify-between gap-2">
        <label
          htmlFor={id}
          className="text-label font-medium tracking-wider text-muted uppercase"
        >
          {label}
        </label>
        {aside}
      </div>
      <div
        className={`flex items-center gap-2 rounded border bg-surface-low px-3 py-2 transition-colors focus-within:border-primary ${
          error ? "border-danger" : "border-outline-variant/50"
        }`}
      >
        {icon ? <span className="text-outline">{icon}</span> : null}
        <input
          id={id}
          name={name}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${id}-error` : undefined}
          className="w-full bg-transparent text-body text-foreground outline-none placeholder:text-outline"
          {...input}
        />
        {trailing}
      </div>
      {error ? (
        <p id={`${id}-error`} className="text-caption text-danger">
          {error}
        </p>
      ) : null}
      {children}
    </div>
  );
}

export function PasswordField(
  props: Omit<FieldProps, "type" | "icon" | "trailing"> & { showStrength?: boolean },
) {
  const { showStrength, onChange, ...rest } = props;
  const [visible, setVisible] = useState(false);
  const [value, setValue] = useState("");

  return (
    <Field
      {...rest}
      type={visible ? "text" : "password"}
      icon={<LockIcon />}
      onChange={(e) => {
        setValue(e.target.value);
        onChange?.(e);
      }}
      trailing={
        <button
          type="button"
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? "Hide password" : "Show password"}
          className="flex items-center text-outline transition-colors hover:text-foreground"
        >
          {visible ? <EyeOffIcon /> : <EyeIcon />}
        </button>
      }
    >
      {showStrength ? <StrengthMeter value={value} /> : null}
    </Field>
  );
}

export const passwordRules = [
  { label: "≥10 chars", test: (v: string) => v.length >= 10 },
  { label: "1 Uppercase", test: (v: string) => /[A-Z]/.test(v) },
  { label: "1 Number", test: (v: string) => /[0-9]/.test(v) },
];

function StrengthMeter({ value }: { value: string }) {
  const results = passwordRules.map((rule) => rule.test(value));
  const score = results.filter(Boolean).length;

  return (
    <div className="space-y-1 pt-1">
      <div className="flex items-center gap-1.5" aria-hidden>
        {passwordRules.map((rule, i) => (
          <div
            key={rule.label}
            className={`h-1 flex-1 rounded-full ${score > i ? "bg-secondary" : "bg-surface-high"}`}
          />
        ))}
      </div>
      <ul className="flex items-center justify-between font-mono text-micro">
        {passwordRules.map((rule, i) => (
          <li
            key={rule.label}
            className={`flex items-center gap-0.5 ${results[i] ? "text-secondary" : "text-outline"}`}
          >
            {results[i] ? <CheckIcon size={11} /> : <DotIcon size={11} />}
            {rule.label}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function FormError({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <p
      role="alert"
      className="rounded border border-danger/20 bg-danger-container px-3 py-2.5 text-body text-on-danger-container"
    >
      {message}
    </p>
  );
}

export function SubmitButton({
  pending,
  icon,
  children,
}: {
  pending: boolean;
  icon?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <button
      type="submit"
      disabled={pending}
      className="flex w-full items-center justify-center gap-2 rounded bg-primary-container px-4 py-2.5 text-body font-medium text-on-primary shadow-sm transition-all hover:bg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-60"
    >
      {icon ? <span className="text-secondary-fixed">{icon}</span> : null}
      {children}
    </button>
  );
}
