"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import useSWR from "swr";
import { Spinner } from "@/components/ui";
import { api, ApiError, type Role, type User } from "@/lib/api";

export function useUser() {
  const { data, isLoading, mutate } = useSWR<User | null>(
    "/api/me",
    async (url: string) => {
      try {
        return await api.get<User>(url);
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
    { revalidateOnFocus: false, shouldRetryOnError: false },
  );
  return { user: data ?? null, loading: isLoading, mutate };
}

export const homeFor = (role: Role) => (role === "organiser" ? "/organiser" : "/staff");

export async function signOut() {
  await api.post("/api/auth/logout");
  window.location.href = "/";
}

/** Renders children only for the given role; otherwise sends people where they belong. */
export function RequireRole({ role, children }: { role: Role; children: ReactNode }) {
  const { user, loading } = useUser();
  const router = useRouter();
  useEffect(() => {
    if (loading) return;
    if (!user) router.replace("/login");
    else if (user.role !== role) router.replace(homeFor(user.role));
  }, [user, loading, role, router]);
  if (loading || !user || user.role !== role) return <Spinner />;
  return <>{children}</>;
}
