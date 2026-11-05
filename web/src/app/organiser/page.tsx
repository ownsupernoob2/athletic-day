"use client";

import { ArrowRight, Check, Download } from "lucide-react";
import Link from "next/link";
import useSWR from "swr";
import { buttonClass, Card, cx, Empty, HouseBadge, PageTitle, Pill, Progress, Spinner } from "@/components/ui";
import { fetcher, timeAgo, type GroupStatus, type ScheduleData, type Settings, type Student } from "@/lib/api";

export default function Overview() {
  const { data: groups } = useSWR<GroupStatus[]>("/api/groups", fetcher, { refreshInterval: 15000 });
  const { data: settings } = useSWR<Settings>("/api/settings", fetcher);
  const { data: schedule } = useSWR<ScheduleData>("/api/schedule", fetcher);
  const { data: students } = useSWR<Student[]>("/api/students", fetcher);

  if (!groups || !settings || !schedule || !students) return <Spinner />;

  const total = groups.reduce((n, g) => n + g.students, 0);
  const ready = groups.reduce((n, g) => n + g.complete, 0);
  const reported = groups.filter((g) => g.students > 0 && g.complete === g.students).length;
  const hasSchedule = schedule.heats.length > 0;

  const steps = [
    { n: 1, title: "Collect entries", done: total > 0 && ready === total, text: total === 0 ? "Import your students first." : `${ready} of ${total} students have enough events.`, href: "/organiser/students", cta: total === 0 ? "Import students" : "See students" },
    { n: 2, title: "Build the schedule", done: hasSchedule, text: hasSchedule ? `${schedule.heats.length} heats created.` : "Create heats, randomise or edit by hand.", href: "/organiser/schedule", cta: hasSchedule ? "Edit schedule" : "Create schedule" },
    { n: 3, title: "Publish", done: settings.published, text: settings.published ? "Students and parents can see it." : "Make it visible to everyone.", href: "/organiser/schedule", cta: settings.published ? "View" : "Go publish" },
  ];

  return (
    <>
      <PageTitle
        title="Overview"
        subtitle={`${settings.title}. ${reported} of ${groups.length} groups are completely ready.`}
        actions={
          <a href="/api/export/zip" className={buttonClass("primary")}>
            <Download className="size-4" /> Download all entries (.zip)
          </a>
        }
      />

      <div className="mb-10 grid gap-4 md:grid-cols-3">
        {steps.map((s) => (
          <Card key={s.n} className="flex flex-col p-5">
            <div className="flex items-center gap-3">
              <span className={cx("grid size-7 place-items-center rounded-full text-sm font-semibold", s.done ? "bg-emerald-100 text-emerald-700" : "bg-zinc-100 text-zinc-600")}>{s.done ? <Check className="size-4" /> : s.n}</span>
              <h2 className="font-semibold">{s.title}</h2>
            </div>
            <p className="mt-2 flex-1 text-sm text-zinc-500">{s.text}</p>
            <Link href={s.href} className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-indigo-600 hover:text-indigo-700">
              {s.cta} <ArrowRight className="size-4" />
            </Link>
          </Card>
        ))}
      </div>

      <h2 className="mb-3 text-lg font-semibold">Who has filled in their group?</h2>
      {groups.length === 0 ? (
        <Empty title="No groups yet">Import students or add staff accounts to see progress here.</Empty>
      ) : (
        <Card className="overflow-x-auto">
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
                <tr key={g.code} className="hover:bg-zinc-50">
                  <td className="px-4 py-3">
                    <Link href={`/organiser/students?group=${g.code}`} className="inline-block" title="See this group's students">
                      <HouseBadge group={g.code} house={g.house} />
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-zinc-600">{g.staff.length ? g.staff.join(", ") : <span className="text-zinc-400">No staff account</span>}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div className="flex-1">
                        <Progress value={g.complete} max={g.students} />
                      </div>
                      <span className="w-12 text-right tabular-nums text-zinc-600">
                        {g.complete}/{g.students}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    {g.students > 0 && g.complete === g.students ? <Pill tone="green">Done</Pill> : g.started > 0 ? <Pill tone="amber">In progress</Pill> : <Pill>Not started</Pill>}
                  </td>
                  <td className="px-4 py-3 text-zinc-500">{g.last_update ? timeAgo(g.last_update) : "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
    </>
  );
}
