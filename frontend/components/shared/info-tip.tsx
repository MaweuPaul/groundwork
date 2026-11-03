'use client';

import { Info } from 'lucide-react';

import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';

// A click-to-open explanation next to a field label. Click rather than hover so it works on touch screens.
const InfoTip = ({ label, children }: { label: string; children: React.ReactNode }) => {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={`What is ${label}?`}
          className="inline-flex align-middle text-muted-foreground transition-colors hover:text-foreground"
        >
          <Info className="size-3.5" />
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" className="w-72 border-border bg-surface-lowest text-caption leading-relaxed normal-case tracking-normal">
        {children}
      </PopoverContent>
    </Popover>
  );
};

export default InfoTip;
