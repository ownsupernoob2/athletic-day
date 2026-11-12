"use client";

import { Check } from "lucide-react";
import { cx } from "@/components/ui";
import type { AthleticEvent } from "@/lib/api";

/** Tap-to-toggle chips for every event, grouped into Track and Field. Selected = solid ink. */
export function EventPicker({
  events,
  selected,
  onToggle,
  compact,
  limits,
}: {
  events: AthleticEvent[];
  selected: number[];
  onToggle: (event: AthleticEvent, on: boolean) => void;
  compact?: boolean;
  /** When given, events that would go over the limits can't be picked. Taking one off is always allowed. */
  limits?: { maxTotal: number; maxPerType: number };
}) {
  const sel = new Set(selected);
  const chosen = events.filter((e) => sel.has(e.id));
  const blockedReason = (e: AthleticEvent) => {
    if (!limits || sel.has(e.id)) return "";
    if (chosen.length >= limits.maxTotal) return `At most ${limits.maxTotal} events. Take one off first.`;
    if (chosen.filter((c) => c.kind === e.kind).length >= limits.maxPerType) return `At most ${limits.maxPerType} ${e.kind} events. Take one off first.`;
    return "";
  };
  const groups = [
    { label: "Track", items: events.filter((e) => e.kind === "track") },
    { label: "Field", items: events.filter((e) => e.kind === "field") },
  ].filter((g) => g.items.length > 0);

  return (
    <div className={cx("grid gap-x-8 gap-y-3", !compact && "sm:grid-cols-2")}>
      {groups.map((g) => (
        <div key={g.label} className="grid grid-cols-[3.25rem_1fr] items-start gap-2">
          <p className="pt-2 font-mono text-[10px] font-semibold uppercase tracking-[0.18em] text-zinc-400">{g.label}</p>
          <div className="flex flex-wrap gap-1.5">
            {g.items.map((e) => {
              const on = sel.has(e.id);
              const blocked = blockedReason(e);
              return (
                <button
                  key={e.id}
                  type="button"
                  aria-pressed={on}
                  disabled={!!blocked}
                  title={blocked || e.name}
                  onClick={() => onToggle(e, !on)}
                  className={cx(
                    "inline-flex h-8 items-center gap-1 rounded-sm px-2.5 text-sm font-medium transition-colors",
                    on
                      ? "bg-ink text-white hover:bg-signal"
                      : blocked
                        ? "bg-transparent text-zinc-300 ring-1 ring-inset ring-zinc-200"
                        : "bg-transparent text-zinc-600 ring-1 ring-inset ring-ink/20 hover:bg-white hover:text-ink hover:ring-ink",
                  )}
                >
                  {on && <Check className="size-3.5" strokeWidth={3} />}
                  {e.code}
                </button>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}
