import { AppSidebar } from '@/components/shared/app-sidebar';
import Header from '@/components/shared/header';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { getMe } from '@/lib/api';

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  // A missing or rejected session already redirected to /login inside getMe.
  const me = await getMe();
  if (!me.ok) throw new Error(me.error);

  return (
    <SidebarProvider>
      <AppSidebar user={me.data} />
      {/* min-w-0: lets wide content (tables) scroll inside the page instead of widening it. */}
      <SidebarInset className="min-w-0">
        <Header />
        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
