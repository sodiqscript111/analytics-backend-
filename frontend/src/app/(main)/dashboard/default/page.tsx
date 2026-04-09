"use client";

import * as React from "react";
import { ChartAreaInteractive } from "./_components/chart-area-interactive";
import { DataTable } from "./_components/data-table";
import { SectionCards } from "./_components/section-cards";

const API_BASE = "http://localhost:8080";

export interface AnalyticsData {
  action_counts: Record<string, number>;
  avg_duration: number;
  total_events: number;
  processing_type: string;
}

export interface EventItem {
  id: number;
  user_id: string;
  action: string;
  element: string;
  duration: number;
  timestamp: string;
}

async function fetchAnalytics(): Promise<AnalyticsData | null> {
  try {
    const res = await fetch(`${API_BASE}/analytics/clickhouse`, { cache: "no-store" });
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

async function fetchRecentFeed(): Promise<EventItem[]> {
  try {
    const res = await fetch(`${API_BASE}/events/recent`, { cache: "no-store" });
    if (!res.ok) return [];
    return await res.json();
  } catch {
    return [];
  }
}

async function fetchEvents(): Promise<EventItem[]> {
  try {
    const res = await fetch(`${API_BASE}/events`, { cache: "no-store" });
    if (!res.ok) return [];
    const data = await res.json();
    return data.events || [];
  } catch {
    return [];
  }
}

export default function Page() {
  const [analytics, setAnalytics] = React.useState<AnalyticsData | null>(null);
  const [recentFeed, setRecentFeed] = React.useState<EventItem[]>([]);
  const [events, setEvents] = React.useState<EventItem[]>([]);
  const [autoRefresh, setAutoRefresh] = React.useState(true);

  // Initial load of recent feed
  React.useEffect(() => {
    fetchRecentFeed().then(setRecentFeed);
  }, []);

  // SSE for recent feed
  React.useEffect(() => {
    const es = new EventSource(`${API_BASE}/events/stream`);

    es.onmessage = (event) => {
      try {
        const newEvent: EventItem = JSON.parse(event.data);
        setRecentFeed((prev) => {
          // Avoid duplicates if initial fetch and stream overlap
          if (prev.some((e) => e.id === newEvent.id)) return prev;
          return [newEvent, ...prev].slice(0, 50);
        });
      } catch (e) {
        console.error("Failed to parse SSE event", e);
      }
    };

    es.onerror = (err) => {
      console.error("EventSource failed:", err);
    };

    return () => {
      es.close();
    };
  }, []);

  const refresh = React.useCallback(async () => {
    const [a, e] = await Promise.all([
      fetchAnalytics(),
      fetchEvents(),
    ]); // Removed fetchRecentFeed from polling
    if (a) setAnalytics(a);
    setEvents(e);
  }, []);

  React.useEffect(() => {
    refresh();
  }, [refresh]);

  React.useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(refresh, 1000);
    return () => clearInterval(interval);
  }, [autoRefresh, refresh]);

  return (
    <div className="@container/main flex flex-col gap-4 md:gap-6">
      <SectionCards analytics={analytics} />
      <ChartAreaInteractive analytics={analytics} recentFeed={recentFeed} />
      <DataTable data={events} />
    </div>
  );
}
