import { cn } from '@/lib/utils';

const Border = ({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) => {
  return (
    <div
      className={cn(
        'rounded-md border border-border bg-surface-lowest',
        className,
      )}
    >
      {children}
    </div>
  );
};

export default Border;
