import type { Metadata } from "next";
import { RegisterForm } from "./register-form";

export const metadata: Metadata = {
  title: "Create account · Groundwork",
};

export default function RegisterPage() {
  return (
    <>
      {/* The tabs above show which form this is, so the heading is for screen readers. */}
      <h1 className="sr-only">Create a Groundwork account</h1>
      <RegisterForm />
    </>
  );
}
