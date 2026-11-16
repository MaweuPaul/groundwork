'use client';

import { useSyncExternalStore } from 'react';

import { formatUtc } from '@/lib/format';

type Style = 'datetime' | 'date' | 'day-time';

type Parts = { date: string; time: string; seconds: string; zone: string };

function parts(at: Date, timeZone?: string): Parts {
  const get = (options: Intl.DateTimeFormatOptions) =>
    Object.fromEntries(
      new Intl.DateTimeFormat('en-US', { ...options, timeZone, hourCycle: 'h23' })
        .formatToParts(at)
        .map((p) => [p.type, p.value]),
    );
  const p = get({
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    timeZoneName: 'short',
  });
  return {
    date: `${p.year}-${p.month}-${p.day}`,
    time: `${p.hour}:${p.minute}`,
    seconds: p.second,
    zone: timeZone === 'UTC' ? 'UTC' : p.timeZoneName,
  };
}

function render(iso: string, style: Style, timeZone?: string) {
  const at = new Date(iso);
  const p = parts(at, timeZone);
  if (style === 'date') return p.date;
  if (style === 'datetime') return `${p.date} ${p.time}:${p.seconds} ${p.zone}`;

  const yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  const label =
    p.date === parts(new Date(), timeZone).date
      ? 'Today'
      : p.date === parts(yesterday, timeZone).date
        ? 'Yesterday'
        : p.date;
  return `${label} ${p.time} ${p.zone}`;
}

const subscribe = () => () => {};

// Shows an instant in the viewer's own time zone. The server doesn't know it, so the first
// render (and the server's HTML) is UTC; the browser switches to local time on hydration.
// The UTC value stays in the tooltip, since sealed records are kept in UTC.
const LocalTime = ({ iso, style = 'datetime' }: { iso: string; style?: Style }) => {
  const text = useSyncExternalStore(
    subscribe,
    () => render(iso, style),
    () => render(iso, style, 'UTC'),
  );
  return (
    <time dateTime={iso} title={formatUtc(iso)}>
      {text}
    </time>
  );
};

export default LocalTime;
