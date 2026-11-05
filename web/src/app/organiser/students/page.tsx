"use client";

import { Download, Pencil, Search, Trash2, Upload, UserPlus } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";
import useSWR from "swr";
import { StudentModal } from "@/components/StudentModal";
import { Button, Card, Confirm, Empty, ErrorNote, HouseBadge, Input, Modal, PageTitle, Pill, Select, Spinner, useToast } from "@/components/ui";
import { ageGroups, api, errorMessage, fetcher, houseNames, timeAgo, type AthleticEvent, type KamarStatus, type Settings, type Student } from "@/lib/api";

export default function StudentsPage() {
  return (
    <Suspense fallback={<Spinner />}>
      <Students />
    </Suspense>
  );
}

const PAGE = 100;

function Students() {
  const toast = useToast();
  const params = useSearchParams();
  const { data: students, mutate } = useSWR<Student[]>("/api/students", fetcher);
  const { data: events } = useSWR<AthleticEvent[]>("/api/events", fetcher);
  const { data: settings } = useSWR<Settings>("/api/settings", fetcher);
  const { data: kamar } = useSWR<KamarStatus>("/api/kamar/status", fetcher, { refreshInterval: 30000 });

  const [query, setQuery] = useState(params.get("group") ?? "");
  const [house, setHouse] = useState("");
  const [age, setAge] = useState("");
  const [event, setEvent] = useState("");
  const [onlyTodo, setOnlyTodo] = useState(false);
  const [limit, setLimit] = useState(PAGE);
  const [editing, setEditing] = useState<Student | "new" | null>(null);
  const [deleting, setDeleting] = useState<Student | null>(null);
  const [importing, setImporting] = useState(false);
  const [wiping, setWiping] = useState(false);

  const evById = useMemo(() => new Map((events ?? []).map((e) => [e.id, e])), [events]);
  const min = settings?.min_events ?? 3;

  const filtered = useMemo(() => {
    const tokens = query.toLowerCase().split(/\s+/).filter(Boolean);
    return (students ?? []).filter((s) => {
      if (house && s.house !== house) return false;
      if (age && s.age_group !== age) return false;
      if (event && !s.events.includes(Number(event))) return false;
      if (onlyTodo && s.events.length >= min) return false;
      if (!tokens.length) return true;
      const hay = `${s.first_name} ${s.last_name} ${s.group_code} ${s.house} age ${s.age} ${s.age_group} ${s.events.map((id) => `${evById.get(id)?.code} ${evById.get(id)?.name}`).join(" ")}`.toLowerCase();
      return tokens.every((t) => hay.includes(t));
    });
  }, [students, query, house, age, event, onlyTodo, min, evById]);

  if (!students || !events || !settings) return <Spinner />;

  const anyFilter = query || house || age || event || onlyTodo;

  return (
    <>
      <PageTitle
        title="Students"
        subtitle={
          <>
            {students.length} students. Search by name, group, age, house or event.
            <span className="mt-1 block text-zinc-500">
              {kamar?.last
                ? `KAMAR: ${kamar.linked} students linked, last update ${timeAgo(kamar.last.at)}.`
                : kamar?.configured
                  ? "KAMAR: waiting for the first update from the school."
                  : "KAMAR: not connected yet."}
            </span>
          </>
        }
        actions={
          <>
            <Button onClick={() => setImporting(true)}>
              <Upload className="size-4" /> Import CSV
            </Button>
            <Button variant="primary" onClick={() => setEditing("new")}>
              <UserPlus className="size-4" /> Add student
            </Button>
          </>
        }
      />

      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-[1fr_140px_140px_160px]">
        <div className="relative sm:col-span-2 lg:col-span-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
          <Input
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setLimit(PAGE);
            }}
            placeholder="Search e.g. “khan”, “SSH”, “14”, “javelin”"
            className="pl-9"
            aria-label="Search students"
          />
        </div>
        <Select value={house} onChange={(e) => setHouse(e.target.value)} aria-label="House">
          <option value="">All houses</option>
          {houseNames.map((h) => (
            <option key={h}>{h}</option>
          ))}
        </Select>
        <Select value={age} onChange={(e) => setAge(e.target.value)} aria-label="Age group">
          <option value="">All ages</option>
          {ageGroups.map((a) => (
            <option key={a}>{a}</option>
          ))}
        </Select>
        <Select value={event} onChange={(e) => setEvent(e.target.value)} aria-label="Event">
          <option value="">Any event</option>
          {events.map((e) => (
            <option key={e.id} value={e.id}>
              {e.name}
            </option>
          ))}
        </Select>
      </div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3 text-sm">
        <label className="flex items-center gap-2 text-zinc-600">
          <input type="checkbox" checked={onlyTodo} onChange={(e) => setOnlyTodo(e.target.checked)} className="size-4 rounded border-zinc-300 accent-indigo-600" />
          Only students with fewer than {min} events
        </label>
        <span className="text-zinc-500">
          {filtered.length} shown
          {anyFilter && (
            <button
              className="ml-3 font-medium text-indigo-600 hover:underline"
              onClick={() => {
                setQuery("");
                setHouse("");
                setAge("");
                setEvent("");
                setOnlyTodo(false);
              }}
            >
              Clear filters
            </button>
          )}
        </span>
      </div>

      {students.length === 0 ? (
        <Empty title="No students yet">
          Click <strong>Import CSV</strong> to load your school&apos;s list, or add students one at a time.
        </Empty>
      ) : filtered.length === 0 ? (
        <Empty title="No students match">Try fewer words or clear the filters.</Empty>
      ) : (
        <Card className="overflow-x-auto">
          <table className="w-full min-w-[720px] text-left text-sm">
            <thead className="border-b border-zinc-100 text-xs uppercase tracking-wider text-zinc-400">
              <tr>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Age</th>
                <th className="px-4 py-3 font-medium">Group</th>
                <th className="px-4 py-3 font-medium">Events</th>
                <th className="w-24 px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {filtered.slice(0, limit).map((s) => (
                <tr key={s.id} className="hover:bg-zinc-50">
                  <td className="px-4 py-2.5 font-medium">
                    {s.last_name}, {s.first_name}
                    {s.added_by_staff && <span className="ml-2 text-xs font-normal text-zinc-400">added by staff</span>}
                  </td>
                  <td className="px-4 py-2.5 text-zinc-600">
                    {s.age} <span className="text-zinc-400">· {s.age_group}</span>
                  </td>
                  <td className="px-4 py-2.5">
                    <HouseBadge group={s.group_code} house={s.house} />
                  </td>
                  <td className="px-4 py-2.5">
                    {s.events.length === 0 ? (
                      <Pill tone="amber">None yet</Pill>
                    ) : (
                      <div className="flex flex-wrap items-center gap-1">
                        {s.events.map((id) => (
                          <span key={id} className="rounded-md bg-zinc-100 px-1.5 py-0.5 text-xs text-zinc-700">
                            {evById.get(id)?.code}
                          </span>
                        ))}
                        {s.events.length < min && <Pill tone="amber">needs {min - s.events.length} more</Pill>}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="flex justify-end gap-1">
                      <button onClick={() => setEditing(s)} className="rounded-md p-2 text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900" aria-label={`Edit ${s.first_name} ${s.last_name}`}>
                        <Pencil className="size-4" />
                      </button>
                      <button onClick={() => setDeleting(s)} className="rounded-md p-2 text-zinc-500 hover:bg-red-50 hover:text-red-600" aria-label={`Remove ${s.first_name} ${s.last_name}`}>
                        <Trash2 className="size-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
      {filtered.length > limit && (
        <div className="mt-4 text-center">
          <Button onClick={() => setLimit((l) => l + PAGE)}>Show {Math.min(PAGE, filtered.length - limit)} more</Button>
        </div>
      )}

      {students.length > 0 && (
        <div className="mt-12 flex flex-wrap items-center justify-between gap-3 rounded-md border border-zinc-200 p-4 text-sm">
          <span className="text-zinc-500">Starting a new year? Remove everyone and import a fresh list.</span>
          <Button variant="danger" size="sm" onClick={() => setWiping(true)}>
            Remove all students
          </Button>
        </div>
      )}

      {editing && (
        <StudentModal student={editing === "new" ? undefined : editing} events={events.filter((e) => e.active)} onClose={() => setEditing(null)} onChanged={() => mutate()} />
      )}
      {deleting && (
        <Confirm
          danger
          title="Remove this student?"
          confirmLabel="Remove"
          body={
            <>
              <strong>
                {deleting.first_name} {deleting.last_name}
              </strong>{" "}
              will be removed from the list and from any heats.
            </>
          }
          onClose={() => setDeleting(null)}
          onConfirm={async () => {
            try {
              await api.del(`/api/students/${deleting.id}`);
              toast("Student removed");
              mutate();
            } catch (e) {
              toast(errorMessage(e), "error");
            }
          }}
        />
      )}
      {importing && <ImportModal onClose={() => setImporting(false)} onDone={() => mutate()} />}
      {wiping && <WipeModal onClose={() => setWiping(false)} onDone={() => mutate()} />}
    </>
  );
}

function ImportModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<{ added: number; skipped: number; errors: string[] } | null>(null);

  async function run() {
    if (!file) return;
    setBusy(true);
    setError("");
    const form = new FormData();
    form.append("file", file);
    try {
      setResult(await api.upload("/api/students/import", form));
      onDone();
    } catch (e) {
      setError(errorMessage(e));
    }
    setBusy(false);
  }

  function downloadTemplate() {
    const csv = "First Name,Last Name,DOB,Group ID\nJames,O'Connor,12/05/2010,SYP\nAmir,Khan,03/08/2011,SSH\n";
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
    a.download = "students-template.csv";
    a.click();
  }

  return (
    <Modal title="Import students" onClose={onClose}>
      {result ? (
        <div className="space-y-4">
          <p className="text-sm">
            <strong>{result.added}</strong> students added{result.skipped > 0 && <>, {result.skipped} skipped</>}.
          </p>
          {result.errors.length > 0 && (
            <div className="max-h-40 overflow-y-auto rounded bg-amber-50 p-3 text-xs text-amber-900">
              {result.errors.map((e) => (
                <p key={e}>{e}</p>
              ))}
            </div>
          )}
          <div className="flex justify-end">
            <Button variant="primary" onClick={onClose}>
              Done
            </Button>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-sm text-zinc-600">
            Choose a CSV with these columns: <strong>First Name, Last Name, DOB, Group ID</strong>. Dates look like 31/12/2010. Students already in the list are skipped.
          </p>
          <button onClick={downloadTemplate} className="inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:underline">
            <Download className="size-4" /> Download an example file
          </button>
          <input
            type="file"
            accept=".csv,text/csv"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            className="block w-full text-sm file:mr-3 file:rounded file:border-0 file:bg-zinc-100 file:px-4 file:py-2 file:text-sm file:font-medium hover:file:bg-zinc-200"
          />
          {error && <ErrorNote>{error}</ErrorNote>}
          <div className="flex justify-end gap-2">
            <Button onClick={onClose}>Cancel</Button>
            <Button variant="primary" disabled={!file} loading={busy} onClick={run}>
              Import
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
}

function WipeModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const toast = useToast();
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <Modal title="Remove all students?" onClose={onClose}>
      <p className="text-sm text-zinc-600">This deletes every student, their events and the schedule&apos;s heats. It can&apos;t be undone. Download the entries first if you need them.</p>
      <div className="mt-4">
        <label className="mb-1.5 block text-sm font-medium">
          Type <code className="rounded bg-zinc-100 px-1">DELETE</code> to confirm
        </label>
        <Input value={text} onChange={(e) => setText(e.target.value)} autoFocus />
      </div>
      <div className="mt-6 flex justify-end gap-2">
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="danger"
          disabled={text !== "DELETE"}
          loading={busy}
          onClick={async () => {
            setBusy(true);
            try {
              await api.post("/api/students/clear", { confirm: text });
              toast("All students removed");
              onDone();
              onClose();
            } catch (e) {
              toast(errorMessage(e), "error");
              setBusy(false);
            }
          }}
        >
          Remove everyone
        </Button>
      </div>
    </Modal>
  );
}
