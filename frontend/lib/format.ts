export function formatArea(ha: number) {
  return ha.toLocaleString('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

export function formatNdvi(value: number | undefined) {
  return value === undefined ? '—' : value.toFixed(3);
}

export function formatCoordinates(lat: number, lon: number) {
  return `${lat.toFixed(4)}, ${lon.toFixed(4)}`;
}

export function formatDate(iso: string) {
  return new Date(iso).toISOString().slice(0, 10);
}

const units: [Intl.RelativeTimeFormatUnit, number][] = [
  ['day', 86_400_000],
  ['hour', 3_600_000],
  ['minute', 60_000],
];

export function timeAgo(iso: string, now = Date.now()) {
  const elapsed = now - new Date(iso).getTime();
  const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
  for (const [unit, ms] of units) {
    if (elapsed >= ms) return rtf.format(-Math.floor(elapsed / ms), unit);
  }
  return 'just now';
}
