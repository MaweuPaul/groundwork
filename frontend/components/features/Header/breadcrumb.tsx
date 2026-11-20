'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Fragment } from 'react';

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { labelFromSegment } from '@/lib/paths';

const labels: Record<string, string> = {
  parcels: 'Parcels',
  snapshots: 'Snapshots',
  methodologies: 'Methodologies',
  new: 'Register',
};

// Path segments with no page of their own; shown as text, not links.
const noPage = new Set(['snapshots']);

const SHORT_ID = /^[0-9a-hjkmnp-tv-z]{8}$/;

function label(segment: string) {
  const decoded = decodeURIComponent(segment);
  return labels[segment] ?? labelFromSegment(decoded) ?? (SHORT_ID.test(decoded) ? decoded : 'Details');
}

export function BreadcrumbHeader() {
  const pathname = usePathname();
  const segments = pathname.split('/').filter(Boolean);

  const crumbs = [
    { href: '/', label: 'Overview', linked: true },
    ...segments.map((segment, i) => ({
      href: '/' + segments.slice(0, i + 1).join('/'),
      label: label(segment),
      linked: !noPage.has(segment),
    })),
  ];

  return (
    <Breadcrumb className="min-w-0">
      <BreadcrumbList className="flex-nowrap font-mono text-[13px] tracking-wider uppercase">
        {crumbs.map((crumb, i) => (
          <Fragment key={crumb.href}>
            {i > 0 && <BreadcrumbSeparator className="hidden md:block" />}
            <BreadcrumbItem className={i === crumbs.length - 1 ? 'min-w-0' : 'hidden md:inline-flex'}>
              {i === crumbs.length - 1 ? (
                <BreadcrumbPage className="truncate font-bold">
                  {crumb.label}
                </BreadcrumbPage>
              ) : !crumb.linked ? (
                <span>{crumb.label}</span>
              ) : (
                <BreadcrumbLink asChild>
                  <Link href={crumb.href}>{crumb.label}</Link>
                </BreadcrumbLink>
              )}
            </BreadcrumbItem>
          </Fragment>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
