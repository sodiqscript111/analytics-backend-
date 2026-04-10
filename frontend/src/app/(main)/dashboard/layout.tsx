import type { ReactNode } from "react";

import { SearchDialog } from "./_components/sidebar/search-dialog";

export default function Layout({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b bg-card/70 backdrop-blur">
        <div className="mx-auto flex w-full max-w-7xl items-center justify-between gap-3 px-4 py-3 lg:px-6">
          <div className="flex flex-col">
            <span className="text-lg font-semibold">Analytics Dashboard</span>
            <span className="text-xs text-muted-foreground">
              Search, monitor, and inspect live events.
            </span>
          </div>
          <SearchDialog />
        </div>
      </header>

      <main className="mx-auto w-full max-w-7xl p-4 md:p-6">{children}</main>
    </div>
  );
}
