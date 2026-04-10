"use client";

import * as React from "react";
import { RefreshCcw } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import { Button } from "@/components/ui/button";
import {
  API_BASE,
  EMPTY_FILTERS,
  type AnalyticsData,
  buildSearchParams,
  formatDateTimeLocal,
  hasActiveFilters,
  type SearchEventsResponse,
  type SearchFilters,
  type EventItem,
} from "@/lib/analytics-dashboard";

import { ChartAreaInteractive } from "./_components/chart-area-interactive";
import { DataTable } from "./_components/data-table";
import { SectionCards } from "./_components/section-cards";

async function fetchAnalytics(signal?: AbortSignal): Promise<AnalyticsData | null> {
  try {
    const res = await fetch(`${API_BASE}/analytics/clickhouse`, {
      cache: "no-store",
      signal,
    });
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

async function fetchRecentFeed(signal?: AbortSignal): Promise<EventItem[]> {
  try {
    const res = await fetch(`${API_BASE}/events/recent`, {
      cache: "no-store",
      signal,
    });
    if (!res.ok) return [];
    return await res.json();
  } catch {
    return [];
  }
}

async function fetchEvents(signal?: AbortSignal): Promise<EventItem[]> {
  try {
    const res = await fetch(`${API_BASE}/events`, { cache: "no-store", signal });
    if (!res.ok) return [];
    const data = await res.json();
    return data.events || [];
  } catch {
    return [];
  }
}

async function fetchSearchEvents(
  filters: SearchFilters,
  cursor?: string,
  signal?: AbortSignal,
): Promise<SearchEventsResponse | null> {
  try {
    const params = buildSearchParams(filters, cursor);
    const res = await fetch(`${API_BASE}/search/events?${params.toString()}`, {
      cache: "no-store",
      signal,
    });
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

export default function Page() {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const filters = React.useMemo<SearchFilters>(
    () => ({
      query: searchParams.get("q") ?? EMPTY_FILTERS.query,
      action: searchParams.get("action") ?? EMPTY_FILTERS.action,
      userId: searchParams.get("user_id") ?? EMPTY_FILTERS.userId,
      from: formatDateTimeLocal(searchParams.get("from") ?? EMPTY_FILTERS.from),
      to: formatDateTimeLocal(searchParams.get("to") ?? EMPTY_FILTERS.to),
    }),
    [searchParams],
  );

  const isSearchMode = hasActiveFilters(filters);

  const [analytics, setAnalytics] = React.useState<AnalyticsData | null>(null);
  const [recentFeed, setRecentFeed] = React.useState<EventItem[]>([]);
  const [events, setEvents] = React.useState<EventItem[]>([]);
  const [searchResponse, setSearchResponse] =
    React.useState<SearchEventsResponse | null>(null);
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const [isAnalyticsLoading, setIsAnalyticsLoading] = React.useState(false);
  const [isEventsLoading, setIsEventsLoading] = React.useState(false);
  const [isSearchLoading, setIsSearchLoading] = React.useState(false);
  const [isLoadingMore, setIsLoadingMore] = React.useState(false);
  const [lastUpdated, setLastUpdated] = React.useState<string>("");

  const updateLastUpdated = React.useCallback(() => {
    setLastUpdated(new Date().toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    }));
  }, []);

  const updateFilters = React.useCallback(
    (next: Partial<SearchFilters>) => {
      const params = new URLSearchParams(searchParams.toString());
      const merged = { ...filters, ...next };

      const mappings: Array<[string, string]> = [
        ["q", merged.query],
        ["action", merged.action],
        ["user_id", merged.userId],
        ["from", merged.from],
        ["to", merged.to],
      ];

      for (const [key, value] of mappings) {
        if (value) {
          params.set(key, value);
        } else {
          params.delete(key);
        }
      }

      const query = params.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, {
        scroll: false,
      });
    },
    [filters, pathname, router, searchParams],
  );

  const clearFilters = React.useCallback(() => {
    router.replace(pathname, { scroll: false });
  }, [pathname, router]);

  const refreshAnalytics = React.useCallback(async () => {
    setIsAnalyticsLoading(true);
    const data = await fetchAnalytics();
    if (data) {
      setAnalytics(data);
      updateLastUpdated();
    }
    setIsAnalyticsLoading(false);
  }, [updateLastUpdated]);

  const refreshDefaultEvents = React.useCallback(async () => {
    setIsEventsLoading(true);
    const rows = await fetchEvents();
    setEvents(rows);
    setIsEventsLoading(false);
  }, []);

  const runSearch = React.useCallback(
    async (cursor?: string, append = false) => {
      if (!hasActiveFilters(filters)) {
        setSearchResponse(null);
        return;
      }

      if (append) {
        setIsLoadingMore(true);
      } else {
        setIsSearchLoading(true);
      }

      const response = await fetchSearchEvents(filters, cursor);
      if (response) {
        setSearchResponse((previous) => {
          if (!append || !previous) {
            return response;
          }

          return {
            ...response,
            items: [...previous.items, ...response.items],
          };
        });
      } else if (!append) {
        setSearchResponse({
          items: [],
          total: 0,
          took_ms: 0,
          source: "elasticsearch",
        });
      }

      setIsSearchLoading(false);
      setIsLoadingMore(false);
    },
    [filters],
  );

  React.useEffect(() => {
    const controller = new AbortController();
    fetchRecentFeed(controller.signal).then(setRecentFeed);
    refreshAnalytics();
    refreshDefaultEvents();

    return () => {
      controller.abort();
    };
  }, [refreshAnalytics, refreshDefaultEvents]);

  React.useEffect(() => {
    const es = new EventSource(`${API_BASE}/events/stream`);

    es.onmessage = (event) => {
      try {
        const nextEvent: EventItem = JSON.parse(event.data);

        setRecentFeed((previous) => {
          if (previous.some((item) => item.id === nextEvent.id)) {
            return previous;
          }
          return [nextEvent, ...previous].slice(0, 50);
        });

        setEvents((previous) => {
          if (previous.some((item) => item.id === nextEvent.id)) {
            return previous;
          }
          return [nextEvent, ...previous].slice(0, 50);
        });
      } catch (error) {
        console.error("Failed to parse SSE event", error);
      }
    };

    es.onerror = (error) => {
      console.error("EventSource failed:", error);
    };

    return () => {
      es.close();
    };
  }, []);

  React.useEffect(() => {
    if (!isSearchMode) {
      setSearchResponse(null);
      return;
    }

    const controller = new AbortController();
    const timeout = window.setTimeout(async () => {
      setIsSearchLoading(true);
      const response = await fetchSearchEvents(filters, undefined, controller.signal);
      if (response) {
        setSearchResponse(response);
      } else {
        setSearchResponse({
          items: [],
          total: 0,
          took_ms: 0,
          source: "elasticsearch",
        });
      }
      setIsSearchLoading(false);
    }, 300);

    return () => {
      controller.abort();
      window.clearTimeout(timeout);
    };
  }, [filters, isSearchMode]);

  React.useEffect(() => {
    if (!autoRefresh) return;

    const interval = window.setInterval(() => {
      refreshAnalytics();
    }, 5000);

    return () => {
      window.clearInterval(interval);
    };
  }, [autoRefresh, refreshAnalytics]);

  const handleManualRefresh = React.useCallback(async () => {
    await refreshAnalytics();
    if (isSearchMode) {
      await runSearch();
      return;
    }
    await refreshDefaultEvents();
  }, [isSearchMode, refreshAnalytics, refreshDefaultEvents, runSearch]);

  const handleLoadMore = React.useCallback(async () => {
    if (!searchResponse?.next_cursor) return;
    await runSearch(searchResponse.next_cursor, true);
  }, [runSearch, searchResponse?.next_cursor]);

  const tableRows = isSearchMode
    ? searchResponse?.items ?? []
    : events;

  return (
    <div className="@container/main flex flex-col gap-4 md:gap-6">
      <div className="flex flex-col gap-3 rounded-xl border bg-card px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-semibold">Realtime Analytics</h1>
          <p className="text-sm text-muted-foreground">
            Live event feed, lighter refresh cadence, and fast event search.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant={autoRefresh ? "default" : "outline"}
            size="sm"
            onClick={() => setAutoRefresh((current) => !current)}
          >
            Auto refresh {autoRefresh ? "on" : "off"}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleManualRefresh}
            disabled={isAnalyticsLoading || isEventsLoading || isSearchLoading}
          >
            <RefreshCcw className="size-4" />
            Refresh
          </Button>
          <span className="text-xs text-muted-foreground">
            {lastUpdated ? `Updated ${lastUpdated}` : "Waiting for first refresh"}
          </span>
        </div>
      </div>

      <SectionCards analytics={analytics} />
      <ChartAreaInteractive analytics={analytics} recentFeed={recentFeed} />
      <DataTable
        data={tableRows}
        filters={filters}
        isSearchMode={isSearchMode}
        isLoading={isEventsLoading || isSearchLoading}
        isLoadingMore={isLoadingMore}
        nextCursor={searchResponse?.next_cursor ?? ""}
        resultCount={isSearchMode ? searchResponse?.total ?? tableRows.length : tableRows.length}
        searchTookMs={searchResponse?.took_ms ?? 0}
        sourceLabel={isSearchMode ? "Elasticsearch" : "PostgreSQL"}
        onFilterChange={updateFilters}
        onClearFilters={clearFilters}
        onLoadMore={handleLoadMore}
      />
    </div>
  );
}
