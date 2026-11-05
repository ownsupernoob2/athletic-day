"use client";

import { ArrowRight } from "lucide-react";
import Link from "next/link";
import useSWR from "swr";
import { fetcher, formatDate, type PublicSchedule } from "@/lib/api";
import { homeFor, useUser } from "@/lib/auth";

export default function Home() {
  const { user } = useUser();
  const { data } = useSWR<PublicSchedule>("/api/public/schedule", fetcher);

  const links = [
    { href: "/schedule", title: "View the schedule", text: "Every event and heat. Find a student, download it or print the booklet." },
    user
      ? { href: homeFor(user.role), title: "Go to your dashboard", text: `Signed in as ${user.name}.` }
      : { href: "/login", title: "Staff & organisers sign in", text: "Choose events for your group, or manage the whole day." },
  ];

  return (
    <div className="max-w-2xl py-8 sm:py-14">
      <p className="text-[15px] text-zinc-500">{data?.event_date ? formatDate(data.event_date) : "Welcome"}</p>
      <h1 className="mt-2 text-5xl font-light leading-[1.05] text-signal sm:text-6xl">{data?.title ?? "Athletics Day"}</h1>
      <p className="mt-5 text-lg font-light text-zinc-600">Who is running what, and when. Pick what you need below.</p>

      <nav className="mt-12 border-t border-signal" aria-label="Start here">
        {links.map((l) => (
          <Link key={l.href} href={l.href} className="group flex items-center justify-between gap-6 border-b border-signal py-5">
            <span>
              <span className="block text-2xl font-light text-signal group-hover:text-ink">{l.title}</span>
              <span className="mt-1 block text-[15px] text-zinc-500">{l.text}</span>
            </span>
            <ArrowRight className="size-5 shrink-0 text-signal transition group-hover:translate-x-1 group-hover:text-ink" strokeWidth={1.5} />
          </Link>
        ))}
      </nav>

      {!user && <p className="mt-6 text-sm text-zinc-400">Students and parents don&apos;t need an account. Just open the schedule.</p>}
    </div>
  );
}
