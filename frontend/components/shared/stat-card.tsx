import type { LucideIcon } from 'lucide-react';

import Border from '@/components/shared/border';

const StatCard = ({
  label,
  icon: Icon,
  value,
  unit,
  caption,
}: {
  label: string;
  icon: LucideIcon;
  value: string;
  unit?: string;
  caption: string;
}) => {
  return (
    <Border className="flex flex-col gap-3 p-5">
      <div className="flex items-center justify-between">
        <span className="font-mono text-micro tracking-[0.14em] text-muted-foreground uppercase">
          {label}
        </span>
        <Icon className="size-4 text-olive" />
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-[34px] leading-none font-semibold tracking-tight">{value}</span>
        {unit && <span className="font-mono text-caption text-muted-foreground">{unit}</span>}
      </div>
      <p className="text-label text-muted-foreground">{caption}</p>
    </Border>
  );
};

export default StatCard;
