'use client';

import './globals.css';

import { TriangleAlert } from 'lucide-react';

import ErrorState from '@/components/shared/error-state';
import { Button } from '@/components/ui/button';

// Last resort when the root layout itself fails. It replaces the root layout,
// so it brings its own <html>, <body> and styles.
export default function GlobalError({
  error,
  unstable_retry,
}: {
  error: Error & { digest?: string };
  unstable_retry: () => void;
}) {
  return (
    <html lang="en">
      <body className="flex min-h-svh flex-col bg-background font-sans text-foreground antialiased">
        <title>Something went wrong · Groundwork</title>
        <ErrorState
          icon={TriangleAlert}
          code="Something went wrong"
          title="Groundwork hit an unexpected error"
          message="Try again in a moment. If it keeps happening, the reference below helps trace it."
          detail={error.digest ? `Reference: ${error.digest}` : undefined}
        >
          <Button size="lg" onClick={() => unstable_retry()}>
            Try again
          </Button>
        </ErrorState>
      </body>
    </html>
  );
}
