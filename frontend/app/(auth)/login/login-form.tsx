"use client";

import Link from "next/link";
import { useActionState } from "react";
import { login, type AuthFormState } from "../actions";
import { FederatedSSO } from "../federated-sso";
import { Field, FormError, PasswordField, SubmitButton } from "../form-field";
import { ArrowIcon, MailIcon } from "../icons";

const initialState: AuthFormState = {};

export function LoginForm() {
  const [state, formAction, pending] = useActionState(login, initialState);

  return (
    <form action={formAction} noValidate className="space-y-3">
      <FormError message={state.error} />

      <Field
        label="Email Address"
        name="email"
        type="email"
        autoComplete="email"
        placeholder="you@organization.com"
        required
        icon={<MailIcon />}
        defaultValue={state.values?.email}
        error={state.fieldErrors?.email}
      />

      <PasswordField
        label="Password"
        name="password"
        autoComplete="current-password"
        required
        error={state.fieldErrors?.password}
        aside={
          <Link
            href="/forgot-password"
            className="text-caption font-medium text-primary transition-colors hover:text-primary-container"
          >
            Forgot password?
          </Link>
        }
      />

      <label className="flex cursor-pointer items-center gap-2 py-0.5 select-none">
        <input
          type="checkbox"
          name="remember"
          defaultChecked={state.values?.remember !== "off"}
          className="size-4 rounded accent-primary-container"
        />
        <span className="text-caption text-muted">Keep me signed in for 30 days</span>
      </label>

      <div>
        <SubmitButton pending={pending} icon={<ArrowIcon size={20} />}>
          {pending ? "Signing in…" : "Sign In"}
        </SubmitButton>
      </div>

      <p className="pt-1 text-center text-label text-muted">
        Don&apos;t have an account?{" "}
        <Link href="/register" className="font-medium text-primary underline underline-offset-2">
          Create Account
        </Link>
      </p>

      <FederatedSSO />
    </form>
  );
}
