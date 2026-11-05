"use client";

import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import { Button, Card, Confirm, cx, ErrorNote, Field, Input, PageTitle, Pill, Select, Spinner, useToast } from "@/components/ui";
import { api, errorMessage, fetcher, type AthleticEvent, type Settings } from "@/lib/api";

export default function EventsPage() {
  const toast = useToast();
  const { data: events, mutate } = useSWR<AthleticEvent[]>("/api/events", fetcher);
  const { data: settings, mutate: mutateSettings } = useSWR<Settings>("/api/settings", fetcher);
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [kind, setKind] = useState("track");
  const [error, setError] = useState("");
  const [deleting, setDeleting] = useState<AthleticEvent | null>(null);

  if (!events || !settings) return <Spinner />;

  async function act(fn: () => Promise<unknown>) {
    try {
      await fn();
      await mutate();
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  }

  async function add(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api.post("/api/events", { name, code, kind });
      setName("");
      setCode("");
      toast("Event added");
      mutate();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <>
      <PageTitle title="Events & rules" subtitle="Choose which events are on offer, and set the rules for the day." />

      <SettingsCard settings={settings} onSaved={() => mutateSettings()} />

      <h2 className="mb-3 mt-10 text-lg font-semibold">Events</h2>
      <Card className="divide-y divide-zinc-100">
        {events.map((ev, i) => (
          <div key={ev.id} className={cx("flex flex-wrap items-center gap-3 px-4 py-3", !ev.active && "bg-zinc-50")}>
            <div className="flex">
              <button disabled={i === 0} onClick={() => act(() => api.patch(`/api/events/${ev.id}`, { move_up: true }))} className="rounded p-1.5 text-zinc-400 hover:bg-zinc-100 disabled:opacity-30" aria-label={`Move ${ev.name} up`}>
                <ArrowUp className="size-4" />
              </button>
              <button disabled={i === events.length - 1} onClick={() => act(() => api.patch(`/api/events/${ev.id}`, { move_up: false }))} className="rounded p-1.5 text-zinc-400 hover:bg-zinc-100 disabled:opacity-30" aria-label={`Move ${ev.name} down`}>
                <ArrowDown className="size-4" />
              </button>
            </div>
            <div className={cx("min-w-0 flex-1", !ev.active && "opacity-60")}>
              <p className="font-medium">
                {ev.name} <span className="ml-1 rounded-md bg-zinc-100 px-1.5 py-0.5 text-xs font-normal text-zinc-600">{ev.code}</span>
              </p>
              <p className="text-xs text-zinc-500">{ev.entry_count} entries</p>
            </div>
            <Pill tone={ev.kind === "track" ? "indigo" : "gray"}>{ev.kind === "track" ? "Track" : "Field"}</Pill>
            <label className="flex items-center gap-2 text-sm text-zinc-600">
              <input type="checkbox" checked={ev.active} onChange={(e) => act(() => api.patch(`/api/events/${ev.id}`, { active: e.target.checked }))} className="size-4 rounded border-zinc-300 accent-indigo-600" />
              On offer
            </label>
            <button onClick={() => setDeleting(ev)} className="rounded-md p-2 text-zinc-400 hover:bg-red-50 hover:text-red-600" aria-label={`Delete ${ev.name}`}>
              <Trash2 className="size-4" />
            </button>
          </div>
        ))}
      </Card>
      <p className="mt-2 text-xs text-zinc-500">Unticking &ldquo;On offer&rdquo; hides an event from staff and the CSV but keeps existing entries.</p>

      <h2 className="mb-3 mt-10 text-lg font-semibold">Add an event</h2>
      <Card className="p-5">
        <form onSubmit={add} className="grid gap-4 sm:grid-cols-[1fr_140px_140px_auto] sm:items-end">
          <Field label="Name">{(id) => <Input id={id} required value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Relay 4x100m" />}</Field>
          <Field label="Short name" hint="Shown on buttons">
            {(id) => <Input id={id} value={code} onChange={(e) => setCode(e.target.value)} placeholder="e.g. 4x100" maxLength={12} />}
          </Field>
          <Field label="Type">
            {(id) => (
              <Select id={id} value={kind} onChange={(e) => setKind(e.target.value)}>
                <option value="track">Track</option>
                <option value="field">Field</option>
              </Select>
            )}
          </Field>
          <Button type="submit" variant="primary" className="sm:mb-[22px]">
            <Plus className="size-4" /> Add
          </Button>
        </form>
        {error && <div className="mt-3"><ErrorNote>{error}</ErrorNote></div>}
      </Card>

      {deleting && (
        <Confirm
          danger
          title={`Delete ${deleting.name}?`}
          confirmLabel="Delete event"
          body={
            deleting.entry_count > 0 ? (
              <>
                <strong>{deleting.entry_count} students</strong> have picked this event. Deleting it removes those entries and any heats. If you only want to hide it, untick &ldquo;On offer&rdquo; instead.
              </>
            ) : (
              "Nobody has picked this event yet."
            )
          }
          onClose={() => setDeleting(null)}
          onConfirm={() => act(() => api.del(`/api/events/${deleting.id}`))}
        />
      )}
    </>
  );
}

function SettingsCard({ settings, onSaved }: { settings: Settings; onSaved: () => void }) {
  const toast = useToast();
  const [title, setTitle] = useState(settings.title);
  const [date, setDate] = useState(settings.event_date);
  const [min, setMin] = useState(String(settings.min_events));
  const [heat, setHeat] = useState(String(settings.heat_size));
  const [busy, setBusy] = useState(false);
  const dirty = title !== settings.title || date !== settings.event_date || min !== String(settings.min_events) || heat !== String(settings.heat_size);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.put("/api/settings", { title, event_date: date, min_events: Number(min), heat_size: Number(heat) });
      toast("Settings saved");
      onSaved();
    } catch (err) {
      toast(errorMessage(err), "error");
    }
    setBusy(false);
  }

  return (
    <Card className="p-5">
      <form onSubmit={save} className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="sm:col-span-2">
          <Field label="Title">{(id) => <Input id={id} value={title} onChange={(e) => setTitle(e.target.value)} />}</Field>
        </div>
        <Field label="Athletics day" hint="Ages are worked out on this date">
          {(id) => <Input id={id} type="date" value={date} onChange={(e) => setDate(e.target.value)} />}
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Min. events" hint="Per student">
            {(id) => <Input id={id} type="number" min={0} max={20} value={min} onChange={(e) => setMin(e.target.value)} />}
          </Field>
          <Field label="Heat size" hint="Max runners">
            {(id) => <Input id={id} type="number" min={1} max={20} value={heat} onChange={(e) => setHeat(e.target.value)} />}
          </Field>
        </div>
        <div className="sm:col-span-2 lg:col-span-4">
          <Button type="submit" variant="primary" disabled={!dirty} loading={busy}>
            Save changes
          </Button>
        </div>
      </form>
    </Card>
  );
}
