import { AppSidebar } from '@/components/app-sidebar';
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar';
import { getMe } from '@/lib/api';

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  // A missing or rejected session already redirected to /login inside getMe.
  const me = await getMe();
  if (!me.ok) throw new Error(me.error);

  return (
    <SidebarProvider>
      <AppSidebar role={me.data.role} />
      <SidebarInset>
        <header className="flex h-14 items-center border-b border-border px-4">
          <SidebarTrigger />
        </header>
        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
