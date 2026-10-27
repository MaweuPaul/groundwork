"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const tabs = [
  { href: "/login", label: "Sign In" },
  { href: "/register", label: "Create Account" },
];

// Segmented control from the design; each tab is a real route.
export function AuthTabs() {
  const pathname = usePathname();

  return (
    <nav aria-label="Authentication" className="flex items-center rounded-lg bg-surface-container p-1 shadow-sm">
      {tabs.map((tab) => {
        const active = pathname === tab.href;
        return (
          <Link
            key={tab.href}
            href={tab.href}
            aria-current={active ? "page" : undefined}
            className={
              active
                ? "flex-1 rounded bg-surface-lowest py-2 text-center text-body font-medium text-primary shadow-sm"
                : "flex-1 rounded py-2 text-center text-body font-medium text-muted transition-colors hover:text-foreground"
            }
          >
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
