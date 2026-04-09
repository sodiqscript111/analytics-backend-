import type { ReactNode } from "react";

import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";

export default function Layout({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <SidebarProvider defaultOpen={true}>
      <SidebarInset>
        <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4 lg:px-6">
          <div className="flex items-center gap-2">
            <span className="text-lg font-semibold">Analytics Dashboard</span>
          </div>
        </header>
        <div className="h-full p-4 md:p-6">{children}</div>
      </SidebarInset>
    </SidebarProvider>
  );
}
