"use client";

import { BookOpen, Download, Search } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";
import useSWR from "swr";
import { buttonClass, cx, Empty, HouseBadge, Input, PageTitle, Select, Spinner } from "@/components/ui";
import { ageGroups, fetcher, formatDate, houseNames, type Heat, type PublicSchedule } from "@/lib/api";

export default function SchedulePage() {
  const { data, error } = useSWR<PublicSchedule>("/api/public/schedule", fetcher);
  const [query, setQuery] = useState("");
  const [event, setEvent] = useState("");
  const [age, setAge] = useState("");
  const [house, setHouse] = useState("");

  const events = useMemo(() => {
    const seen = new Map<number, string>();
    data?.heats.forEach((h) => seen.set(h.event_id, h.event_name));
    return [...seen];
  }, [data]);

  // Keep a heat if it matches the dropdown filters and (when searching) contains a match.
  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    const matches = (h: Heat) =>
      h.students.some((s) => `${s.first_name} ${s.last_name} ${s.group_code} ${s.house}`.toLowerCase().includes(q) || h.event_name.toLowerCase().includes(q));
    return (data?.heats ?? []).filter(
      (h) =>
        (!event || h.event_id === Number(event)) &&
        (!age || h.age_group === age) &&
        (!house || h.students.some((s) => s.house === house)) &&
        (!q || matches(h)),
    );
  }, [data, query, event, age, house]);

  if (error) return <Empty title="Couldn't load the schedule">Please refresh the page in a moment.</Empty>;
  if (!data) return <Spinner />;

  if (!data.published)
    return (
      <>
        <PageTitle title={data.title} />
        <Empty title="The schedule isn't ready yet">The organisers are still working on it. Check back soon.</Empty>
      </>
    );

  // event -> age group -> heats
  const byEvent: { id: number; name: string; ages: { name: string; heats: Heat[] }[] }[] = [];
  for (const h of visible) {
    let ev = byEvent.find((e) => e.id === h.event_id);
    if (!ev) byEvent.push((ev = { id: h.event_id, name: h.event_name, ages: [] }));
    let ag = ev.ages.find((a) => a.name === h.age_group);
    if (!ag) ev.ages.push((ag = { name: h.age_group, heats: [] }));
    ag.heats.push(h);
  }
  byEvent.forEach((e) => e.ages.sort((a, b) => ageGroups.indexOf(a.name) - ageGroups.indexOf(b.name)));
  const q = query.trim().toLowerCase();
  const filtering = !!(q || event || age || house);

  return (
    <>
      <section className="mb-8 flex flex-wrap items-end justify-between gap-6 border-b border-signal pb-6">
        <div>
          <p className="mb-3 font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-signal">{data.event_date ? formatDate(data.event_date) : "Programme of events"}</p>
          <h1 className="text-[clamp(2.5rem,6vw,4.25rem)] font-light leading-[1.05] text-signal">{data.title}</h1>
        </div>
        <div className="flex flex-wrap gap-2">
          <a href="/api/public/schedule.csv" className={buttonClass("secondary")}>
            <Download className="size-4" /> Download CSV
          </a>
          <Link href="/schedule/print" className={buttonClass("primary")}>
            <BookOpen className="size-4" /> Print booklet
          </Link>
        </div>
      </section>

      <div className="mb-12 grid gap-3 sm:grid-cols-2 lg:grid-cols-[1fr_180px_150px_150px]">
        <div className="relative sm:col-span-2 lg:col-span-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Find a name, group or event" className="pl-9" aria-label="Search" />
        </div>
        <Select value={event} onChange={(e) => setEvent(e.target.value)} aria-label="Event">
          <option value="">All events</option>
          {events.map(([id, name]) => (
            <option key={id} value={id}>
              {name}
            </option>
          ))}
        </Select>
        <Select value={age} onChange={(e) => setAge(e.target.value)} aria-label="Age group">
          <option value="">All ages</option>
          {ageGroups.map((a) => (
            <option key={a}>{a}</option>
          ))}
        </Select>
        <Select value={house} onChange={(e) => setHouse(e.target.value)} aria-label="House">
          <option value="">All houses</option>
          {houseNames.map((h) => (
            <option key={h}>{h}</option>
          ))}
        </Select>
      </div>

      {byEvent.length === 0 && <Empty title="No results">Try a different name or clear the filters.</Empty>}

      <div className="space-y-20">
        {byEvent.map((ev, n) => (
          <section key={ev.id}>
            <div className="flex items-baseline gap-4 border-b border-signal pb-3">
              <span className="font-mono text-xs font-semibold tabular-nums text-signal">{String(n + 1).padStart(2, "0")}</span>
              <h2 className="text-4xl font-light text-signal sm:text-5xl">{ev.name}</h2>
            </div>
            {ev.ages.map((ag) => (
              <div key={ag.name} className="mt-8 grid gap-4 md:grid-cols-[160px_1fr] md:gap-8">
                <h3 className="font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-zinc-500 md:pt-4">{ag.name}</h3>
                <div className="divide-y divide-zinc-200 border-t border-zinc-300">
                  {ag.heats.map((h) => (
                    <div key={h.id} className="grid grid-cols-[4.5rem_1fr] gap-4 py-4 sm:grid-cols-[6rem_1fr]">
                      <div>
                        <p className="text-5xl font-light leading-[0.9] tabular-nums text-signal sm:text-6xl">{h.number}</p>
                        <p className="mt-2 font-mono text-[10px] font-semibold uppercase tracking-[0.16em] text-zinc-500">Heat{h.time && ` · ${h.time}`}</p>
                      </div>
                      <ol className="grid content-start gap-x-8 sm:grid-cols-2">
                        {h.students.map((s, i) => {
                          const hit = filtering && q !== "" && `${s.first_name} ${s.last_name} ${s.group_code} ${s.house}`.toLowerCase().includes(q);
                          return (
                            <li key={s.id} className={cx("flex items-center gap-3 border-b border-zinc-100 py-1.5 text-[15px] last:border-0", hit && "bg-yellow-100")}>
                              <span className="w-4 font-mono text-xs tabular-nums text-zinc-400">{i + 1}</span>
                              <span className="flex-1 truncate">
                                {s.first_name} <strong className="font-semibold">{s.last_name}</strong>
                              </span>
                              <HouseBadge group={s.group_code} house={s.house} />
                            </li>
                          );
                        })}
                      </ol>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </section>
        ))}
      </div>
    </>
  );
}
