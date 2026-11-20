import { cn } from '@/lib/utils';

const Border = ({
  children,
  className,
  id,
}: {
  children: React.ReactNode;
  className?: string;
  id?: string;
}) => {
  return (
    <div
      id={id}
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
