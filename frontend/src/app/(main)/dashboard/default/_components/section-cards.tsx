"use client";

import { Activity, Database, TrendingUp, Zap } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardAction,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { AnalyticsData } from "@/lib/analytics-dashboard";

interface SectionCardsProps {
  analytics: AnalyticsData | null;
}

export function SectionCards({ analytics }: SectionCardsProps) {
  const totalEvents = analytics?.total_events ?? 0;
  const avgDuration = analytics?.avg_duration ?? 0;
  const uniqueActions = analytics?.action_counts
    ? Object.keys(analytics.action_counts).length
    : 0;
  const processingType = analytics?.processing_type ?? "-";

  return (
    <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 *:data-[slot=card]:bg-linear-to-t *:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card *:data-[slot=card]:shadow-xs dark:*:data-[slot=card]:bg-card">
      <Card className="@container/card">
        <CardHeader>
          <CardDescription>Total Events</CardDescription>
          <CardTitle className="text-2xl tabular-nums font-semibold @[250px]/card:text-3xl">
            {totalEvents.toLocaleString()}
          </CardTitle>
          <CardAction>
            <Badge variant="outline">
              <Activity className="size-3" />
              Live
            </Badge>
          </CardAction>
        </CardHeader>
        <CardFooter className="flex-col items-start gap-1.5 text-sm">
          <div className="line-clamp-1 flex gap-2 font-medium">
            Events processed <TrendingUp className="size-4" />
          </div>
          <div className="text-muted-foreground">
            Total events ingested via pipeline
          </div>
        </CardFooter>
      </Card>

      <Card className="@container/card">
        <CardHeader>
          <CardDescription>Avg Duration</CardDescription>
          <CardTitle className="text-2xl tabular-nums font-semibold @[250px]/card:text-3xl">
            {avgDuration.toFixed(2)}ms
          </CardTitle>
          <CardAction>
            <Badge variant="outline">
              <Zap className="size-3" />
              Perf
            </Badge>
          </CardAction>
        </CardHeader>
        <CardFooter className="flex-col items-start gap-1.5 text-sm">
          <div className="line-clamp-1 flex gap-2 font-medium">
            Average event duration
          </div>
          <div className="text-muted-foreground">
            Milliseconds per event interaction
          </div>
        </CardFooter>
      </Card>

      <Card className="@container/card">
        <CardHeader>
          <CardDescription>Unique Actions</CardDescription>
          <CardTitle className="text-2xl tabular-nums font-semibold @[250px]/card:text-3xl">
            {uniqueActions}
          </CardTitle>
          <CardAction>
            <Badge variant="outline">
              <TrendingUp className="size-3" />
              Types
            </Badge>
          </CardAction>
        </CardHeader>
        <CardFooter className="flex-col items-start gap-1.5 text-sm">
          <div className="line-clamp-1 flex gap-2 font-medium">
            Distinct action types <TrendingUp className="size-4" />
          </div>
          <div className="text-muted-foreground">
            click, scroll, hover, submit, and navigation events
          </div>
        </CardFooter>
      </Card>

      <Card className="@container/card">
        <CardHeader>
          <CardDescription>Processing Engine</CardDescription>
          <CardTitle className="text-2xl tabular-nums font-semibold capitalize @[250px]/card:text-3xl">
            {processingType}
          </CardTitle>
          <CardAction>
            <Badge variant="outline">
              <Database className="size-3" />
              Engine
            </Badge>
          </CardAction>
        </CardHeader>
        <CardFooter className="flex-col items-start gap-1.5 text-sm">
          <div className="line-clamp-1 flex gap-2 font-medium">
            Analytics engine type
          </div>
          <div className="text-muted-foreground">
            Go + Redis + ClickHouse + Postgres + Elasticsearch
          </div>
        </CardFooter>
      </Card>
    </div>
  );
}
