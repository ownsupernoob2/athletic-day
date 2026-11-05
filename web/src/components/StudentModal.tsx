"use client";

import { useState } from "react";
import { EventPicker } from "@/components/EventPicker";
import { Button, ErrorNote, Field, Input, Modal, useToast } from "@/components/ui";
import { api, errorMessage, type AthleticEvent, type Student } from "@/lib/api";

/**
 * Add a student, or edit one. Staff can only add (their group is fixed);
 * organisers can edit everything, including the student's events.
 */
export function StudentModal({
  student,
  fixedGroup,
  events,
  onClose,
  onChanged,
}: {
  student?: Student;
  fixedGroup?: string;
  events?: AthleticEvent[];
  onClose: () => void;
  onChanged: () => void;
}) {
  const toast = useToast();
  const [first, setFirst] = useState(student?.first_name ?? "");
  const [last, setLast] = useState(student?.last_name ?? "");
  const [dob, setDob] = useState(student?.dob ?? "");
  const [group, setGroup] = useState(student?.group_code ?? fixedGroup ?? "");
  const [selected, setSelected] = useState<number[]>(student?.events ?? []);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const body = { first_name: first, last_name: last, dob, group_code: group };
    try {
      if (student) await api.patch(`/api/students/${student.id}`, body);
      else await api.post("/api/students", body);
      toast(student ? "Student updated" : `${first} ${last} added`);
      onChanged();
      onClose();
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  }

  async function toggle(ev: AthleticEvent, on: boolean) {
    if (!student) return;
    setSelected((s) => (on ? [...s, ev.id] : s.filter((x) => x !== ev.id)));
    try {
      await api.put(`/api/students/${student.id}/events/${ev.id}`, { on });
      onChanged();
    } catch (err) {
      setSelected((s) => (on ? s.filter((x) => x !== ev.id) : [...s, ev.id]));
      toast(errorMessage(err), "error");
    }
  }

  return (
    <Modal title={student ? "Edit student" : "Add a student"} onClose={onClose} wide={!!student && !!events}>
      {student?.from_kamar && <p className="mb-4 rounded bg-indigo-50 px-3 py-2 text-sm text-ink">This student comes from KAMAR. Changes to their details here are replaced the next time KAMAR sends an update. Their events are not affected.</p>}
      <form onSubmit={save} className="space-y-4">
        <div className="grid grid-cols-2 gap-3">
          <Field label="First name">{(id) => <Input id={id} required value={first} onChange={(e) => setFirst(e.target.value)} />}</Field>
          <Field label="Last name">{(id) => <Input id={id} required value={last} onChange={(e) => setLast(e.target.value)} />}</Field>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Date of birth">{(id) => <Input id={id} type="date" required value={dob} onChange={(e) => setDob(e.target.value)} />}</Field>
          <Field label="Group" hint={fixedGroup ? "Your group" : "3 letters, e.g. SSH"}>
            {(id) => (
              <Input
                id={id}
                required
                maxLength={3}
                value={group}
                disabled={!!fixedGroup}
                onChange={(e) => setGroup(e.target.value.toUpperCase().replace(/[^A-Z]/g, ""))}
                className="uppercase"
              />
            )}
          </Field>
        </div>
        {error && <ErrorNote>{error}</ErrorNote>}
        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" onClick={onClose}>
            {student ? "Close" : "Cancel"}
          </Button>
          <Button type="submit" variant="primary" loading={busy}>
            {student ? "Save details" : "Add student"}
          </Button>
        </div>
      </form>

      {student && events && (
        <div className="mt-6 border-t border-zinc-100 pt-5">
          <p className="mb-3 text-sm font-medium text-zinc-800">Events (saved as you tap)</p>
          <EventPicker events={events} selected={selected} onToggle={toggle} />
        </div>
      )}
    </Modal>
  );
}
