import type { LucideIcon } from 'lucide-react';

const ErrorState = ({
  icon: Icon,
  code,
  title,
  message,
  detail,
  children,
}: {
  icon: LucideIcon;
  code: string;
  title: string;
  message: string;
  detail?: string;
  children?: React.ReactNode;
}) => {
  return (
    <div className="flex flex-1 items-center justify-center px-6 py-16">
      <div className="flex max-w-md flex-col items-center gap-4 text-center">
        <span className="flex size-14 items-center justify-center rounded-full bg-surface-container text-olive">
          <Icon className="size-7" />
        </span>
        <span className="rounded-sm bg-surface-container px-2 py-0.5 font-mono text-micro font-semibold tracking-[0.14em] text-muted-foreground uppercase">
          {code}
        </span>
        <h1 className="text-h1 font-semibold tracking-tight">{title}</h1>
        <p className="text-body text-muted-foreground">{message}</p>
        {detail && (
          <p className="rounded-sm bg-surface-container px-2 py-1 font-mono text-micro text-muted-foreground">
            {detail}
          </p>
        )}
        {children && <div className="mt-2 flex flex-wrap justify-center gap-2">{children}</div>}
      </div>
    </div>
  );
};

export default ErrorState;
