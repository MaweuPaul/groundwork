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

const labels: Record<string, string> = {
  parcels: 'Parcels',
  snapshots: 'Snapshots',
  new: 'Register',
};

function label(segment: string) {
  return labels[segment] ?? 'Details';
}

export function BreadcrumbHeader() {
  const pathname = usePathname();
  const segments = pathname.split('/').filter(Boolean);

  const crumbs = [
    { href: '/', label: 'Overview' },
    ...segments.map((segment, i) => ({
      href: '/' + segments.slice(0, i + 1).join('/'),
      label: label(segment),
    })),
  ];

  return (
    <Breadcrumb>
      <BreadcrumbList className="font-mono text-[13px] tracking-wider uppercase">
        {crumbs.map((crumb, i) => (
          <Fragment key={crumb.href}>
            {i > 0 && <BreadcrumbSeparator />}
            <BreadcrumbItem>
              {i === crumbs.length - 1 ? (
                <BreadcrumbPage className="font-bold">
                  {crumb.label}
                </BreadcrumbPage>
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
