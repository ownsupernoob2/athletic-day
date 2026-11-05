"use client";

import { KeyRound, Pencil, Trash2, UserPlus, Wand2 } from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import { Button, Card, Confirm, ErrorNote, Field, HouseBadge, Input, Modal, PageTitle, Pill, Select, Spinner, useToast } from "@/components/ui";
import { api, errorMessage, fetcher, type StaffUser } from "@/lib/api";
import { useUser } from "@/lib/auth";

export default function StaffPage() {
  const toast = useToast();
  const { user: me } = useUser();
  const { data: users, mutate } = useSWR<StaffUser[]>("/api/staff", fetcher);
  const [editing, setEditing] = useState<StaffUser | "new" | null>(null);
  const [deleting, setDeleting] = useState<StaffUser | null>(null);

  if (!users) return <Spinner />;

  return (
    <>
      <PageTitle
        title="Staff accounts"
        subtitle="Each staff member signs in and sees only their own group."
        actions={
          <Button variant="primary" onClick={() => setEditing("new")}>
            <UserPlus className="size-4" /> Add staff
          </Button>
        }
      />
      <Card className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead className="border-b border-zinc-100 text-xs uppercase tracking-wider text-zinc-400">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Group</th>
              <th className="w-24 px-4 py-3" />
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {users.map((u) => (
              <tr key={u.id} className="hover:bg-zinc-50">
                <td className="px-4 py-2.5 font-medium">
                  {u.name} {u.id === me?.id && <span className="ml-1 text-xs font-normal text-zinc-400">(you)</span>}
                </td>
                <td className="px-4 py-2.5 text-zinc-600">{u.email}</td>
                <td className="px-4 py-2.5">{u.role === "organiser" ? <Pill tone="indigo">Organiser</Pill> : <HouseBadge group={u.group_code} house={u.house} />}</td>
                <td className="px-4 py-2.5">
                  <div className="flex justify-end gap-1">
                    <button onClick={() => setEditing(u)} className="rounded-md p-2 text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900" aria-label={`Edit ${u.name}`}>
                      <Pencil className="size-4" />
                    </button>
                    {u.id !== me?.id && (
                      <button onClick={() => setDeleting(u)} className="rounded-md p-2 text-zinc-500 hover:bg-red-50 hover:text-red-600" aria-label={`Delete ${u.name}`}>
                        <Trash2 className="size-4" />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

      {editing && <StaffModal user={editing === "new" ? undefined : editing} onClose={() => setEditing(null)} onDone={() => mutate()} />}
      {deleting && (
        <Confirm
          danger
          title={`Delete ${deleting.name}?`}
          confirmLabel="Delete account"
          body="They won't be able to sign in any more. Their group's students and entries stay."
          onClose={() => setDeleting(null)}
          onConfirm={async () => {
            try {
              await api.del(`/api/staff/${deleting.id}`);
              toast("Account deleted");
              mutate();
            } catch (e) {
              toast(errorMessage(e), "error");
            }
          }}
        />
      )}
    </>
  );
}

function randomPassword() {
  const chars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789";
  const bytes = crypto.getRandomValues(new Uint8Array(10));
  return Array.from(bytes, (b) => chars[b % chars.length]).join("");
}

function StaffModal({ user, onClose, onDone }: { user?: StaffUser; onClose: () => void; onDone: () => void }) {
  const toast = useToast();
  const [name, setName] = useState(user?.name ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [group, setGroup] = useState(user?.group_code ?? "");
  const [role, setRole] = useState(user?.role ?? "staff");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const body = { name, email, group_code: group, role, password };
    try {
      if (user) await api.patch(`/api/staff/${user.id}`, body);
      else await api.post("/api/staff", body);
      toast(user ? "Account updated" : "Account created. Share the password with them.");
      onDone();
      onClose();
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  }

  return (
    <Modal title={user ? "Edit account" : "Add staff"} onClose={onClose}>
      <form onSubmit={save} className="space-y-4">
        <Field label="Name">{(id) => <Input id={id} required value={name} onChange={(e) => setName(e.target.value)} placeholder="Ms. Adams" />}</Field>
        <Field label="Email">{(id) => <Input id={id} type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />}</Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Role">
            {(id) => (
              <Select id={id} value={role} onChange={(e) => setRole(e.target.value as "staff" | "organiser")}>
                <option value="staff">Staff</option>
                <option value="organiser">Organiser</option>
              </Select>
            )}
          </Field>
          {role === "staff" && (
            <Field label="Group" hint="3 letters, e.g. SSH">
              {(id) => <Input id={id} required maxLength={3} value={group} onChange={(e) => setGroup(e.target.value.toUpperCase().replace(/[^A-Z]/g, ""))} />}
            </Field>
          )}
        </div>
        <Field label={user ? "New password" : "Password"} hint={user ? "Leave empty to keep the current one" : "At least 8 characters. Share it with them."}>
          {(id) => (
            <div className="flex gap-2">
              <Input id={id} required={!user} minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" className="font-mono" />
              <Button type="button" onClick={() => setPassword(randomPassword())} aria-label="Generate a password" title="Generate a password">
                {user ? <KeyRound className="size-4" /> : <Wand2 className="size-4" />}
              </Button>
            </div>
          )}
        </Field>
        {error && <ErrorNote>{error}</ErrorNote>}
        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={busy}>
            {user ? "Save" : "Create account"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
