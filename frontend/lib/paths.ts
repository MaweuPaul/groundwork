// Clean URLs: /parcels/nyeri-block-a-k7x2p9qd. The short ID at the end is the key;
// the name part is only for people and is corrected by a redirect if it's stale.

const SHORT_ID = /^[0-9a-hjkmnp-tv-z]{8}$/;

export function slugify(text: string) {
  const slug = text
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60)
    .replace(/-+$/, '');
  return slug || 'parcel';
}

export function parcelPath(parcel: { name: string; short_id: string }) {
  return `/parcels/${slugify(parcel.name)}-${parcel.short_id}`;
}

export function snapshotPath(snapshot: { short_id: string }) {
  return `/snapshots/${snapshot.short_id}`;
}

// The ref to send to the API: the trailing short ID, or the whole segment for an
// old UUID link (which then redirects to the clean URL). null when it's neither.
export function refFromSegment(segment: string) {
  const decoded = decodeURIComponent(segment);
  if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(decoded)) return decoded;
  const shortId = decoded.slice(decoded.lastIndexOf('-') + 1);
  return SHORT_ID.test(shortId) ? shortId : null;
}

// "nyeri-block-a-k7x2p9qd" -> "nyeri block a", for the breadcrumb.
export function labelFromSegment(segment: string) {
  const dash = segment.lastIndexOf('-');
  return dash > 0 && SHORT_ID.test(segment.slice(dash + 1))
    ? segment.slice(0, dash).replaceAll('-', ' ')
    : null;
}
