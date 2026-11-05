import Image from "next/image";
import type { CSSProperties } from "react";
import { formatDate, type Heat } from "@/lib/api";
import { LINE, tradRows, type Entry, type Header, type PageDesc, type PrintStyle } from "@/lib/booklet";

export const PAGE_W = "148.5mm"; // A5
export const PAGE_H = "209.5mm"; // a hair under 210mm so a sheet never spills onto a second printed page

type Meta = { title: string; date: string };

const mm = (n: number) => `${n}mm`;

/**
 * One A5 page. Even pages sit on the left of a spread (fold on their right),
 * odd pages on the right (fold on their left), so the inner margin is wider.
 */
export function BookletPage({ page, number, style, meta }: { page: PageDesc; number: number; style: PrintStyle; meta: Meta }) {
  const left = number % 2 === 0;
  const trad = style === "traditional";
  const box: CSSProperties = {
    width: PAGE_W,
    height: PAGE_H,
    padding: `13mm ${left ? "15mm" : "12mm"} 0 ${left ? "12mm" : "15mm"}`,
    fontFamily: trad ? '"Times New Roman", Times, serif' : "var(--font-overpass), system-ui, sans-serif",
    fontSize: "9pt",
    lineHeight: mm(LINE),
    color: "#000",
  };

  return (
    <div className="relative box-border overflow-hidden bg-white" style={box}>
      {page.type === "cover" && <Cover style={style} meta={meta} />}
      {page.type === "back" && <Back style={style} meta={meta} />}
      {page.type === "contents" && <Contents toc={page.toc} style={style} />}
      {page.type === "content" && page.entries.map((e) => <EntryView key={e.heat.id} entry={e} style={style} />)}
      {page.type === "content" && (
        <span className={`absolute bottom-[9mm] ${left ? "left-[12mm]" : "right-[12mm]"} ${trad ? "" : "font-print-mono"}`} style={{ fontSize: "8pt", lineHeight: 1 }}>
          {number}
        </span>
      )}
    </div>
  );
}

// ---------- special pages ----------

function Cover({ style, meta }: { style: PrintStyle; meta: Meta }) {
  if (style === "traditional")
    return (
      <div className="flex h-[170mm] flex-col items-center justify-center text-center">
        <Image src="/logo.png" alt="" width={474} height={471} className="mb-[8mm] h-[30mm] w-auto" />
        <h1 style={{ fontSize: "26pt", lineHeight: 1.15, fontWeight: 700 }}>{meta.title}</h1>
        <p style={{ fontSize: "13pt", marginTop: "6mm" }}>Programme of Events</p>
        {meta.date && <p style={{ fontSize: "11pt", marginTop: "3mm" }}>{formatDate(meta.date)}</p>}
      </div>
    );
  return (
    <div className="flex h-[184mm] flex-col justify-between">
      <Image src="/logo.png" alt="" width={474} height={471} className="h-[26mm] w-auto self-start" />
      <p className="font-print-mono font-semibold uppercase" style={{ fontSize: "7pt", letterSpacing: "0.18em", color: "#e5322d" }}>
        {meta.date ? formatDate(meta.date) : "Programme of events"}
      </p>
      <div>
        <h1 className="font-print font-extrabold" style={{ fontSize: "40pt", lineHeight: 0.92, letterSpacing: "-0.04em" }}>
          {meta.title}
        </h1>
        <div className="mt-[6mm] h-[1.2mm] w-full bg-black" />
        <p className="font-print-mono font-semibold uppercase" style={{ fontSize: "7.5pt", letterSpacing: "0.18em", marginTop: "3mm" }}>
          Programme of events
        </p>
      </div>
    </div>
  );
}

function Back({ style, meta }: { style: PrintStyle; meta: Meta }) {
  const trad = style === "traditional";
  return (
    <div className="flex h-[184mm] flex-col items-center justify-end text-center">
      <p className={trad ? "" : "font-print-mono uppercase"} style={{ fontSize: trad ? "10pt" : "7pt", letterSpacing: trad ? 0 : "0.18em" }}>
        {meta.title}
      </p>
    </div>
  );
}

function Contents({ toc, style }: { toc: { label: string; page: number; time: string }[]; style: PrintStyle }) {
  const trad = style === "traditional";
  return (
    <div>
      <p className={trad ? "font-bold" : "font-print font-extrabold"} style={{ fontSize: trad ? "13pt" : "20pt", letterSpacing: trad ? 0 : "-0.03em", marginBottom: "4mm" }}>
        Contents
      </p>
      <ol style={{ columnCount: 2, columnGap: "7mm", fontSize: "7.5pt", lineHeight: "3.7mm" }}>
        {toc.map((t) => (
          <li key={t.label + t.page} className="flex break-inside-avoid items-baseline gap-1">
            <span className="truncate">{t.label}</span>
            <span className="min-w-2 flex-1 border-b border-dotted border-black/40" />
            {t.time && <span className={trad ? "" : "font-print-mono"}>{t.time}</span>}
            <span className="w-3 text-center">·</span>
            <span className={trad ? "font-bold" : "font-print-mono font-semibold"}>{t.page}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}

// ---------- schedule entries ----------

function EntryView({ entry, style }: { entry: Entry; style: PrintStyle }) {
  return (
    <>
      {entry.headers.map((h, i) => (
        <HeaderView key={i} header={h} />
      ))}
      {style === "traditional" ? <TradHeat heat={entry.heat} /> : <ModernHeat heat={entry.heat} />}
    </>
  );
}

function HeaderView({ header }: { header: Header }) {
  if (header.kind === "title")
    return (
      <div style={{ height: mm(9.5), paddingTop: mm(3), fontWeight: 700, fontSize: "11pt", lineHeight: mm(5) }}>
        {header.text}
        {header.cont && " (continued)"}
      </div>
    );
  if (header.kind === "event")
    return header.cont ? (
      <div className="flex items-end border-b-[0.5mm] border-black" style={{ height: mm(9), paddingBottom: "1mm" }}>
        <span className="font-print-mono font-semibold uppercase" style={{ fontSize: "7pt", letterSpacing: "0.14em" }}>
          {header.text} · continued
        </span>
      </div>
    ) : (
      <div className="flex items-end border-b-[0.5mm] border-black" style={{ height: mm(15), paddingBottom: "1.2mm" }}>
        <span className="font-print font-extrabold" style={{ fontSize: "20pt", lineHeight: 1, letterSpacing: "-0.035em" }}>
          {header.text}
        </span>
      </div>
    );
  return (
    <div className="flex items-end font-print-mono font-semibold uppercase" style={{ height: mm(7.5), paddingBottom: "1mm", fontSize: "6.5pt", letterSpacing: "0.16em", color: "#555" }}>
      {header.text}
    </div>
  );
}

/** The old layout: "Heat 1" label, then Last | First | Group on ONE line per runner, runners 5+ in a second column. */
function TradHeat({ heat }: { heat: Heat }) {
  const row = (s: Heat["students"][number]) => (
    <div key={s.id} className="grid" style={{ gridTemplateColumns: "23mm 22mm 10mm", height: mm(LINE) }}>
      <span className="truncate">{s.last_name}</span>
      <span className="truncate">{s.first_name}</span>
      <b>{s.group_code}</b>
    </div>
  );
  const rows = tradRows(heat.students.length);
  return (
    <div className="grid" style={{ gridTemplateColumns: "14mm 1fr 1fr", columnGap: "3mm", height: mm(rows * LINE + 3.2), paddingBottom: mm(3.2) }}>
      <div>
        <b>Heat {heat.number}</b>
        {heat.time && <div style={{ fontSize: "8pt" }}>{heat.time}</div>}
      </div>
      <div>{heat.students.slice(0, 4).map(row)}</div>
      <div>{heat.students.slice(4).map(row)}</div>
    </div>
  );
}

function ModernHeat({ heat }: { heat: Heat }) {
  const rows = Math.max(Math.ceil(heat.students.length / 2) * LINE, 12);
  return (
    <div className="grid border-t border-black/20" style={{ gridTemplateColumns: "13mm 1fr", height: mm(rows + 4), paddingTop: mm(2) }}>
      <div>
        <p className="font-print font-extrabold tabular-nums" style={{ fontSize: "26pt", lineHeight: 0.85, letterSpacing: "-0.04em" }}>
          {heat.number}
        </p>
        {heat.time && <p className="font-print-mono font-semibold" style={{ fontSize: "7pt", lineHeight: 1, marginTop: "1.5mm" }}>{heat.time}</p>}
      </div>
      <ol className="grid grid-cols-2 content-start gap-x-4" style={{ gridAutoRows: mm(LINE) }}>
        {heat.students.map((s) => (
          <li key={s.id} className="flex items-baseline gap-2 whitespace-nowrap" style={{ lineHeight: mm(LINE) }}>
            <span className="min-w-0 flex-1 truncate">
              {s.first_name} {s.last_name}
            </span>
            <span className="font-print-mono font-bold" style={{ fontSize: "7pt" }}>
              {s.group_code}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}
