"use client";

import { AlertCircle, Check, Download, Loader2, Search, UserPlus } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import useSWR from "swr";
import { EventPicker } from "@/components/EventPicker";
import { StudentModal } from "@/components/StudentModal";
import { Button, buttonClass, cx, Empty, HouseBadge, Input, Progress, Spinner, useToast } from "@/components/ui";
import { ageGroups, api, errorMessage, fetcher, type AthleticEvent, type Settings, type Student } from "@/lib/api";
import { RequireRole, useUser } from "@/lib/auth";

export default function StaffPage() {
  return (
    <RequireRole role="staff">
      <StaffDashboard />
    </RequireRole>
  );
}

type SaveState = "idle" | "saving" | "saved" | "error";

function StaffDashboard() {
  const { user } = useUser();
  const toast = useToast();
  const { data: students, mutate } = useSWR<Student[]>("/api/students", fetcher);
  const { data: events } = useSWR<AthleticEvent[]>("/api/events", fetcher);
  const { data: settings } = useSWR<Settings>("/api/settings", fetcher);
  const [query, setQuery] = useState("");
  const [onlyTodo, setOnlyTodo] = useState(false);
  const [adding, setAdding] = useState(false);
  const [save, setSave] = useState<SaveState>("idle");
  const pending = useRef(0);

  const min = settings?.min_events ?? 3;
  const ready = students?.filter((s) => s.events.length >= min).length ?? 0;

  async function toggle(st: Student, ev: AthleticEvent, on: boolean) {
    pending.current++;
    setSave("saving");
    mutate((cur) => cur?.map((s) => (s.id === st.id ? { ...s, events: on ? [...s.events, ev.id] : s.events.filter((x) => x !== ev.id) } : s)), { revalidate: false });
    try {
      await api.put(`/api/students/${st.id}/events/${ev.id}`, { on });
      if (--pending.current === 0) setSave("saved");
    } catch (e) {
      pending.current--;
      setSave("error");
      toast(`Couldn't save ${st.first_name}'s change: ${errorMessage(e)}`, "error");
      mutate();
    }
  }

  const grouped = useMemo(() => {
    const q = query.trim().toLowerCase();
    const list = (students ?? []).filter((s) => (!q || `${s.first_name} ${s.last_name}`.toLowerCase().includes(q)) && (!onlyTodo || s.events.length < min));
    return ageGroups.map((g) => ({ name: g, items: list.filter((s) => s.age_group === g) })).filter((g) => g.items.length > 0);
  }, [students, query, onlyTodo, min]);

  if (!user || !students || !events) return <Spinner />;

  if (!user.group_code)
    return (
      <Empty title="Your account isn't linked to a group yet">Ask the organiser to add your group (for example SSH) to your account.</Empty>
    );

  return (
    <>
      <section className="mb-5 grid items-end gap-8 md:grid-cols-[auto_1fr]">
        <div>
          <p className="mb-3 flex items-center gap-2 font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-zinc-500">
            Your group <HouseBadge group={user.house || "No house"} house={user.house} />
          </p>
          <h1 className="text-[clamp(4.5rem,15vw,8.5rem)] font-light leading-[0.9] text-signal">{user.group_code}</h1>
        </div>
        <div className="md:justify-self-end md:text-right">
          <p className="text-6xl font-light leading-none tabular-nums text-ink sm:text-7xl">
            {ready}
            <span className="text-zinc-300">/{students.length}</span>
          </p>
          <p className="mt-2 font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-zinc-500">students ready</p>
        </div>
      </section>
      <div className="mb-8">
        <Progress value={ready} max={students.length} />
      </div>

      <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
        <p className="max-w-xl text-[15px] text-zinc-600">
          Hello {user.name}. Tap the events each student will do. Every student needs at least <strong className="text-ink">{min}</strong>. Changes save automatically.
        </p>
        <div className="flex flex-wrap items-center gap-3">
          <SaveBadge state={save} />
          <a href="/api/export/group.csv" className={buttonClass("secondary")}>
            <Download className="size-4" /> Download CSV
          </a>
          <Button variant="primary" onClick={() => setAdding(true)}>
            <UserPlus className="size-4" /> Add a student
          </Button>
        </div>
      </div>

      {students.length === 0 ? (
        <Empty title="No students in your group yet">
          Ask the organiser to import them, or add them yourself with <strong>Add a student</strong>.
        </Empty>
      ) : (
        <>
          <div className="mb-8 flex flex-wrap items-center gap-x-6 gap-y-3">
            <div className="relative w-full sm:w-72">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
              <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Find a student" className="pl-9" aria-label="Find a student" />
            </div>
            <label className="flex items-center gap-2 text-sm text-zinc-600">
              <input type="checkbox" checked={onlyTodo} onChange={(e) => setOnlyTodo(e.target.checked)} className="size-4 accent-ink" />
              Only show students who need more events
            </label>
          </div>

          {grouped.length === 0 && <Empty title="Nobody to show">{onlyTodo ? "Everyone has enough events. Nice work!" : "No student matches your search."}</Empty>}

          <div className="space-y-12">
            {grouped.map((g) => (
              <section key={g.name}>
                <h2 className="flex items-baseline justify-between border-b border-signal pb-2">
                  <span className="text-3xl font-light text-signal">{g.name}</span>
                  <span className="font-mono text-xs tabular-nums text-zinc-500">{g.items.length} students</span>
                </h2>
                <div className="divide-y divide-zinc-200">
                  {g.items.map((s) => {
                    const done = s.events.length >= min;
                    return (
                      <div key={s.id} className="grid gap-3 py-5 lg:grid-cols-[240px_1fr_64px] lg:items-center lg:gap-6">
                        <div>
                          <p className="text-xl font-normal leading-tight text-ink">
                            {s.first_name} {s.last_name}
                          </p>
                          <p className="mt-1 font-mono text-xs text-zinc-500">
                            AGE {s.age}
                            {s.added_by_staff && " · ADDED BY STAFF"}
                          </p>
                        </div>
                        <EventPicker events={events} selected={s.events} onToggle={(ev, on) => toggle(s, ev, on)} compact />
                        <p className={cx("text-3xl font-light tabular-nums lg:text-right", done ? "text-ink" : "text-signal")} aria-label={`${s.events.length} of ${min} events`}>
                          {done ? <Check className="inline size-7" strokeWidth={3} /> : `${s.events.length}/${min}`}
                        </p>
                      </div>
                    );
                  })}
                </div>
              </section>
            ))}
          </div>
        </>
      )}

      {adding && <StudentModal fixedGroup={user.group_code} onClose={() => setAdding(false)} onChanged={() => mutate()} />}
    </>
  );
}

function SaveBadge({ state }: { state: SaveState }) {
  const map = {
    idle: { icon: <Check className="size-4" />, text: "Autosave on", cls: "text-zinc-500" },
    saving: { icon: <Loader2 className="size-4 animate-spin" />, text: "Saving…", cls: "text-zinc-500" },
    saved: { icon: <Check className="size-4" />, text: "All changes saved", cls: "text-emerald-600" },
    error: { icon: <AlertCircle className="size-4" />, text: "Couldn't save. Try again", cls: "text-red-600" },
  }[state];
  return (
    <span className={cx("inline-flex items-center gap-1.5 font-mono text-[11px] font-semibold uppercase tracking-wider", map.cls)} aria-live="polite">
      {map.icon} {map.text}
    </span>
  );
}
