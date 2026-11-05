"use client";

import { LogOut } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { buttonClass, cx } from "@/components/ui";
import { homeFor, signOut, useUser } from "@/lib/auth";

export function Header() {
  const { user } = useUser();
  const path = usePathname();
  const onSchedule = path.startsWith("/schedule");

  return (
    <header className="border-b border-zinc-100 print:hidden">
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
        <Link href={user ? homeFor(user.role) : "/"} className="flex items-center gap-3" aria-label="King's High School athletics home">
          <Image src="/logo.png" alt="" width={474} height={471} priority className="h-12 w-auto" />
          <span className="text-[22px] font-light leading-[1.05] text-ink">
            King&apos;s
            <br />
            High School
          </span>
        </Link>

        <nav className="flex items-center gap-4 sm:gap-6" aria-label="Main">
          <Link href="/schedule" className={cx("text-[15px] hover:text-signal", onSchedule ? "text-signal underline decoration-1 underline-offset-8" : "text-ink")}>
            {user?.role === "organiser" ? "Public schedule" : "Schedule"}
          </Link>
          {user ? (
            <>
              <span className="hidden text-[15px] text-zinc-500 md:inline">{user.name}</span>
              <button onClick={signOut} className={buttonClass("secondary", "sm")}>
                <LogOut className="size-4" />
                <span className="hidden sm:inline">Sign out</span>
              </button>
            </>
          ) : (
            <Link href="/login" className={buttonClass("primary", "sm")}>
              Sign in
            </Link>
          )}
        </nav>
      </div>
    </header>
  );
}
