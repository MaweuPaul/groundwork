import type { Metadata } from "next";
import { LoginForm } from "./login-form";

export const metadata: Metadata = {
  title: "Sign in · Groundwork",
};

export default function LoginPage() {
  return (
    <>
      {/* The tabs above show which form this is, so the heading is for screen readers. */}
      <h1 className="sr-only">Sign in to Groundwork</h1>
      <LoginForm />
    </>
  );
}
