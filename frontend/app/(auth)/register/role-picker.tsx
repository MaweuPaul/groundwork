"use client";

import { LandscapeIcon, ShieldIcon } from "../icons";

export type Role = "auditor" | "asset_holder";

const roles: { value: Role; title: string; subtitle: string; icon: typeof ShieldIcon }[] = [
  { value: "auditor", title: "Auditor", subtitle: "Verifier Engine", icon: ShieldIcon },
  { value: "asset_holder", title: "Asset Holder", subtitle: "Land Registry", icon: LandscapeIcon },
];

// Native radios styled as the design's two tiles, so the choice is
// submitted with the form and works with the keyboard.
export function RolePicker({
  value,
  onChange,
  error,
}: {
  value: Role;
  onChange: (role: Role) => void;
  error?: string;
}) {
  return (
    <fieldset className="space-y-1">
      <legend className="mb-1 text-label font-medium tracking-wider text-muted uppercase">
        Account Profile
      </legend>
      <div className="grid grid-cols-2 gap-3">
        {roles.map(({ value: role, title, subtitle, icon: RoleIcon }) => {
          const selected = value === role;
          return (
            <label
              key={role}
              className={`flex cursor-pointer items-center gap-3 rounded px-3 py-2 shadow-sm transition-all has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-primary ${
                selected
                  ? "bg-primary-container ring-1 ring-secondary-container"
                  : "bg-surface-container hover:bg-surface-high"
              }`}
            >
              <input
                type="radio"
                name="role"
                value={role}
                checked={selected}
                onChange={() => onChange(role)}
                className="sr-only"
              />
              <RoleIcon size={22} className={selected ? "text-secondary-fixed" : "text-secondary"} />
              <span className="min-w-0">
                <span
                  className={`block truncate text-[15px] leading-5 font-semibold ${
                    selected ? "text-surface" : "text-foreground"
                  }`}
                >
                  {title}
                </span>
                <span
                  className={`block truncate text-caption ${
                    selected ? "text-primary-fixed-dim" : "text-muted"
                  }`}
                >
                  {subtitle}
                </span>
              </span>
            </label>
          );
        })}
      </div>
      {error ? <p className="text-caption text-danger">{error}</p> : null}
    </fieldset>
  );
}
