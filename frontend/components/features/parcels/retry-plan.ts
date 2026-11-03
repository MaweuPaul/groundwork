import type { SnapshotOptions } from '@/components/features/parcels/run-snapshot-dialog';
import type { SnapshotSummary } from '@/lib/types';

type RetrySource = Pick<
  SnapshotSummary,
  'date_start' | 'date_end' | 'max_scene_cloud_pct' | 'cloud_score_threshold' | 'failure_message'
>;

// failure_message is "CODE: reason".
export const failureCode = (message?: string) => message?.split(': ')[0];

const MIN_WINDOW_DAYS = 30;
const EARLIEST = '2015-06-23';

const utc = (day: string) => new Date(`${day}T00:00:00Z`);

function windowDays(dateStart: string, dateEnd: string) {
  return (utc(dateEnd).getTime() - utc(dateStart).getTime()) / 86_400_000 + 1;
}

function daysBefore(day: string, days: number) {
  const d = utc(day);
  d.setUTCDate(d.getUTCDate() - days);
  const result = d.toISOString().slice(0, 10);
  return result < EARLIEST ? EARLIEST : result;
}

// Starts a short window 30 days before its end, so more satellite passes are included.
function widened(dateStart: string, dateEnd: string) {
  if (windowDays(dateStart, dateEnd) >= MIN_WINDOW_DAYS) return dateStart;
  return daysBefore(dateEnd, MIN_WINDOW_DAYS - 1);
}

// What to change for the retry, based on why the snapshot failed. Data failures get looser
// settings; anything else retries with the same ones.
export function retryPlan(s: RetrySource): { initial: SnapshotOptions; note: string } {
  const same: SnapshotOptions = {
    dateStart: s.date_start,
    dateEnd: s.date_end,
    maxSceneCloudPct: s.max_scene_cloud_pct,
    cloudScoreThreshold: s.cloud_score_threshold,
  };
  const code = failureCode(s.failure_message);
  if (code === 'COMPUTATION_TOO_LARGE') {
    const days = windowDays(s.date_start, s.date_end);
    const dateStart = daysBefore(s.date_end, MIN_WINDOW_DAYS - 1);
    return {
      initial: { ...same, dateStart: dateStart > s.date_start ? dateStart : s.date_start },
      note: `The ${days}-day window was too much imagery for one Earth Engine request. Suggested: the last ${MIN_WINDOW_DAYS} days of it. Run more short snapshots to cover the rest of the period.`,
    };
  }
  if (code !== 'NO_IMAGERY' && code !== 'NO_CLEAR_PIXELS') {
    return { initial: same, note: 'Prefilled with the failed attempt’s settings. Adjust any of them before starting.' };
  }

  const cloudPct = Math.max(60, Math.min(100, s.max_scene_cloud_pct * 2));
  const dateStart = widened(s.date_start, s.date_end);
  const changes = [`scenes up to ${cloudPct}% cloud (was ${s.max_scene_cloud_pct}%)`];
  if (dateStart !== s.date_start) changes.push(`a ${MIN_WINDOW_DAYS}-day window`);

  if (code === 'NO_CLEAR_PIXELS') {
    changes.push('the lenient clear-pixel mask');
    return {
      initial: { ...same, dateStart, maxSceneCloudPct: cloudPct, cloudScoreThreshold: 0.5 },
      note: `Every scene was cloudy over the parcel. Suggested: ${changes.join(', ')}.`,
    };
  }
  return {
    initial: { ...same, dateStart, maxSceneCloudPct: cloudPct },
    note: `No scene passed the ${s.max_scene_cloud_pct}% cloud limit. Suggested: ${changes.join(', ')}. Clouds over the parcel are still masked pixel by pixel.`,
  };
}
