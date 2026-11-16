'use client';

import { useRouter } from 'next/navigation';
import type { MouseEvent, ReactNode } from 'react';

// Clicks inside a portal (e.g. a dialog opened from the row) still bubble here through React,
// so only clicks on the row's own DOM, outside links and buttons, navigate.
const ClickableRow = ({ href, className, children }: { href: string; className?: string; children: ReactNode }) => {
  const router = useRouter();

  const onClick = (e: MouseEvent<HTMLTableRowElement>) => {
    const target = e.target as HTMLElement;
    if (!e.currentTarget.contains(target) || target.closest('a, button, input')) return;
    if (window.getSelection()?.toString()) return;
    if (e.metaKey || e.ctrlKey) {
      window.open(href, '_blank');
      return;
    }
    router.push(href);
  };

  return (
    <tr onClick={onClick} className={`cursor-pointer ${className ?? ''}`}>
      {children}
    </tr>
  );
};

export default ClickableRow;
