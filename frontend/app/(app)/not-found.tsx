import { MapPinOff } from 'lucide-react';
import Link from 'next/link';

import ErrorState from '@/components/shared/error-state';
import { Button } from '@/components/ui/button';

// notFound() inside the app, e.g. a parcel that doesn't exist or isn't yours.
// The API answers 404 for both, so the message covers both.
export default function AppNotFound() {
  return (
    <ErrorState
      icon={MapPinOff}
      code="Error 404"
      title="Record not found"
      message="It doesn't exist, or it belongs to another account. Check the link, or find it in your parcel list."
    >
      <Button asChild size="lg">
        <Link href="/parcels">View parcels</Link>
      </Button>
      <Button asChild size="lg" variant="outline">
        <Link href="/">Go to overview</Link>
      </Button>
    </ErrorState>
  );
}
