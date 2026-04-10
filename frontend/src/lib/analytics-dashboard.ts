"use client";

export const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";

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

export interface SearchFilters {
  query: string;
  action: string;
  userId: string;
  from: string;
  to: string;
}

export interface SearchEventsResponse {
  items: EventItem[];
  next_cursor?: string;
  total: number;
  took_ms: number;
  source: string;
}

export const EMPTY_FILTERS: SearchFilters = {
  query: "",
  action: "",
  userId: "",
  from: "",
  to: "",
};

export function hasActiveFilters(filters: SearchFilters): boolean {
  return Boolean(
    filters.query ||
      filters.action ||
      filters.userId ||
      filters.from ||
      filters.to,
  );
}

export function buildSearchParams(
  filters: SearchFilters,
  cursor?: string,
  size = 20,
): URLSearchParams {
  const params = new URLSearchParams();

  if (filters.query) params.set("q", filters.query);
  if (filters.action) params.set("action", filters.action);
  if (filters.userId) params.set("user_id", filters.userId);
  if (filters.from) params.set("from", filters.from);
  if (filters.to) params.set("to", filters.to);
  params.set("size", String(size));

  if (cursor) {
    params.set("cursor", cursor);
  }

  return params;
}

export function formatDateTimeLocal(value: string): string {
  if (!value) return "";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }

  const year = parsed.getFullYear();
  const month = String(parsed.getMonth() + 1).padStart(2, "0");
  const day = String(parsed.getDate()).padStart(2, "0");
  const hours = String(parsed.getHours()).padStart(2, "0");
  const minutes = String(parsed.getMinutes()).padStart(2, "0");

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}
