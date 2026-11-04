'use client';

import { CalendarDays, ChevronDown } from 'lucide-react';
import { useState } from 'react';

import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';

// Values are calendar days as "YYYY-MM-DD". They're turned into local-midnight Dates for
// the calendar and back, so no timezone can shift the day.
const toDate = (day: string) => {
  const [y, m, d] = day.split('-').map(Number);
  return new Date(y, m - 1, d);
};
const toDay = (date: Date) =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;

const display = new Intl.DateTimeFormat('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });

const DatePicker = ({
  id,
  name,
  value,
  onChange,
  min,
  max,
}: {
  id?: string;
  name: string;
  value: string;
  onChange: (day: string) => void;
  min?: string;
  max?: string;
}) => {
  const [open, setOpen] = useState(false);
  const selected = value ? toDate(value) : undefined;
  const minDate = min ? toDate(min) : undefined;
  const maxDate = max ? toDate(max) : undefined;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <input type="hidden" name={name} value={value} />
      <PopoverTrigger asChild>
        <button
          id={id}
          type="button"
          className="flex w-full items-center gap-2 rounded-sm border border-transparent bg-surface-container px-3 py-2.5 text-left font-mono text-label outline-none transition-colors hover:border-border focus-visible:border-primary data-[state=open]:border-primary"
        >
          <CalendarDays className="size-4 shrink-0 text-muted-foreground" />
          <span className={`flex-1 ${selected ? '' : 'text-muted-foreground'}`}>
            {selected ? display.format(selected) : 'Pick a date'}
          </span>
          <ChevronDown className="size-4 text-muted-foreground" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto border-border bg-surface-lowest p-0">
        <Calendar
          mode="single"
          selected={selected}
          defaultMonth={selected}
          captionLayout="dropdown"
          startMonth={minDate}
          endMonth={maxDate}
          disabled={[...(minDate ? [{ before: minDate }] : []), ...(maxDate ? [{ after: maxDate }] : [])]}
          onSelect={(date) => {
            if (!date) return;
            onChange(toDay(date));
            setOpen(false);
          }}
          className="bg-transparent"
        />
        {maxDate && (
          <div className="flex justify-end border-t border-border px-3 py-2">
            <button
              type="button"
              onClick={() => {
                onChange(toDay(maxDate));
                setOpen(false);
              }}
              className="font-mono text-micro font-semibold tracking-[0.08em] text-primary uppercase hover:underline"
            >
              Latest allowed day
            </button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
};

export default DatePicker;
