import type { Metadata } from "next";
import { DemoButton } from "./demo-button";
import { LoginForm } from "./login-form";

export const metadata: Metadata = {
  title: "Sign in · Groundwork",
};

const API_URL = process.env.API_URL ?? "http://localhost:8080";

async function demoAvailable() {
  try {
    const res = await fetch(`${API_URL}/auth/demo`, { cache: "no-store" });
    return res.ok && (await res.json()).available === true;
  } catch {
    return false;
  }
}

export default async function LoginPage() {
  const demo = await demoAvailable();

  return (
    <>
      {/* The tabs above show which form this is, so the heading is for screen readers. */}
      <h1 className="sr-only">Sign in to Groundwork</h1>
      {demo && (
        <div className="mb-4 space-y-4">
          <DemoButton />
          <div className="relative flex items-center justify-center">
            <div className="h-px w-full bg-surface-high" />
            <span className="absolute bg-surface-lowest px-3 font-mono text-micro tracking-wider text-outline uppercase">
              or sign in
            </span>
          </div>
        </div>
      )}
      <LoginForm />
    </>
  );
}
