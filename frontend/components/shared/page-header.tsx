import type { LucideIcon } from 'lucide-react';

const PageHeader = ({
  eyebrow,
  icon: Icon,
  badge,
  title,
  description,
  actions,
}: {
  eyebrow?: string;
  icon?: LucideIcon;
  badge?: string;
  title: string;
  description?: string;
  actions?: React.ReactNode;
}) => {
  return (
    <div className="flex flex-wrap items-end justify-between gap-4">
      <div className="flex max-w-2xl items-start gap-3">
        {Icon && (
          <span className="mt-1 flex size-11 shrink-0 items-center justify-center rounded-md bg-primary text-sidebar-primary">
            <Icon className="size-5" />
          </span>
        )}
        <div>
          {eyebrow && (
            <span className="mb-2 inline-block rounded-sm bg-surface-container px-2 py-0.5 font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase">
              {eyebrow}
            </span>
          )}
          <h1 className="flex flex-wrap items-center gap-3 text-display font-semibold tracking-tight">
            {title}
            {badge && (
              <span className="rounded-sm bg-secondary-container px-2 py-0.5 font-mono text-micro font-semibold tracking-[0.12em] text-on-secondary-container uppercase">
                {badge}
              </span>
            )}
          </h1>
          {description && <p className="mt-1 text-body text-muted-foreground">{description}</p>}
        </div>
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
};

export default PageHeader;
