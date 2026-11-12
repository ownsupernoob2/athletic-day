"use client";

import { Check } from "lucide-react";
import useSWR from "swr";
import { Card, cx, HouseBadge, PageTitle, Pill, Progress as Bar, Spinner } from "@/components/ui";
import { fetcher, timeAgo, type Activity, type ActivityEvent, type GroupStatus, type Progress } from "@/lib/api";
import { RequireRole } from "@/lib/auth";

export default function ProgressPage() {
  return (
    <RequireRole>
      <ProgressView />
    </RequireRole>
  );
}

function ProgressView() {
  const { data: p } = useSWR<Progress>("/api/progress", fetcher, { refreshInterval: 5000 });
  if (!p) return <Spinner />;

  const steps = [
    { n: 1, title: "Groups pick events", done: p.groups_total > 0 && p.groups_done === p.groups_total, text: `${p.groups_done} of ${p.groups_total} groups are finished.` },
    { n: 2, title: "Schedule is built", done: p.schedule !== "none", text: p.schedule === "none" ? "The organiser builds it once every group is done." : `${p.heats} heats created.` },
    { n: 3, title: "Schedule is published", done: p.schedule === "published", text: p.schedule === "published" ? "Everyone can see it." : "Last step before the day." },
  ];

  return (
    <>
      <PageTitle
        title="Athletics day progress"
        subtitle={p.demo ? undefined : "How the setup of the day is going across the whole school."}
        actions={p.demo ? <span className="rounded bg-indigo-50 px-3 py-1.5 text-sm font-medium text-ink ring-1 ring-inset ring-signal/40">This page is only for testing</span> : undefined}
      />

      <Card className="mb-6 p-6">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <p className="text-5xl font-light tabular-nums text-ink">
            {p.groups_done}
            <span className="text-zinc-300">/{p.groups_total}</span>
            <span className="ml-3 text-lg text-zinc-500">groups finished</span>
          </p>
          <p className="text-sm text-zinc-500">
            {p.ready} of {p.students} students have at least {p.min_events} events.
          </p>
        </div>
        <div className="mt-5">
          <Bar value={p.groups_done} max={p.groups_total} />
        </div>
      </Card>

      <div className="mb-10 grid gap-4 md:grid-cols-3">
        {steps.map((s) => (
          <Card key={s.n} className="p-5">
            <div className="flex items-center gap-3">
              <span className={cx("grid size-7 place-items-center rounded-full text-sm font-semibold", s.done ? "bg-ink text-white" : "bg-zinc-100 text-zinc-600")}>{s.done ? <Check className="size-4" /> : s.n}</span>
              <h2 className="font-semibold">{s.title}</h2>
            </div>
            <p className="mt-2 text-sm text-zinc-500">{s.text}</p>
          </Card>
        ))}
      </div>

      {p.demo && p.groups && <DemoView groups={p.groups} />}
    </>
  );
}

function DemoView({ groups }: { groups: GroupStatus[] }) {
  const { data: activity } = useSWR<Activity>("/api/activity", fetcher, { refreshInterval: 3000 });
  const done = (g: GroupStatus) => g.students > 0 && g.complete === g.students;
  const counts = {
    done: groups.filter(done).length,
    progress: groups.filter((g) => !done(g) && g.started > 0).length,
    none: groups.filter((g) => g.started === 0).length,
  };

  return (
    <>
      <h2 className="mb-3 text-xl font-light text-signal">Who has finished</h2>
      <p className="mb-3 text-sm text-zinc-500">
        {counts.done} finished · {counts.progress} in progress · {counts.none} not started
      </p>
      <Card className="mb-12 overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead className="border-b border-zinc-100 text-xs uppercase tracking-wider text-zinc-400">
            <tr>
              <th className="px-4 py-3 font-medium">Group</th>
              <th className="px-4 py-3 font-medium">Staff</th>
              <th className="w-56 px-4 py-3 font-medium">Students ready</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Last change</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {groups.map((g) => (
              <tr key={g.code}>
                <td className="px-4 py-2.5">
                  <HouseBadge group={g.code} house={g.house} />
                </td>
                <td className="px-4 py-2.5 text-zinc-600">{g.staff.length ? g.staff.join(", ") : <span className="text-zinc-400">No staff account</span>}</td>
                <td className="px-4 py-2.5">
                  <div className="flex items-center gap-3">
                    <div className="flex-1">
                      <Bar value={g.complete} max={g.students} />
                    </div>
                    <span className="w-12 text-right tabular-nums text-zinc-600">
                      {g.complete}/{g.students}
                    </span>
                  </div>
                </td>
                <td className="px-4 py-2.5">{done(g) ? <Pill tone="green">Finished</Pill> : g.started > 0 ? <Pill tone="amber">In progress</Pill> : <Pill>Not started</Pill>}</td>
                <td className="px-4 py-2.5 text-zinc-500">{g.last_update ? timeAgo(g.last_update) : "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

      <div className="grid gap-8 lg:grid-cols-[1fr_320px]">
        <section>
          <h2 className="mb-3 text-xl font-light text-signal">What the server is doing</h2>
          <Card className="max-h-[520px] overflow-y-auto">
            {!activity && <Spinner />}
            {activity && activity.events.length === 0 && <p className="p-6 text-center text-sm text-zinc-400">Nothing has happened yet.</p>}
            <ul className="divide-y divide-zinc-100">
              {activity?.events.map((e, i) => (
                <EventRow key={`${e.at}-${i}`} e={e} />
              ))}
            </ul>
          </Card>
          <p className="mt-2 text-xs text-zinc-400">Updates every few seconds. Shows the last 150 events since the server started.</p>
        </section>

        <section>
          <h2 className="mb-3 text-xl font-light text-signal">Server</h2>
          <Card className="p-5">
            {activity ? (
              <dl className="grid grid-cols-2 gap-x-4 gap-y-4 text-sm">
                <Stat label="Running for" value={uptime(activity.system.uptime_seconds)} />
                <Stat label="Requests" value={activity.system.requests.toLocaleString()} />
                <Stat label="Students" value={activity.system.students.toLocaleString()} />
                <Stat label="Event picks" value={activity.system.entries.toLocaleString()} />
                <Stat label="Heats" value={String(activity.system.heats)} />
                <Stat label="Accounts" value={String(activity.system.accounts)} />
                <Stat label="Signed in now" value={String(activity.system.sessions)} />
                <Stat label="Database" value={`${activity.system.database_kb.toLocaleString()} KB`} />
                <Stat label="Memory" value={`${activity.system.memory_mb.toFixed(1)} MB`} />
                <Stat label="KAMAR" value={activity.system.kamar_enabled ? (activity.system.kamar_last ? "Receiving" : "Waiting") : "Off"} />
              </dl>
            ) : (
              <Spinner />
            )}
          </Card>
        </section>
      </div>
    </>
  );
}

const kindStyle: Record<string, string> = {
  auth: "bg-zinc-100 text-zinc-700",
  entries: "bg-indigo-50 text-ink",
  students: "bg-emerald-50 text-emerald-700",
  schedule: "bg-amber-50 text-amber-800",
  staff: "bg-zinc-100 text-zinc-700",
  kamar: "bg-sky-50 text-sky-700",
  system: "bg-zinc-100 text-zinc-500",
};

function EventRow({ e }: { e: ActivityEvent }) {
  return (
    <li className="flex items-baseline gap-3 px-4 py-2 text-sm">
      <span className="w-[4.5rem] shrink-0 tabular-nums text-xs text-zinc-400">{new Date(e.at).toLocaleTimeString([], { hour12: false })}</span>
      <span className={cx("w-[4.5rem] shrink-0 rounded px-1.5 py-0.5 text-center text-[10px] font-semibold uppercase tracking-wide", kindStyle[e.kind] ?? kindStyle.system)}>{e.kind}</span>
      <span className="min-w-0 flex-1">
        {e.actor && <strong className="font-semibold text-ink">{e.actor} </strong>}
        {e.text}
      </span>
    </li>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wider text-zinc-400">{label}</dt>
      <dd className="mt-0.5 text-lg font-light tabular-nums text-ink">{value}</dd>
    </div>
  );
}

function uptime(s: number) {
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  return d ? `${d}d ${h}h` : h ? `${h}h ${m}m` : `${m}m ${s % 60}s`;
}
