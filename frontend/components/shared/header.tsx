import { BreadcrumbHeader } from '../features/Header/breadcrumb';
import HeaderActions from '../features/Header/header-actions';
import Search from '../features/Header/search';
import { Separator } from '@/components/ui/separator';
import { SidebarTrigger } from '@/components/ui/sidebar';

const Header = () => {
  return (
    <header className="sticky top-0 z-10 flex h-14 shrink-0 items-center gap-3 border-b border-border bg-background px-4">
      <SidebarTrigger />
      <Separator orientation="vertical" className="data-vertical:h-5 data-vertical:self-center" />
      <BreadcrumbHeader />
      <div className="ml-auto flex items-center gap-2">
        <Search />
        <HeaderActions />
      </div>
    </header>
  );
};

export default Header;
