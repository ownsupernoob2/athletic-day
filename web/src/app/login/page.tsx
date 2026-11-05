"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Button, ErrorNote, Field, Input } from "@/components/ui";
import { api, errorMessage, type User } from "@/lib/api";
import { homeFor, useUser } from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const { user, mutate } = useUser();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (user) router.replace(homeFor(user.role));
  }, [user, router]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const u = await api.post<User>("/api/auth/login", { email, password });
      await mutate(u, false);
      router.replace(homeFor(u.role));
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-sm py-10 sm:py-16">
      <h1 className="border-b border-signal pb-3 text-4xl font-light text-signal">Sign in</h1>
      <p className="mt-4 text-sm text-zinc-500">For staff and organisers. Your organiser gave you your email and password.</p>
      <form onSubmit={submit} className="mt-8 space-y-4">
        <Field label="Email">
          {(id) => <Input id={id} type="email" autoComplete="username" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@school.org" />}
        </Field>
        <Field label="Password">
          {(id) => <Input id={id} type="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />}
        </Field>
        {error && <ErrorNote>{error}</ErrorNote>}
        <Button type="submit" variant="primary" loading={busy} className="w-full">
          Sign in
        </Button>
      </form>
      <p className="mt-6 text-center text-sm text-zinc-400">Forgot your password? Ask the organiser to reset it.</p>
    </div>
  );
}
