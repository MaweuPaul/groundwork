import { AuthTabs } from "./auth-tabs";
import { CompassIcon, LeafIcon, SatelliteIcon, ShieldIcon } from "./icons";

// Split layout shared by sign-in and create-account, following the
// "Groundwork Auth — Desktop Split View" Stitch screen. The page is locked
// to the viewport; only very short screens fall back to scrolling the form.
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-dvh w-full overflow-hidden">
      <EvidencePanel />

      <div className="flex w-full flex-col overflow-y-auto bg-surface px-4 py-5 sm:px-8 lg:w-1/2 lg:px-14 lg:py-6 xl:px-20">
        <div className="mx-auto my-auto w-full max-w-xl space-y-4">
          <header className="flex items-center justify-between border-b border-surface-high pb-4">
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded bg-primary text-secondary-fixed shadow-sm">
                <LeafIcon size={22} />
              </div>
              <div>
                <span className="block text-[18px] leading-6 font-semibold tracking-tight">
                  GROUNDWORK
                </span>
                <span className="block font-mono text-micro text-outline">
                  SATELLITE LAND VERIFICATION
                </span>
              </div>
            </div>
          </header>

          <AuthTabs />

          <section className="rounded-xl border border-outline-variant/40 bg-surface-lowest p-5 shadow-sm lg:px-7 lg:py-6">
            {children}
          </section>
        </div>

        <footer className="pt-4">
          <div className="mx-auto flex max-w-xl flex-wrap items-center justify-center gap-x-6 gap-y-1 border-t border-surface-high pt-3 font-mono text-micro text-outline">
            <span>GROUNDWORK © {new Date().getFullYear()}</span>
            <span aria-hidden>·</span>
            <span>SENTINEL-2 L2A PROCESSING</span>
          </div>
        </footer>
      </div>
    </div>
  );
}

function EvidencePanel() {
  return (
    <aside className="relative hidden w-1/2 flex-col justify-between overflow-hidden bg-primary p-12 shadow-2xl lg:flex">
      {/* Source image has a white frame; scaling crops it out. */}
      <div
        className="absolute inset-0 scale-[1.35] bg-cover bg-center opacity-80 mix-blend-luminosity"
        style={{ backgroundImage: "url('/auth-satellite.jpg')" }}
        aria-hidden
      />
      <div
        className="pointer-events-none absolute inset-0 bg-linear-to-t from-primary via-primary/50 to-primary/30"
        aria-hidden
      />

      <div className="relative z-10 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3 rounded border border-outline-variant/30 bg-surface-lowest/90 px-3 py-1.5 shadow-sm backdrop-blur-sm">
          <span className="size-2 animate-pulse rounded-full bg-secondary" aria-hidden />
          <span className="font-mono text-caption font-medium tracking-wider text-primary">
            SENTINEL-2 L2A BOA REFLECTANCE
          </span>
        </div>
        <div className="flex items-center gap-2 rounded border border-surface-tint/40 bg-primary-container/90 px-3 py-1.5 font-mono text-micro text-surface shadow-sm backdrop-blur-sm">
          <SatelliteIcon size={14} className="text-primary-fixed" />
          <span>PARCEL ID: #8492-FR</span>
        </div>
      </div>

      <div className="relative z-10 my-auto flex items-center justify-center py-8">
        <div className="relative aspect-16/10 w-full max-w-lg">
          <svg
            className="size-full text-secondary-container drop-shadow-md"
            viewBox="0 0 400 240"
            fill="none"
            preserveAspectRatio="none"
            aria-hidden
          >
            <polygon
              points="35,25 150,15 260,65 370,40 385,190 230,225 70,175 25,120"
              fill="currentColor"
              fillOpacity="0.14"
              stroke="currentColor"
              strokeDasharray="4 4"
              strokeWidth="1.5"
            />
            {[
              [150, 15],
              [370, 40],
              [230, 225],
              [70, 175],
            ].map(([cx, cy]) => (
              <circle key={`${cx}-${cy}`} cx={cx} cy={cy} r="4" fill="currentColor" />
            ))}
            <path d="M180,110 L220,110 M200,90 L200,130" stroke="currentColor" strokeWidth="1.2" />
            <circle
              cx="200"
              cy="110"
              r="16"
              stroke="currentColor"
              strokeDasharray="2 2"
              strokeWidth="0.8"
            />
          </svg>
          <div className="absolute top-4 right-6 rounded bg-primary-container/80 px-2.5 py-1 font-mono text-micro text-secondary-fixed backdrop-blur-sm">
            NDVI: 0.784 (HEALTHY CANOPY)
          </div>
        </div>
      </div>

      <div className="relative z-10 space-y-4">
        <div className="max-w-lg space-y-2">
          <div className="inline-flex items-center gap-2 font-mono text-micro tracking-widest text-secondary-fixed uppercase">
            <ShieldIcon size={14} />
            <span>Reproducible satellite evidence</span>
          </div>
          <h2 className="text-display font-semibold tracking-tight text-surface drop-shadow-sm">
            Trust the ground, not the paperwork.
          </h2>
          <p className="text-body text-primary-fixed-dim">
            Land claims verified against versioned satellite measurements, auditable down to the
            exact scene.
          </p>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-primary-container/80 pt-4 font-mono text-caption text-primary-fixed-dim">
          <div className="flex items-center gap-2">
            <CompassIcon size={16} className="text-secondary-fixed" />
            <span>LAT 44°50&apos;16&quot;N · LON 0°34&apos;45&quot;W</span>
          </div>
          <span className="tracking-wider text-secondary-fixed uppercase">
            Verified satellite evidence
          </span>
        </div>
      </div>
    </aside>
  );
}
