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

export type RainPoint = {
  id: string;
  dateStart: string;
  dateEnd: string;
  pctOfNormal: number;
  totalMm: number;
  normalMm: number;
};

const PAD = { top: 26, right: 20, bottom: 28, left: 44 };
// Keeps a bar at either end of the time axis from being cut in half.
const EDGE = 14;
const NDVI_H = 170;
const RAIN_H = 120;
const GAP = 44;
const BAR_W = 16;
const LINE = '#1b4332';
const WETTER = '#2a78d6';
const DRIER = '#e34948';
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

// A bar from the 100%-of-normal line to the value: rounded at the data end, square at the line.
function barPath(cx: number, yBase: number, yValue: number) {
  const x0 = cx - BAR_W / 2;
  const x1 = cx + BAR_W / 2;
  const r = Math.min(4, Math.abs(yValue - yBase));
  const s = yValue < yBase ? 1 : -1;
  return `M${x0},${yBase} V${yValue + s * r} Q${x0},${yValue} ${x0 + r},${yValue} H${x1 - r} Q${x1},${yValue} ${x1},${yValue + s * r} V${yBase} Z`;
}

const Chip = ({ color, label }: { color: string; label: string }) => (
  <span className="flex items-center gap-1.5">
    <span className="size-2.5 rounded-xs" style={{ background: color }} />
    {label}
  </span>
);

// Both series arrive oldest first and share one time axis, each point at the end of its window,
// so a change in vegetation can be read against the rain that fell in the same window.
const ParcelTrend = ({ ndvi, rain }: { ndvi: NdviPoint[]; rain: RainPoint[] }) => {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  const [hover, setHover] = useState<number | null>(null);

  const hasPoints = ndvi.length + rain.length > 0;

  // Re-runs when the first result arrives: the chart container doesn't exist before then.
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasPoints]);

  if (!hasPoints) {
    return (
      <div className="flex h-70 flex-col items-center justify-center gap-1 text-center">
        <span className="text-h3 font-semibold">No results yet</span>
        <span className="max-w-xs text-label text-muted-foreground">
          The trend appears once a snapshot completes. Run one to start the record.
        </span>
      </div>
    );
  }

  const latestNdvi = ndvi.at(-1);
  const latestRain = rain.at(-1);

  const time = (day: string) => Date.parse(day);
  const times = [...new Set([...ndvi, ...rain].map((p) => time(p.dateEnd)))].sort((a, b) => a - b);
  const t0 = times[0];
  const t1 = times.at(-1)!;

  const innerW = Math.max(width - PAD.left - PAD.right - 2 * EDGE, 0);
  const x = (t: number) => PAD.left + EDGE + (t1 === t0 ? innerW / 2 : ((t - t0) / (t1 - t0)) * innerW);

  // NDVI panel: -1..1, the axis starts at 0 unless a value is negative.
  const ndviTop = PAD.top;
  const ndviBottom = ndviTop + (ndvi.length ? NDVI_H : 0);
  const ndviMin = ndvi.length ? Math.min(0, Math.floor(Math.min(...ndvi.map((p) => p.ndvi)) * 5) / 5) : 0;
  const ndviTicks = Array.from(
    { length: Math.round((1 - ndviMin) / 0.2) + 1 },
    (_, i) => +(ndviMin + i * 0.2).toFixed(1),
  );
  const yNdvi = (v: number) => ndviTop + ((1 - v) / (1 - ndviMin)) * NDVI_H;

  // Rain panel: % of normal from 0, bars from the 100% line.
  const rainTop = ndvi.length ? ndviBottom + GAP : PAD.top;
  const rainBottom = rainTop + (rain.length ? RAIN_H : 0);
  const rainMax = Math.max(150, Math.ceil(Math.max(0, ...rain.map((p) => p.pctOfNormal)) / 50) * 50);
  const rainStep = rainMax > 300 ? 100 : 50;
  const rainTicks = Array.from({ length: rainMax / rainStep + 1 }, (_, i) => i * rainStep);
  const yRain = (v: number) => rainTop + (1 - v / rainMax) * RAIN_H;

  const height = (rain.length ? rainBottom : ndviBottom) + PAD.bottom;
  const plotTop = PAD.top;
  const plotBottom = height - PAD.bottom;
  const xLabels = times.length > 1 ? [t0, t1] : [t0];

  const ndviPath = ndvi.map((p, i) => `${i ? 'L' : 'M'}${x(time(p.dateEnd))},${yNdvi(p.ndvi)}`).join(' ');

  const onMove = (e: React.PointerEvent<SVGRectElement>) => {
    const mx = e.clientX - e.currentTarget.getBoundingClientRect().left + PAD.left;
    let nearest = times[0];
    for (const t of times) if (Math.abs(x(t) - mx) < Math.abs(x(nearest) - mx)) nearest = t;
    setHover(nearest);
  };

  const hoveredNdvi = hover === null ? undefined : ndvi.find((p) => time(p.dateEnd) === hover);
  const hoveredRain = hover === null ? undefined : rain.find((p) => time(p.dateEnd) === hover);

  const gridLine = (y: number, label: string, key: string, strong = false) => (
    <g key={key}>
      <line x1={PAD.left} x2={width - PAD.right} y1={y} y2={y} stroke={strong ? 'var(--outline)' : 'var(--border)'} />
      <text x={PAD.left - 8} y={y} dy="0.32em" textAnchor="end" className="fill-muted-foreground font-mono text-[10px]">
        {label}
      </text>
    </g>
  );

  const panelLabel = (y: number, text: string) => (
    <text x={PAD.left} y={y - 12} className="fill-muted-foreground font-mono text-[10px] tracking-[0.12em] uppercase">
      {text}
    </text>
  );

  const summary = [
    latestNdvi && `latest mean NDVI ${latestNdvi.ndvi.toFixed(3)} for the period ending ${latestNdvi.dateEnd}`,
    latestRain &&
      `latest rainfall ${latestRain.pctOfNormal.toFixed(0)}% of normal for the period ending ${latestRain.dateEnd}`,
  ]
    .filter(Boolean)
    .join('; ');

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end gap-x-6 gap-y-2">
        {latestNdvi && (
          <div>
            <span className="block font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase">
              Latest mean NDVI
            </span>
            <span className="font-mono text-[40px] leading-none font-semibold tracking-tight">
              {latestNdvi.ndvi.toFixed(3)}
            </span>
          </div>
        )}
        {latestNdvi && <Delta current={latestNdvi.ndvi} previous={ndvi.at(-2)} />}
        {latestRain && (
          <div>
            <span className="block font-mono text-micro tracking-[0.12em] text-muted-foreground uppercase">
              Latest rainfall
            </span>
            <span className="font-mono text-h3 font-semibold">{latestRain.pctOfNormal.toFixed(0)}% of normal</span>
            <span className="ml-2 text-label text-muted-foreground">to {latestRain.dateEnd}</span>
          </div>
        )}
      </div>

      <div ref={ref} className="relative">
        {width > 0 && (
          <svg width={width} height={height} role="img" aria-label={`Parcel trend: ${summary}.`}>
            {ndvi.length > 0 && (
              <>
                {panelLabel(ndviTop, 'Mean NDVI')}
                {ndviTicks.map((tick) => gridLine(yNdvi(tick), tick.toFixed(1), `n${tick}`))}
              </>
            )}
            {rain.length > 0 && (
              <>
                {panelLabel(rainTop, 'Rainfall, % of normal')}
                {rainTicks.map((tick) => gridLine(yRain(tick), `${tick}%`, `r${tick}`, tick === 100))}
              </>
            )}
            {xLabels.map((t, i) => (
              <text
                key={t}
                x={x(t)}
                y={height - 8}
                textAnchor={xLabels.length > 1 ? (i === 0 ? 'start' : 'end') : 'middle'}
                className="fill-muted-foreground font-mono text-[10px]"
              >
                {monthYear.format(t)}
              </text>
            ))}

            {hover !== null && (
              <line
                x1={x(hover)}
                x2={x(hover)}
                y1={plotTop}
                y2={plotBottom}
                stroke="var(--outline)"
                strokeDasharray="3 3"
              />
            )}

            {rain.map((p) => (
              <path
                key={p.id}
                d={barPath(x(time(p.dateEnd)), yRain(100), yRain(p.pctOfNormal))}
                fill={p.pctOfNormal >= 100 ? WETTER : DRIER}
              />
            ))}

            {ndvi.length > 0 && (
              <path
                d={ndviPath}
                fill="none"
                stroke={LINE}
                strokeWidth={2}
                strokeLinejoin="round"
                strokeLinecap="round"
              />
            )}
            {ndvi.map((p) => (
              <circle
                key={p.id}
                cx={x(time(p.dateEnd))}
                cy={yNdvi(p.ndvi)}
                r={hover === time(p.dateEnd) ? 5.5 : 4}
                fill={LINE}
                stroke={SURFACE}
                strokeWidth={2}
              />
            ))}

            <rect
              x={PAD.left}
              y={plotTop}
              width={Math.max(width - PAD.left - PAD.right, 0)}
              height={plotBottom - plotTop}
              fill="transparent"
              onPointerMove={onMove}
              onPointerLeave={() => setHover(null)}
            />
          </svg>
        )}

        {hover !== null && (hoveredNdvi || hoveredRain) && (
          <div
            className="pointer-events-none absolute z-10 -translate-x-1/2 rounded-sm bg-primary px-2.5 py-1.5 font-mono text-micro whitespace-nowrap text-primary-foreground shadow-md"
            style={{ left: Math.min(Math.max(x(hover), 110), width - 110), top: 0 }}
          >
            {hoveredNdvi && (
              <span className="block">
                <span className="text-primary-foreground/70">
                  {hoveredNdvi.dateStart} → {hoveredNdvi.dateEnd}
                </span>
                <br />
                NDVI {hoveredNdvi.ndvi.toFixed(3)}
                {hoveredNdvi.validPixelPct !== undefined && ` · ${hoveredNdvi.validPixelPct.toFixed(0)}% valid pixels`}
              </span>
            )}
            {hoveredRain && (
              <span className={`block ${hoveredNdvi ? 'mt-1' : ''}`}>
                <span className="text-primary-foreground/70">
                  {hoveredRain.dateStart} → {hoveredRain.dateEnd}
                </span>
                <br />
                Rain {Math.round(hoveredRain.totalMm)} mm · {hoveredRain.pctOfNormal.toFixed(0)}% of normal (
                {Math.round(hoveredRain.normalMm)} mm)
              </span>
            )}
          </div>
        )}
      </div>

      <div className="flex flex-wrap gap-x-4 gap-y-1 font-mono text-micro text-muted-foreground">
        {ndvi.length > 0 && <Chip color={LINE} label="Mean NDVI" />}
        {rain.length > 0 && <Chip color={WETTER} label="Rain above normal" />}
        {rain.length > 0 && <Chip color={DRIER} label="Rain below normal" />}
        {rain.length === 0 && <span>Run a rainfall snapshot to see rain on the same time axis.</span>}
      </div>
    </div>
  );
};

export default ParcelTrend;
