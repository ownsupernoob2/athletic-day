"use client";

import { AlertTriangle, ExternalLink, Eye, EyeOff, Plus, Search, Shuffle, Trash2, UserPlus, X, ListOrdered } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";
import useSWR from "swr";
import {
  Button,
  buttonClass,
  Card,
  Confirm,
  cx,
  Empty,
  HouseBadge,
  Input,
  Modal,
  PageTitle,
  Pill,
  Select,
  Spinner,
  useToast,
} from "@/components/ui";
import { ageGroups, api, errorMessage, fetcher, timeAgo, type Heat, type ScheduleData, type Settings, type Student } from "@/lib/api";

export default function SchedulePage() {
  const toast = useToast();
  const { data, mutate } = useSWR<ScheduleData>("/api/schedule", fetcher);
  const [eventFilter, setEventFilter] = useState("");
  const [ageFilter, setAgeFilter] = useState("");
  const [confirm, setConfirm] = useState<null | { title: string; body: string; label: string; run: () => Promise<void> }>(null);
  const [adding, setAdding] = useState<Heat | null>(null);
  const [showUnplaced, setShowUnplaced] = useState(false);

  const heats = data?.heats;
  const events = useMemo(() => {
    const m = new Map<number, string>();
    heats?.forEach((h) => m.set(h.event_id, h.event_name));
    return [...m];
  }, [heats]);

  if (!data) return <Spinner />;
  const { settings, unplaced } = data;
  const hasSchedule = data.heats.length > 0;
  const stale = hasSchedule && settings.entries_changed_at > settings.generated_at;

  // Run an API call that returns the whole updated schedule.
  async function apply(fn: () => Promise<ScheduleData>, message?: string) {
    try {
      const next = await fn();
      await mutate(next, false);
      if (message) toast(message);
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  }

  const generate = (randomize: boolean, eventId = 0) =>
    apply(() => api.post<ScheduleData>("/api/schedule/generate", { randomize, event_id: eventId }), randomize ? "Heats shuffled" : "Schedule created");

  async function setPublished(published: boolean) {
    try {
      await api.post<Settings>("/api/schedule/publish", { published });
      await mutate();
      toast(published ? "Published. Everyone can see it now." : "Unpublished. Only organisers can see it.");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  }

  const ask = (c: NonNullable<typeof confirm>) => setConfirm(c);

  const visible = data.heats.filter((h) => (!eventFilter || h.event_id === Number(eventFilter)) && (!ageFilter || h.age_group === ageFilter));
  const byEvent = events
    .filter(([id]) => !eventFilter || id === Number(eventFilter))
    .map(([id, name]) => ({ id, name, heats: visible.filter((h) => h.event_id === id) }));

  return (
    <>
      <PageTitle
        title="Schedule"
        subtitle="Create the heats, tweak them by hand, then publish for everyone."
        actions={
          hasSchedule && (
            <Link href="/schedule" className={buttonClass("secondary")}>
              <ExternalLink className="size-4" /> Public page
            </Link>
          )
        }
      />

      <Card className="mb-6 p-5">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="font-semibold">{!hasSchedule ? "No schedule yet" : settings.published ? "Published" : "Draft (only you can see it)"}</h2>
              {hasSchedule && (settings.published ? <Pill tone="green">Live</Pill> : <Pill tone="amber">Draft</Pill>)}
            </div>
            <p className="mt-0.5 text-sm text-zinc-500">
              {hasSchedule
                ? `${data.heats.length} heats, up to ${settings.heat_size} per heat. Edits here show on the public page straight away.`
                : `Heats of up to ${settings.heat_size} are made for each event and age group, from the entries staff filled in.`}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {!hasSchedule ? (
              <>
                <Button onClick={() => generate(false)}>
                  <ListOrdered className="size-4" /> Create A–Z
                </Button>
                <Button variant="primary" onClick={() => generate(true)}>
                  <Shuffle className="size-4" /> Create randomly
                </Button>
              </>
            ) : (
              <>
                <Button
                  onClick={() =>
                    ask({ title: "Shuffle every heat?", body: "All heats are rebuilt in a new random order. Any changes you made by hand are lost.", label: "Shuffle all", run: () => generate(true) })
                  }
                >
                  <Shuffle className="size-4" /> Shuffle all
                </Button>
                <Button
                  onClick={() =>
                    ask({ title: "Reset to A–Z?", body: "All heats are rebuilt in surname order. Any changes you made by hand are lost.", label: "Reset", run: () => generate(false) })
                  }
                >
                  <ListOrdered className="size-4" /> Reset A–Z
                </Button>
                {settings.published ? (
                  <Button onClick={() => setPublished(false)}>
                    <EyeOff className="size-4" /> Unpublish
                  </Button>
                ) : (
                  <Button variant="primary" onClick={() => setPublished(true)}>
                    <Eye className="size-4" /> Publish
                  </Button>
                )}
              </>
            )}
          </div>
        </div>
      </Card>

      {hasSchedule && (
        <TimesCard
          settings={settings}
          heats={data.heats}
          onApply={(start, end) => apply(() => api.post<ScheduleData>("/api/schedule/times", { start, end }), "Times set")}
          onClear={() => apply(() => api.post<ScheduleData>("/api/schedule/times", { clear: true }), "Times cleared")}
          ask={ask}
        />
      )}

      {stale && (
        <Banner tone="amber">
          Entries changed {timeAgo(settings.entries_changed_at)}, after the schedule was made. Check the list below, or shuffle again to include the changes.
        </Banner>
      )}
      {unplaced.length > 0 && (
        <Banner tone="amber">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>
              <strong>{unplaced.length}</strong> {unplaced.length === 1 ? "entry isn't" : "entries aren't"} in a heat yet.
            </span>
            <button className="font-medium underline" onClick={() => setShowUnplaced((v) => !v)}>
              {showUnplaced ? "Hide" : "Show and fix"}
            </button>
          </div>
          {showUnplaced && (
            <ul className="mt-3 divide-y divide-amber-200/70 rounded bg-white/60">
              {unplaced.map((u) => {
                const options = data.heats.filter((h) => h.event_id === u.event_id);
                const ev = events.find(([id]) => id === u.event_id)?.[1] ?? "Event";
                return (
                  <li key={`${u.student.id}-${u.event_id}`} className="flex flex-wrap items-center gap-3 px-3 py-2 text-sm">
                    <span className="flex-1">
                      {u.student.first_name} {u.student.last_name} <span className="text-zinc-500">· {ev} · {u.age_group}</span>
                    </span>
                    <Select
                      className="!h-8 !w-44 text-xs"
                      value=""
                      aria-label="Place in heat"
                      onChange={(e) => apply(() => api.post<ScheduleData>(`/api/heats/${e.target.value}/students`, { student_id: u.student.id }), "Placed")}
                    >
                      <option value="">{options.length ? "Place in heat…" : "No heats for this event"}</option>
                      {options.map((h) => (
                        <option key={h.id} value={h.id}>
                          {h.age_group} · Heat {h.number}
                        </option>
                      ))}
                    </Select>
                  </li>
                );
              })}
            </ul>
          )}
        </Banner>
      )}

      {!hasSchedule ? (
        <Empty title="Nothing to show yet">
          Press <strong>Create A–Z</strong> or <strong>Create randomly</strong> above. You can still change everything afterwards.
        </Empty>
      ) : (
        <>
          <div className="mb-6 flex flex-wrap gap-3">
            <Select value={eventFilter} onChange={(e) => setEventFilter(e.target.value)} className="!w-52" aria-label="Event">
              <option value="">All events</option>
              {events.map(([id, name]) => (
                <option key={id} value={id}>
                  {name}
                </option>
              ))}
            </Select>
            <Select value={ageFilter} onChange={(e) => setAgeFilter(e.target.value)} className="!w-40" aria-label="Age group">
              <option value="">All ages</option>
              {ageGroups.map((a) => (
                <option key={a}>{a}</option>
              ))}
            </Select>
          </div>

          <div className="space-y-12">
            {byEvent.map((ev) => (
              <section key={ev.id}>
                <div className="mb-4 flex flex-wrap items-center justify-between gap-2 border-b border-zinc-200 pb-2">
                  <h2 className="text-xl font-semibold tracking-tight">{ev.name}</h2>
                  <div className="flex items-center gap-2">
                    <Select
                      className="!h-8 !w-36 text-xs"
                      value=""
                      aria-label={`Add heat to ${ev.name}`}
                      onChange={(e) => apply(() => api.post<ScheduleData>("/api/heats", { event_id: ev.id, age_group: e.target.value }), "Heat added")}
                    >
                      <option value="">+ Add a heat…</option>
                      {ageGroups.map((a) => (
                        <option key={a}>{a}</option>
                      ))}
                    </Select>
                    <Button
                      size="sm"
                      onClick={() =>
                        ask({
                          title: `Shuffle ${ev.name}?`,
                          body: "This event's heats are rebuilt in a new random order. Manual changes to this event are lost.",
                          label: "Shuffle",
                          run: () => generate(true, ev.id),
                        })
                      }
                    >
                      <Shuffle className="size-3.5" /> Shuffle
                    </Button>
                  </div>
                </div>
                <div className="space-y-6">
                  {ageGroups
                    .filter((a) => ev.heats.some((h) => h.age_group === a))
                    .map((a) => (
                      <div key={a}>
                        <h3 className="mb-2.5 text-sm font-semibold text-indigo-600">{a}</h3>
                        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                          {ev.heats
                            .filter((h) => h.age_group === a)
                            .map((h) => (
                              <HeatCard key={`${h.id}-${h.time}`} heat={h} siblings={data.heats.filter((x) => x.event_id === h.event_id)} apply={apply} onAdd={() => setAdding(h)} />
                            ))}
                        </div>
                      </div>
                    ))}
                </div>
              </section>
            ))}
          </div>

          <div className="mt-12 flex justify-end">
            <Button
              variant="danger"
              size="sm"
              onClick={() =>
                ask({ title: "Delete the whole schedule?", body: "All heats are removed and the schedule is unpublished. Entries stay as they are.", label: "Delete schedule", run: () => apply(() => api.del<ScheduleData>("/api/schedule"), "Schedule deleted") })
              }
            >
              <Trash2 className="size-4" /> Delete schedule
            </Button>
          </div>
        </>
      )}

      {confirm && <Confirm danger title={confirm.title} body={confirm.body} confirmLabel={confirm.label} onConfirm={confirm.run} onClose={() => setConfirm(null)} />}
      {adding && <AddStudentModal heat={adding} onClose={() => setAdding(null)} apply={apply} />}
    </>
  );
}

function Banner({ tone, children }: { tone: "amber"; children: React.ReactNode }) {
  return (
    <div className={cx("mb-4 flex gap-3 rounded p-4 text-sm", tone === "amber" && "bg-amber-50 text-amber-900")}>
      <AlertTriangle className="mt-0.5 size-4 shrink-0" />
      <div className="flex-1">{children}</div>
    </div>
  );
}

type Apply = (fn: () => Promise<ScheduleData>, message?: string) => Promise<void>;

function HeatCard({ heat, siblings, apply, onAdd }: { heat: Heat; siblings: Heat[]; apply: Apply; onAdd: () => void }) {
  const toast = useToast();
  const [time, setTime] = useState(heat.time);

  async function saveTime() {
    if (time.trim() === heat.time) return;
    try {
      await api.patch(`/api/heats/${heat.id}`, { time });
      toast("Time saved");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  }

  return (
    <Card className="overflow-hidden">
      <div className="flex items-center gap-2 border-b border-zinc-100 bg-zinc-50 px-3 py-2">
        <span className="text-sm font-semibold">Heat {heat.number}</span>
        <input
          type="time"
          value={time}
          onChange={(e) => setTime(e.target.value)}
          onBlur={saveTime}
          onKeyDown={(e) => e.key === "Enter" && e.currentTarget.blur()}
                    aria-label={`Time for heat ${heat.number}`}
          className="ml-auto h-7 w-[5.5rem] rounded-md border-0 bg-transparent px-2 text-right text-sm tabular-nums text-zinc-600 ring-1 ring-inset ring-transparent hover:ring-zinc-300 focus:bg-white focus:ring-2 focus:ring-ink focus:outline-none"
        />
        <button
          onClick={() => apply(() => api.del<ScheduleData>(`/api/heats/${heat.id}`), "Heat deleted")}
          className="rounded p-1 text-zinc-400 hover:bg-red-50 hover:text-red-600"
          aria-label={`Delete heat ${heat.number}`}
          title="Delete this heat"
        >
          <Trash2 className="size-3.5" />
        </button>
      </div>
      <ol className="divide-y divide-zinc-100">
        {heat.students.length === 0 && <li className="px-3 py-4 text-center text-sm text-zinc-400">Empty heat</li>}
        {heat.students.map((s, i) => (
          <li key={s.id} className="flex items-center gap-2 px-3 py-1.5 text-sm">
            <span className="w-4 text-xs tabular-nums text-zinc-400">{i + 1}</span>
            <span className="min-w-0 flex-1 truncate">
              {s.first_name} <strong className="font-semibold">{s.last_name}</strong>
            </span>
            <HouseBadge group={s.group_code} house={s.house} />
            <select
              value=""
              aria-label={`Move ${s.first_name} ${s.last_name}`}
              onChange={(e) => apply(() => api.post<ScheduleData>(`/api/heats/${e.target.value}/students`, { student_id: s.id }), "Moved")}
              className="h-7 w-[68px] rounded-md border-0 bg-white px-1 text-xs text-zinc-500 ring-1 ring-inset ring-zinc-200 hover:ring-zinc-300"
            >
              <option value="">Move…</option>
              {siblings
                .filter((x) => x.id !== heat.id)
                .map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.age_group} · Heat {x.number}
                  </option>
                ))}
            </select>
            <button
              onClick={() => apply(() => api.del<ScheduleData>(`/api/heats/${heat.id}/students/${s.id}`))}
              className="rounded p-1 text-zinc-400 hover:bg-red-50 hover:text-red-600"
              aria-label={`Take ${s.first_name} ${s.last_name} out of this heat`}
              title="Take out of this heat"
            >
              <X className="size-3.5" />
            </button>
          </li>
        ))}
      </ol>
      <button onClick={onAdd} className="flex w-full items-center justify-center gap-1.5 border-t border-zinc-100 py-2 text-xs font-medium text-indigo-600 hover:bg-indigo-50">
        <Plus className="size-3.5" /> Add student
      </button>
    </Card>
  );
}

function AddStudentModal({ heat, onClose, apply }: { heat: Heat; onClose: () => void; apply: Apply }) {
  const { data: students } = useSWR<Student[]>("/api/students", fetcher);
  const [q, setQ] = useState("");
  const inHeat = new Set(heat.students.map((s) => s.id));
  const matches = useMemo(() => {
    const tokens = q.toLowerCase().split(/\s+/).filter(Boolean);
    return (students ?? [])
      .filter((s) => !inHeat.has(s.id) && tokens.every((t) => `${s.first_name} ${s.last_name} ${s.group_code} ${s.age_group}`.toLowerCase().includes(t)))
      .slice(0, 40);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [students, q, heat]);

  return (
    <Modal title={`Add to ${heat.event_name}, ${heat.age_group}, heat ${heat.number}`} onClose={onClose}>
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
        <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search by name or group" className="pl-9" autoFocus />
      </div>
      <p className="mt-2 text-xs text-zinc-500">The student is also entered in this event, and taken out of their other heat for it.</p>
      <ul className="mt-3 max-h-80 divide-y divide-zinc-100 overflow-y-auto rounded border border-zinc-200">
        {!students && <li className="p-4 text-center text-sm text-zinc-400">Loading…</li>}
        {students && matches.length === 0 && <li className="p-4 text-center text-sm text-zinc-400">No students found</li>}
        {matches.map((s) => (
          <li key={s.id}>
            <button
              className="flex w-full items-center gap-3 px-3 py-2 text-left text-sm hover:bg-zinc-50"
              onClick={async () => {
                await apply(() => api.post<ScheduleData>(`/api/heats/${heat.id}/students`, { student_id: s.id }), `${s.first_name} added`);
                onClose();
              }}
            >
              <UserPlus className="size-4 text-zinc-400" />
              <span className="flex-1">
                {s.first_name} <strong>{s.last_name}</strong> <span className="text-zinc-400">· {s.age_group}</span>
              </span>
              <HouseBadge group={s.group_code} house={s.house} />
            </button>
          </li>
        ))}
      </ul>
    </Modal>
  );
}

function TimesCard({
  settings,
  heats,
  onApply,
  onClear,
  ask,
}: {
  settings: Settings;
  heats: Heat[];
  onApply: (start: string, end: string) => Promise<void>;
  onClear: () => Promise<void>;
  ask: (c: { title: string; body: string; label: string; run: () => Promise<void> }) => void;
}) {
  const [start, setStart] = useState(settings.day_start);
  const [end, setEnd] = useState(settings.day_end);
  const hasTimes = heats.some((h) => h.time);

  const minutes = (v: string) => {
    const [h, m] = v.split(":").map(Number);
    return h * 60 + m;
  };
  const span = start && end ? minutes(end) - minutes(start) : 0;
  const slot = span > 0 ? span / heats.length : 0;

  const run = () =>
    hasTimes
      ? ask({ title: "Replace the current times?", body: "Every heat gets a new time. Times you typed in by hand are replaced.", label: "Set times", run: () => onApply(start, end) })
      : onApply(start, end);

  return (
    <Card className="mb-6 p-5">
      <h2 className="font-semibold">Times</h2>
      <p className="mt-0.5 text-sm text-zinc-500">
        Pick when the day starts and when you want everything finished. Heats are spread evenly in between. You can still change any heat&apos;s time below.
      </p>
      <div className="mt-4 flex flex-wrap items-end gap-3">
        <label className="text-sm font-medium">
          <span className="mb-1.5 block">Start</span>
          <Input type="time" value={start} onChange={(e) => setStart(e.target.value)} className="!w-32" />
        </label>
        <label className="text-sm font-medium">
          <span className="mb-1.5 block">Finish</span>
          <Input type="time" value={end} onChange={(e) => setEnd(e.target.value)} className="!w-32" />
        </label>
        <Button variant="primary" disabled={span <= 0} onClick={run}>
          Set times automatically
        </Button>
        {hasTimes && (
          <Button onClick={() => ask({ title: "Clear all times?", body: "Every heat loses its time.", label: "Clear", run: onClear })}>Clear times</Button>
        )}
      </div>
      <p className="mt-3 text-sm text-zinc-500">
        {span > 0 ? `${heats.length} heats in ${Math.floor(span / 60)} h ${span % 60} min is about ${slot < 10 ? slot.toFixed(1) : Math.round(slot)} minutes per heat.` : "The finish time must be after the start time."}
      </p>
    </Card>
  );
}
