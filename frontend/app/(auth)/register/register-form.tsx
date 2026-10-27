"use client";

import Link from "next/link";
import { useActionState, useState } from "react";
import { register, type AuthFormState } from "../actions";
import { FederatedSSO } from "../federated-sso";
import { Field, FormError, PasswordField, SubmitButton } from "../form-field";
import { AtIcon, CheckIcon, MailIcon } from "../icons";
import { RolePicker, type Role } from "./role-picker";

const initialState: AuthFormState = {};

export function RegisterForm() {
  const [state, formAction, pending] = useActionState(register, initialState);
  const errors = state.fieldErrors ?? {};
  const values = state.values ?? {};
  const [role, setRole] = useState<Role>("auditor");

  return (
    <form action={formAction} noValidate className="space-y-3">
      <FormError message={state.error} />

      <RolePicker value={role} onChange={setRole} error={errors.role} />

      <div className="grid grid-cols-2 gap-3">
        <Field
          label="First Name"
          name="first_name"
          autoComplete="given-name"
          placeholder="First name"
          required
          defaultValue={values.first_name}
          error={errors.first_name}
        />
        <Field
          label="Last Name"
          name="last_name"
          autoComplete="family-name"
          placeholder="Last name"
          required
          defaultValue={values.last_name}
          error={errors.last_name}
        />
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-[2fr_3fr]">
        <Field
          label="Username"
          name="username"
          autoComplete="username"
          placeholder="your-handle"
          required
          icon={<AtIcon />}
          defaultValue={values.username}
          error={errors.username}
        />
        <Field
          label="Work Email"
          name="email"
          type="email"
          autoComplete="email"
          placeholder="you@organization.com"
          required
          icon={<MailIcon />}
          defaultValue={values.email}
          error={errors.email}
        />
      </div>

      <PasswordField
        label="Password"
        name="password"
        autoComplete="new-password"
        required
        showStrength
        error={errors.password}
      />

      <div>
        <SubmitButton pending={pending} icon={<CheckIcon size={20} />}>
          {pending
            ? "Creating account…"
            : role === "auditor"
              ? "Create Auditor Account"
              : "Create Asset Holder Account"}
        </SubmitButton>
      </div>

      <p className="pt-1 text-center text-label text-muted">
        Already have an account?{" "}
        <Link href="/login" className="font-medium text-primary underline underline-offset-2">
          Sign In
        </Link>
      </p>

      <FederatedSSO />
    </form>
  );
}
