"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { REMEMBER_COOKIE, sessionCookies, type TokenPair } from "@/lib/session";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

export type AuthFormState = {
  error?: string;
  fieldErrors?: Partial<Record<string, string>>;
  values?: Record<string, string>;
};

type ApiResult =
  | { ok: true; data: TokenPair }
  | { ok: false; status: number; error: string };

async function postJSON(path: string, body: unknown): Promise<ApiResult> {
  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    return { ok: false, status: 0, error: "Can't reach the server. Try again in a moment." };
  }

  const json = await res.json().catch(() => ({}));
  if (!res.ok) {
    return { ok: false, status: res.status, error: json.error ?? "Something went wrong." };
  }
  return { ok: true, data: json as TokenPair };
}

async function storeTokens(tokens: TokenPair, remember: boolean) {
  const cookieStore = await cookies();
  for (const cookie of sessionCookies(tokens, remember)) {
    cookieStore.set(cookie);
  }
  // A previous "remember me" login on this browser mustn't carry over.
  if (!remember) cookieStore.delete(REMEMBER_COOKIE);
}

function field(formData: FormData, name: string) {
  return String(formData.get(name) ?? "").trim();
}

export async function login(_prev: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const email = field(formData, "email");
  const password = String(formData.get("password") ?? "");
  const remember = formData.get("remember") === "on";
  const values = { email, remember: remember ? "on" : "off" };

  const fieldErrors: Record<string, string> = {};
  if (!email) fieldErrors.email = "Enter your email.";
  if (!password) fieldErrors.password = "Enter your password.";
  if (Object.keys(fieldErrors).length) return { fieldErrors, values };

  const result = await postJSON("/auth/login", { email, password });
  if (!result.ok) {
    const error =
      result.status === 401 ? "That email and password don't match an account." : result.error;
    return { error, values };
  }

  await storeTokens(result.data, remember);
  redirect("/");
}

export async function register(_prev: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const values = {
    first_name: field(formData, "first_name"),
    last_name: field(formData, "last_name"),
    username: field(formData, "username"),
    email: field(formData, "email"),
  };
  const password = String(formData.get("password") ?? "");
  const role = field(formData, "role");

  const fieldErrors: Record<string, string> = {};
  if (role !== "auditor" && role !== "asset_holder") fieldErrors.role = "Choose an account profile.";
  if (!values.first_name) fieldErrors.first_name = "Enter your first name.";
  if (!values.last_name) fieldErrors.last_name = "Enter your last name.";
  if (!values.username) fieldErrors.username = "Choose a username.";
  if (!values.email) fieldErrors.email = "Enter your email.";
  // Same rules as the strength meter in form-field.tsx.
  if (password.length < 10 || !/[A-Z]/.test(password) || !/[0-9]/.test(password)) {
    fieldErrors.password = "Use at least 10 characters, with an uppercase letter and a number.";
  }
  if (Object.keys(fieldErrors).length) return { fieldErrors, values };

  const result = await postJSON("/auth/register", { ...values, password, role });
  if (!result.ok) {
    if (result.status === 409) {
      const fieldErrors = result.error.includes("username")
        ? { username: "This username is taken." }
        : { email: "This email is already registered." };
      return { fieldErrors, values };
    }
    return { error: result.error, values };
  }

  await storeTokens(result.data, true);
  redirect("/");
}
