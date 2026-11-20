"use client";

import { useActionState } from "react";
import { demoLogin, type AuthFormState } from "../actions";
import { FormError } from "../form-field";
import { ArrowIcon } from "../icons";

const initialState: AuthFormState = {};

export function DemoButton() {
  const [state, formAction, pending] = useActionState(demoLogin, initialState);

  return (
    <form action={formAction} className="space-y-2">
      <FormError message={state.error} />
      <button
        type="submit"
        disabled={pending}
        className="flex w-full items-center justify-center gap-2 rounded border border-primary/30 bg-secondary-container px-4 py-2.5 text-body font-medium text-on-secondary-container transition-colors hover:border-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {pending ? "Opening the demo…" : "Try the demo"}
        <ArrowIcon size={18} />
      </button>
      <p className="text-center text-caption text-muted-foreground">
        No account needed. The demo account is shared, so anything you add there can be seen by
        everyone who tries it.
      </p>
    </form>
  );
}
