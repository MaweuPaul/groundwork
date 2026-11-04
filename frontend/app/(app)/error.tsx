'use client';

import { RotateCw, TriangleAlert } from 'lucide-react';
import Link from 'next/link';
import { useEffect } from 'react';

import ErrorState from '@/components/shared/error-state';
import { Button } from '@/components/ui/button';

// Errors thrown while rendering an app page, e.g. the API being unreachable.
// The sidebar and header stay, so the user can navigate away.
export default function AppError({
  error,
  unstable_retry,
}: {
  error: Error & { digest?: string };
  unstable_retry: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <ErrorState
      icon={TriangleAlert}
      code="Something went wrong"
      title="We couldn't load this page"
      message="This is usually temporary, for example the server being briefly unreachable. Your data is safe; try again in a moment."
      detail={error.digest ? `Reference: ${error.digest}` : undefined}
    >
      <Button size="lg" onClick={() => unstable_retry()}>
        <RotateCw />
        Try again
      </Button>
      <Button asChild size="lg" variant="outline">
        <Link href="/">Go to overview</Link>
      </Button>
    </ErrorState>
  );
}
