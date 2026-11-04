import { Box, MapPinOff } from 'lucide-react';
import Link from 'next/link';

import ErrorState from '@/components/shared/error-state';
import { Button } from '@/components/ui/button';

// URLs that match no route. Renders inside the root layout only, so it brings its own frame.
export default function NotFound() {
  return (
    <main className="flex min-h-svh flex-col bg-background">
      <header className="flex items-center gap-3 px-6 py-5">
        <span className="flex size-8 items-center justify-center rounded-md bg-primary text-sidebar-primary">
          <Box className="size-4.5" strokeWidth={2.25} />
        </span>
        <span className="text-[17px] font-semibold tracking-tight">GROUNDWORK</span>
      </header>
      <ErrorState
        icon={MapPinOff}
        code="Error 404"
        title="This page isn't on the map"
        message="The address may be mistyped, or the page may have moved. Nothing has been changed."
      >
        <Button asChild size="lg">
          <Link href="/">Go to overview</Link>
        </Button>
      </ErrorState>
    </main>
  );
}
