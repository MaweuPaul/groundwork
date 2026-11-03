'use client';

import { Box, Layers, LayoutGrid, LogOut, type LucideIcon } from 'lucide-react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { logout } from '@/app/(auth)/actions';
import type { Role, User } from '@/lib/types';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar';

type NavItem = {
  title: string;
  url: string;
  icon: LucideIcon;
  roles: Role[];
};

const navItems: NavItem[] = [
  {
    title: 'Overview',
    url: '/',
    icon: LayoutGrid,
    roles: ['asset_holder', 'auditor'],
  },
  { title: 'Parcel', url: '/parcel', icon: Layers, roles: ['asset_holder'] },
  // Hidden until their pages and backend exist:
  // { title: 'Claims Registry', url: '/claims-registry', icon: ClipboardList, roles: ['auditor'] },
  // { title: 'Spectral Telemetry', url: '/spectral-telemetry', icon: Satellite, roles: ['asset_holder', 'auditor'] },
  // { title: 'Audit Ledgers', url: '/audit-ledgers', icon: ScrollText, roles: ['auditor'] },
  // { title: 'Evidence Vault', url: '/evidence-vault', icon: FolderLock, roles: ['asset_holder', 'auditor'] },
];

const roleLabels: Record<Role, string> = {
  asset_holder: 'Asset holder',
  auditor: 'Auditor',
};

function isActive(pathname: string, url: string) {
  return url === '/'
    ? pathname === '/'
    : pathname === url || pathname.startsWith(`${url}/`);
}

export function AppSidebar({ user }: { user: User }) {
  const pathname = usePathname();
  const items = navItems.filter((item) => item.roles.includes(user.role));
  const initials =
    `${user.first_name[0] ?? ''}${user.last_name[0] ?? ''}`.toUpperCase();

  return (
    <Sidebar>
      <SidebarHeader className="border-b border-sidebar-border px-4 py-4">
        <Link href="/" className="flex items-center gap-3">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-sidebar-primary text-sidebar-primary-foreground">
            <Box className="size-4.5" strokeWidth={2.25} />
          </div>
          <div className="leading-none">
            <span className="block text-[17px] font-semibold tracking-tight text-white">
              GROUNDWORK
            </span>
            <span className="mt-1 block font-mono text-micro">
              <span className="font-semibold text-sidebar-primary">SYSTEM</span>{' '}
              <span className="text-sidebar-foreground/60">v1.0.0</span>
            </span>
          </div>
        </Link>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup className="px-2 pt-4">
          <SidebarGroupLabel className="mb-1 h-auto px-2 font-mono text-micro tracking-[0.14em] text-sidebar-foreground/55 uppercase">
            Parcel verification
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu className="gap-0.5">
              {items.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton
                    asChild
                    isActive={isActive(pathname, item.url)}
                    className="relative h-9 gap-3 rounded-sm px-3 text-[14px] text-sidebar-foreground/85 data-active:font-semibold data-active:text-white data-active:before:absolute data-active:before:inset-y-1 data-active:before:left-0 data-active:before:w-0.75 data-active:before:rounded-full data-active:before:bg-sidebar-primary"
                  >
                    <Link href={item.url}>
                      <item.icon />
                      <span>{item.title}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="gap-2 border-t border-sidebar-border p-3">
        <div className="flex items-center gap-3 px-1">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-sidebar-accent text-[12px] font-semibold text-sidebar-primary">
            {initials}
          </div>
          <div className="min-w-0 leading-tight">
            <span className="block truncate text-[13px] font-semibold text-white">
              {user.first_name} {user.last_name}
            </span>
            <span className="block truncate text-[12px] text-sidebar-foreground/60">
              {user.email}
            </span>
            <span className="mt-0.5 block font-mono text-micro tracking-[0.1em] text-sidebar-primary uppercase">
              {roleLabels[user.role]}
            </span>
          </div>
        </div>
        <form action={logout}>
          <SidebarMenuButton
            type="submit"
            className="h-9 gap-3 rounded-sm px-3 text-[14px] text-sidebar-foreground/85 hover:text-white"
          >
            <LogOut />
            <span>Sign out</span>
          </SidebarMenuButton>
        </form>
      </SidebarFooter>
    </Sidebar>
  );
}
