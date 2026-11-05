"use client";

import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { cx } from "@/components/ui";
import { RequireRole } from "@/lib/auth";

const tabs = [
  { href: "/organiser", label: "Overview", exact: true },
  { href: "/organiser/students", label: "Students" },
  { href: "/organiser/events", label: "Events & rules" },
  { href: "/organiser/schedule", label: "Schedule" },
  { href: "/organiser/staff", label: "Staff accounts" },
];

/** Like the school website: a section title with a list of links divided by thin blue lines. */
export default function OrganiserLayout({ children }: { children: ReactNode }) {
  const path = usePathname();
  return (
    <RequireRole role="organiser">
      <div className="grid gap-8 lg:grid-cols-[250px_1fr] lg:gap-14">
        <aside className="print:hidden">
          <h2 className="hidden border-b border-signal pb-3 text-[32px] font-light leading-tight text-signal lg:block">Organiser</h2>
          <nav aria-label="Organiser" className="-mx-4 flex overflow-x-auto px-4 lg:mx-0 lg:block lg:overflow-visible lg:px-0">
            {tabs.map((t) => {
              const active = t.exact ? path === t.href : path.startsWith(t.href);
              return (
                <Link
                  key={t.href}
                  href={t.href}
                  className={cx(
                    "flex shrink-0 items-center gap-3 whitespace-nowrap px-3 py-3 text-[17px] font-light text-signal transition-colors hover:text-ink lg:border-b lg:border-signal lg:px-0 lg:py-[18px]",
                    active && "border-b-2 border-signal lg:border-b",
                  )}
                  aria-current={active ? "page" : undefined}
                >
                  {active && <ArrowRight className="hidden size-4 lg:block" strokeWidth={1.5} />}
                  {t.label}
                </Link>
              );
            })}
          </nav>
        </aside>
        <div className="min-w-0">{children}</div>
      </div>
    </RequireRole>
  );
}
