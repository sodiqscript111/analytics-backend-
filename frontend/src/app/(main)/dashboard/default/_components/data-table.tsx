"use client";

import { Search, X } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { EventItem, SearchFilters } from "@/lib/analytics-dashboard";

interface DataTableProps {
  data: EventItem[];
  filters: SearchFilters;
  isSearchMode: boolean;
  isLoading: boolean;
  isLoadingMore: boolean;
  nextCursor: string;
  resultCount: number;
  searchTookMs: number;
  sourceLabel: string;
  onFilterChange: (next: Partial<SearchFilters>) => void;
  onClearFilters: () => void;
  onLoadMore: () => void;
}

function truncateId(id: number | undefined): string {
  if (!id) return "-";
  const value = String(id);
  return value.length > 10 ? `${value.slice(0, 6)}...${value.slice(-4)}` : value;
}

function formatTimestamp(value: string | undefined): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return `${date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  })} ${date.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
  })}`;
}

const actionColors: Record<string, "default" | "secondary" | "outline"> = {
  click: "default",
  scroll: "secondary",
  hover: "outline",
  submit: "default",
  navigate: "secondary",
  focus: "outline",
  blur: "outline",
  keypress: "secondary",
};

const actionOptions = ["", "click", "scroll", "hover", "submit", "navigate", "focus", "blur", "keypress"];

export function DataTable({
  data,
  filters,
  isSearchMode,
  isLoading,
  isLoadingMore,
  nextCursor,
  resultCount,
  searchTookMs,
  sourceLabel,
  onFilterChange,
  onClearFilters,
  onLoadMore,
}: DataTableProps) {
  return (
    <Card>
      <CardHeader className="gap-4">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle>Event Log</CardTitle>
            <CardDescription>
              {isSearchMode
                ? `${resultCount} search results from ${sourceLabel}${searchTookMs ? ` in ${searchTookMs}ms` : ""}`
                : `${resultCount} recent events from ${sourceLabel}`}
            </CardDescription>
          </div>
          <Badge variant={isSearchMode ? "default" : "outline"}>
            {isSearchMode ? "Search mode" : "Default view"}
          </Badge>
        </div>

        <div className="grid gap-3 md:grid-cols-5">
          <div className="md:col-span-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder="Search by id, user, action, or element"
                value={filters.query}
                onChange={(event) => onFilterChange({ query: event.target.value })}
              />
            </div>
          </div>

          <select
            className="border-input h-9 rounded-md border bg-transparent px-3 text-sm shadow-xs"
            value={filters.action}
            onChange={(event) => onFilterChange({ action: event.target.value })}
          >
            {actionOptions.map((option) => (
              <option key={option || "all"} value={option}>
                {option ? option : "All actions"}
              </option>
            ))}
          </select>

          <Input
            placeholder="Filter by user"
            value={filters.userId}
            onChange={(event) => onFilterChange({ userId: event.target.value })}
          />

          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              onClick={onClearFilters}
              disabled={!isSearchMode}
            >
              <X className="size-4" />
              Clear
            </Button>
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-2">
          <Input
            type="datetime-local"
            value={filters.from}
            onChange={(event) => onFilterChange({ from: event.target.value })}
          />
          <Input
            type="datetime-local"
            value={filters.to}
            onChange={(event) => onFilterChange({ to: event.target.value })}
          />
        </div>
      </CardHeader>

      <CardContent>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[110px]">ID</TableHead>
                <TableHead>User</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Element</TableHead>
                <TableHead className="text-right">Duration</TableHead>
                <TableHead className="text-right">Timestamp</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="py-8 text-center text-muted-foreground"
                  >
                    Loading events...
                  </TableCell>
                </TableRow>
              ) : data.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="py-8 text-center text-muted-foreground"
                  >
                    {isSearchMode
                      ? "No events matched the current filters"
                      : "No events recorded yet"}
                  </TableCell>
                </TableRow>
              ) : (
                data.map((event, index) => (
                  <TableRow key={`${event.id}-${index}`}>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {truncateId(event.id)}
                    </TableCell>
                    <TableCell className="font-medium">{event.user_id}</TableCell>
                    <TableCell>
                      <Badge
                        variant={actionColors[event.action] ?? "default"}
                        className="capitalize"
                      >
                        {event.action}
                      </Badge>
                    </TableCell>
                    <TableCell>{event.element}</TableCell>
                    <TableCell className="text-right font-mono tabular-nums">
                      {event.duration?.toFixed(2)}ms
                    </TableCell>
                    <TableCell className="text-right text-muted-foreground">
                      {formatTimestamp(event.timestamp)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {isSearchMode && nextCursor ? (
          <div className="mt-4 flex justify-end">
            <Button
              variant="outline"
              size="sm"
              onClick={onLoadMore}
              disabled={isLoadingMore}
            >
              {isLoadingMore ? "Loading..." : "Load more"}
            </Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
