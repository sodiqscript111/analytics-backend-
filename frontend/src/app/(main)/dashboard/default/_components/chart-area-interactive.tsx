"use client";

import * as React from "react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import type { AnalyticsData, EventItem } from "@/lib/analytics-dashboard";

const chartConfig = {
  count: {
    label: "Event Count",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig;

interface ChartAreaInteractiveProps {
  analytics: AnalyticsData | null;
  recentFeed: EventItem[];
}

export function ChartAreaInteractive({
  analytics,
  recentFeed,
}: ChartAreaInteractiveProps) {
  const chartData = React.useMemo(() => {
    if (!analytics?.action_counts) return [];
    return Object.entries(analytics.action_counts)
      .map(([action, count]) => ({
        action,
        count: Number(count),
      }))
      .sort((left, right) => right.count - left.count);
  }, [analytics]);

  const displayFeed = React.useMemo(() => recentFeed.slice(0, 7), [recentFeed]);

  return (
    <div className="grid grid-cols-1 gap-4 @3xl/main:grid-cols-5">
      <Card className="@3xl/main:col-span-3">
        <CardHeader>
          <CardTitle>Action Breakdown</CardTitle>
          <CardDescription>
            Event count per action type from ClickHouse.
          </CardDescription>
        </CardHeader>
        <CardContent className="px-2 pt-4 sm:px-6 sm:pt-6">
          <ChartContainer
            config={chartConfig}
            className="aspect-auto h-62 w-full"
          >
            <BarChart data={chartData} layout="vertical">
              <CartesianGrid horizontal={false} />
              <XAxis type="number" tickLine={false} axisLine={false} />
              <YAxis
                dataKey="action"
                type="category"
                tickLine={false}
                axisLine={false}
                width={80}
                className="text-xs"
              />
              <ChartTooltip
                cursor={false}
                content={<ChartTooltipContent indicator="line" />}
              />
              <Bar
                dataKey="count"
                fill="var(--color-count)"
                radius={[0, 4, 4, 0]}
              />
            </BarChart>
          </ChartContainer>
        </CardContent>
      </Card>

      <Card className="@3xl/main:col-span-2">
        <CardHeader>
          <CardTitle>Recent Feed</CardTitle>
          <CardDescription>
            Live events from the SSE stream.
          </CardDescription>
        </CardHeader>
        <CardContent className="px-2 sm:px-6">
          <div className="space-y-3">
            {displayFeed.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                No recent events
              </p>
            ) : (
              displayFeed.map((event) => (
                <div
                  key={event.id}
                  className="flex items-center justify-between rounded-lg border bg-card p-3 text-sm"
                >
                  <div className="flex flex-col gap-0.5">
                    <span className="font-medium capitalize">{event.action}</span>
                    <span className="text-xs text-muted-foreground">
                      {event.element} · {event.user_id}
                    </span>
                  </div>
                  <div className="flex flex-col items-end gap-0.5">
                    <span className="font-mono text-xs tabular-nums">
                      {event.duration?.toFixed(1)}ms
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {event.timestamp
                        ? new Date(event.timestamp).toLocaleTimeString("en-US", {
                            hour12: false,
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })
                        : "-"}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
