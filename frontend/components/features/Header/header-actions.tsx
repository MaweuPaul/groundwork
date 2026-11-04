import { Bell, SlidersHorizontal, User } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';

const HeaderActions = () => {
  return (
    <div className="flex items-center gap-2">
      <Separator orientation="vertical" className="mx-1 data-vertical:h-6 data-vertical:self-center" />
      <Button variant="outline" size="icon-lg" aria-label="Notifications" className="bg-surface-lowest">
        <Bell />
      </Button>
      <Button variant="outline" size="icon-lg" aria-label="Settings" className="bg-surface-lowest">
        <SlidersHorizontal />
      </Button>
      <Button size="icon-lg" aria-label="Account">
        <User />
      </Button>
    </div>
  );
};

export default HeaderActions;
