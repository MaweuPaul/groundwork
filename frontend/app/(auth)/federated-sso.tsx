import { GlobeIcon, KeyIcon } from "./icons";

const providers = [
  { label: "Copernicus / ESA", short: "Copernicus", icon: <GlobeIcon className="text-primary" /> },
  { label: "Hardware Key (FIDO2)", short: "Security key", icon: <KeyIcon className="text-secondary" /> },
];

// Placeholder for federated sign-in. Disabled until the API supports it.
export function FederatedSSO() {
  return (
    <div className="space-y-3">
      <div className="relative flex items-center justify-center">
        <div className="h-px w-full bg-surface-high" />
        <span className="absolute bg-surface-lowest px-3 font-mono text-micro tracking-wider text-outline uppercase">
          Federated audit SSO · coming soon
        </span>
      </div>
      <div className="grid grid-cols-2 gap-3">
        {providers.map((p) => (
          <button
            key={p.label}
            type="button"
            disabled
            title="Coming soon"
            className="flex w-full cursor-not-allowed items-center justify-center gap-2 rounded border border-outline-variant/30 bg-surface-container px-3 py-2 text-label font-medium text-foreground opacity-60"
          >
            {p.icon}
            <span className="truncate sm:hidden">{p.short}</span>
            <span className="hidden truncate sm:inline">{p.label}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
