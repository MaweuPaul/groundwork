'use client';

import { ArrowDownRight, ArrowRight, ArrowUpRight } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';

export type NdviPoint = {
  id: string;
  dateStart: string;
  dateEnd: string;
  ndvi: number;
  validPixelPct?: number;
};

const HEIGHT = 220;
const PAD = { top: 12, right: 20, bottom: 28, left: 40 };
const LINE = '#1b4332';
const SURFACE = '#ffffff';

const monthYear = new Intl.DateTimeFormat('en', { month: 'short', year: '2-digit', timeZone: 'UTC' });

function Delta({ current, previous }: { current: number; previous?: NdviPoint }) {
  if (!previous) return null;
  const change = current - previous.ndvi;
  const Icon = change > 0.0005 ? ArrowUpRight : change < -0.0005 ? ArrowDownRight : ArrowRight;
  return (
    <span className="flex items-center gap-1 text-label text-muted-foreground">
      <Icon className="size-4" />
      {change >= 0 ? '+' : ''}
      {change.toFixed(3)} vs previous ({previous.dateEnd})
    </span>
  );
}

// Points arrive oldest first. NDVI ranges -1..1; the axis starts at 0 unless a value is negative.
const NdviTrend = ({ points }: { points: NdviPoint[] }) => {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  const [hover, setHover] = useState<number | null>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const latest = points.at(-1);

  if (!latest) {
    return (
      <div className="flex h-70 flex-col items-center justify-center gap-1 text-center">
        <span className="text-h3 font-semibold">No results yet</span>
        <span className="max-w-xs text-label text-muted-foreground">
          The trend appears once a snapshot completes. Run one to start the record.
        </span>
      </div>
    );
  }

  const times = points.map((p) => Date.parse(p.dateEnd));
  const t0 = Math.min(...times);
  const t1 = Math.max(...times);
  const yMin = Math.min(0, Math.floor(Math.min(...points.map((p) => p.ndvi)) * 5) / 5);
  const ticks = Array.from({ length: Math.round((1 - yMin) / 0.2) + 1 }, (_, i) => +(yMin + i * 0.2).toFixed(1));

  const innerW = Math.max(width - PAD.left - PAD.right, 0);
  const innerH = HEIGHT - PAD.top - PAD.bottom;
  const x = (t: number) => PAD.left + (t1 === t0 ? innerW / 2 : ((t - t0) / (t1 - t0)) * innerW);
  const y = (v: number) => PAD.top + ((1 - v) / (1 - yMin)) * innerH;

  const path = points.map((p, i) => `${i ? 'L' : 'M'}${x(times[i])},${y(p.ndvi)}`).join(' ');
  const xLabels = points.length > 1 ? [0, points.length - 1] : [0];

  const onMove = (e: React.PointerEvent<SVGRectElement>) => {
    const mx = e.clientX - e.currentTarget.getBoundingClientRect().left + PAD.left;
    let nearest = 0;
    times.forEach((t, i) => {
      if (Math.abs(x(t) - mx) < Math.abs(x(times[nearest]) - mx)) nearest = i;
    });
    setHover(nearest);
  };

  const hovered = hover === null ? null : points[hover];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end gap-x-4 gap-y-1">
        <div>
          <span className="block font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase">
            Latest mean NDVI
          </span>
          <span className="font-mono text-[40px] leading-none font-semibold tracking-tight">
            {latest.ndvi.toFixed(3)}
          </span>
        </div>
        <Delta current={latest.ndvi} previous={points.at(-2)} />
      </div>

      <div ref={ref} className="relative">
        {width > 0 && (
          <svg
            width={width}
            height={HEIGHT}
            role="img"
            aria-label={`Mean NDVI for ${points.length} completed snapshots, latest ${latest.ndvi.toFixed(3)} for the period ending ${latest.dateEnd}.`}
          >
            {ticks.map((tick) => (
              <g key={tick}>
                <line x1={PAD.left} x2={width - PAD.right} y1={y(tick)} y2={y(tick)} stroke="var(--border)" />
                <text x={PAD.left - 8} y={y(tick)} dy="0.32em" textAnchor="end" className="fill-muted-foreground font-mono text-[10px]">
                  {tick.toFixed(1)}
                </text>
              </g>
            ))}
            {xLabels.map((i) => (
              <text
                key={i}
                x={x(times[i])}
                y={HEIGHT - 8}
                textAnchor={points.length > 1 ? (i === 0 ? 'start' : 'end') : 'middle'}
                className="fill-muted-foreground font-mono text-[10px]"
              >
                {monthYear.format(times[i])}
              </text>
            ))}

            {hovered && (
              <line
                x1={x(times[hover!])}
                x2={x(times[hover!])}
                y1={PAD.top}
                y2={HEIGHT - PAD.bottom}
                stroke="var(--outline)"
                strokeDasharray="3 3"
              />
            )}
            <path d={path} fill="none" stroke={LINE} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
            {points.map((p, i) => (
              <circle
                key={p.id}
                cx={x(times[i])}
                cy={y(p.ndvi)}
                r={hover === i ? 5.5 : 4}
                fill={LINE}
                stroke={SURFACE}
                strokeWidth={2}
              />
            ))}

            <rect
              x={PAD.left}
              y={PAD.top}
              width={innerW}
              height={innerH}
              fill="transparent"
              onPointerMove={onMove}
              onPointerLeave={() => setHover(null)}
            />
          </svg>
        )}

        {hovered && (
          <div
            className="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full rounded-sm bg-primary px-2.5 py-1.5 font-mono text-micro whitespace-nowrap text-primary-foreground shadow-md"
            style={{
              left: Math.min(Math.max(x(times[hover!]), 90), width - 90),
              top: y(hovered.ndvi) - 10,
            }}
          >
            <span className="block text-primary-foreground/70">
              {hovered.dateStart} → {hovered.dateEnd}
            </span>
            NDVI {hovered.ndvi.toFixed(3)}
            {hovered.validPixelPct !== undefined && ` · ${hovered.validPixelPct.toFixed(0)}% valid pixels`}
          </div>
        )}
      </div>
    </div>
  );
};

export default NdviTrend;
