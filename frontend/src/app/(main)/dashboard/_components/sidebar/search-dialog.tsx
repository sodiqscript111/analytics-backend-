"use client";

import * as React from "react";
import { Activity, LayoutDashboard, Search } from "lucide-react";
import { usePathname, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "@/components/ui/command";
import {
  API_BASE,
  buildSearchParams,
  type EventItem,
  type SearchEventsResponse,
} from "@/lib/analytics-dashboard";

const staticItems = [
  {
    group: "Navigation",
    label: "Realtime Dashboard",
    href: "/dashboard/default",
    icon: LayoutDashboard,
  },
];

function formatTimeLabel(timestamp: string): string {
  const parsed = new Date(timestamp);
  if (Number.isNaN(parsed.getTime())) {
    return "recent";
  }

  return parsed.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
  });
}

async function searchEvents(query: string, signal?: AbortSignal): Promise<EventItem[]> {
  if (!query.trim()) return [];

  const params = buildSearchParams(
    {
      query,
      action: "",
      userId: "",
      from: "",
      to: "",
    },
    undefined,
    8,
  );

  const res = await fetch(`${API_BASE}/search/events?${params.toString()}`, {
    cache: "no-store",
    signal,
  });
  if (!res.ok) {
    return [];
  }

  const payload = (await res.json()) as SearchEventsResponse;
  return payload.items;
}

export function SearchDialog() {
  const pathname = usePathname();
  const router = useRouter();

  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [results, setResults] = React.useState<EventItem[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);

  React.useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === "j" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        setOpen((current) => !current);
      }
    };

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

  React.useEffect(() => {
    if (!open) return;
    if (!query.trim()) {
      setResults([]);
      setIsLoading(false);
      return;
    }

    const controller = new AbortController();
    const timeout = window.setTimeout(async () => {
      setIsLoading(true);
      const items = await searchEvents(query, controller.signal);
      setResults(items);
      setIsLoading(false);
    }, 300);

    return () => {
      controller.abort();
      window.clearTimeout(timeout);
    };
  }, [open, query]);

  const openDashboardSearch = React.useCallback(
    (searchQuery: string) => {
      const params = new URLSearchParams();
      if (searchQuery) {
        params.set("q", searchQuery);
      }

      const target = params.toString()
        ? `/dashboard/default?${params.toString()}`
        : "/dashboard/default";

      router.push(target);
      setOpen(false);
    },
    [router],
  );

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        className="gap-2"
        onClick={() => setOpen(true)}
      >
        <Search className="size-4" />
        Search
        <kbd className="rounded border bg-muted px-1.5 text-[10px] font-medium">
          Ctrl/Cmd+J
        </kbd>
      </Button>

      <CommandDialog open={open} onOpenChange={setOpen}>
        <CommandInput
          value={query}
          onValueChange={setQuery}
          placeholder="Search events, ids, users, or actions..."
        />
        <CommandList>
          <CommandEmpty>
            {isLoading ? "Searching events..." : "No matching dashboard items"}
          </CommandEmpty>

          <CommandGroup heading="Navigation">
            {staticItems.map((item) => (
              <CommandItem
                key={item.label}
                value={item.label}
                onSelect={() => {
                  router.push(item.href);
                  setOpen(false);
                }}
              >
                <item.icon />
                <span>{item.label}</span>
                {pathname === item.href ? (
                  <CommandShortcut>Open</CommandShortcut>
                ) : null}
              </CommandItem>
            ))}
          </CommandGroup>

          {query.trim() ? <CommandSeparator /> : null}

          {query.trim() ? (
            <CommandGroup heading="Events">
              {results.map((event) => (
                <CommandItem
                  key={event.id}
                  value={`${event.id} ${event.user_id} ${event.action} ${event.element}`}
                  onSelect={() => openDashboardSearch(String(event.id))}
                >
                  <Activity />
                  <div className="flex min-w-0 flex-1 flex-col">
                    <span className="truncate font-medium">
                      {event.action} • {event.element}
                    </span>
                    <span className="truncate text-xs text-muted-foreground">
                      {event.user_id} · {formatTimeLabel(event.timestamp)}
                    </span>
                  </div>
                  <CommandShortcut>{event.id}</CommandShortcut>
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
        </CommandList>
      </CommandDialog>
    </>
  );
}
