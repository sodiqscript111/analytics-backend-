"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

import type { EventItem } from "../page";

interface DataTableProps {
  data: EventItem[];
}

function truncateId(id: number | undefined): string {
  if (!id) return "—";
  const str = String(id);
  return str.length > 10 ? `${str.slice(0, 6)}…${str.slice(-4)}` : str;
}

function formatTimestamp(ts: string | undefined): string {
  if (!ts) return "—";
  try {
    const d = new Date(ts);
    return `${d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
    })} ${d.toLocaleTimeString("en-US", {
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
    })}`;
  } catch {
    return "—";
  }
}

const actionColors: Record<string, string> = {
  click: "default",
  scroll: "secondary",
  hover: "outline",
  submit: "default",
  navigate: "secondary",
  focus: "outline",
  blur: "outline",
  keypress: "secondary",
};

export function DataTable({ data }: DataTableProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Event Log</CardTitle>
        <CardDescription>
          {data.length} events from PostgreSQL
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[100px]">ID</TableHead>
                <TableHead>User</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Element</TableHead>
                <TableHead className="text-right">Duration</TableHead>
                <TableHead className="text-right">Timestamp</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="text-center text-muted-foreground py-8"
                  >
                    No events recorded yet
                  </TableCell>
                </TableRow>
              ) : (
                data.map((event, i) => (
                  <TableRow key={`${event.id}-${i}`}>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {truncateId(event.id)}
                    </TableCell>
                    <TableCell className="font-medium">
                      {event.user_id}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          (actionColors[event.action] as "default" | "secondary" | "outline") || "default"
                        }
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
      </CardContent>
    </Card>
  );
}
