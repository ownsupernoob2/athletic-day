"use client";

import { ArrowRight, Check } from "lucide-react";
import Link from "next/link";
import useSWR from "swr";
import { Card, cx, HouseBadge, Progress as Bar } from "@/components/ui";
import { fetcher, type GroupStatus, type Progress } from "@/lib/api";

/**
 * How the whole school is getting on, shown at the top of the staff page.
 * In the demo (server flag) it also lists every group, who looks after it and whether it is finished.
 */
export function DayProgress() {
  const { data: p } = useSWR<Progress>("/api/progress", fetcher, { refreshInterval: 5000 });
  if (!p) return null;

  const steps = [
    { label: "Groups pick events", done: p.groups_total > 0 && p.groups_done === p.groups_total },
    { label: "Schedule built", done: p.schedule !== "none" },
    { label: "Schedule published", done: p.schedule === "published" },
  ];

  return (
    <Card className="mb-8 p-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-zinc-500">Whole school</p>
          <p className="mt-1 text-4xl font-light tabular-nums text-ink">
            {p.groups_done}
            <span className="text-zinc-300">/{p.groups_total}</span>
            <span className="ml-2 text-base text-zinc-500">groups finished</span>
          </p>
        </div>
        <Link href="/progress" className="inline-flex items-center gap-1 text-sm text-signal hover:text-ink">
          Full progress page <ArrowRight className="size-4" strokeWidth={1.5} />
        </Link>
      </div>

      <div className="mt-4">
        <Bar value={p.groups_done} max={p.groups_total} />
      </div>

      <ol className="mt-4 flex flex-wrap gap-x-6 gap-y-2 text-sm">
        {steps.map((s, i) => (
          <li key={s.label} className={cx("flex items-center gap-2", s.done ? "text-ink" : "text-zinc-500")}>
            <span className={cx("grid size-5 place-items-center rounded-full text-[11px] font-semibold", s.done ? "bg-ink text-white" : "bg-zinc-100 text-zinc-500")}>
              {s.done ? <Check className="size-3" strokeWidth={3} /> : i + 1}
            </span>
            {s.label}
          </li>
        ))}
      </ol>

      {p.demo && p.groups && <Groups groups={p.groups} />}
    </Card>
  );
}

function Groups({ groups }: { groups: GroupStatus[] }) {
  const finished = (g: GroupStatus) => g.students > 0 && g.complete === g.students;
  return (
    <div className="mt-6 border-t border-zinc-100 pt-5">
      <p className="mb-3 font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-zinc-500">Who has finished (demo view)</p>
      <ul className="grid gap-x-6 gap-y-1.5 sm:grid-cols-2 lg:grid-cols-3">
        {groups.map((g) => (
          <li key={g.code} className="flex items-center gap-2 text-sm">
            <HouseBadge group={g.code} house={g.house} />
            <span className="min-w-0 flex-1 truncate text-zinc-600">{g.staff.length ? g.staff.join(", ") : <span className="text-zinc-400">No staff account</span>}</span>
            {finished(g) ? (
              <span className="inline-flex items-center gap-1 text-xs font-medium text-ink">
                <Check className="size-3.5" strokeWidth={3} /> Finished
              </span>
            ) : (
              <span className="text-xs tabular-nums text-signal">
                {g.complete}/{g.students}
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
