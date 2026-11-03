import { Search as SearchIcon } from 'lucide-react';

import Border from '@/components/shared/border';

const Search = () => {
  // here write teh function to inculde the search prefererrable a hook
  return (
    <Border className="flex h-9 w-80 items-center gap-2 px-3 focus-within:border-ring">
      <SearchIcon className="size-4 shrink-0 text-muted-foreground" />
      <input
        type="text"
        autoComplete="off"
        placeholder="Filter parcel, ledger hash, or coordinates…"
        className="min-w-0 flex-1 bg-transparent text-[13px] outline-none placeholder:text-muted-foreground"
      />
      <kbd className="shrink-0 rounded-sm bg-surface-container px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">
        ⌘K
      </kbd>
    </Border>
  );
};

export default Search;
