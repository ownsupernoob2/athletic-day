"use client";

import { ArrowLeft, Printer } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";
import useSWR from "swr";
import { BookletPage, PAGE_H, PAGE_W } from "@/components/BookletPages";
import { buttonClass, cx, Empty, Spinner } from "@/components/ui";
import { fetcher, type PublicSchedule } from "@/lib/api";
import { buildPages, impose, type PrintStyle } from "@/lib/booklet";

type Layout = "booklet" | "pages";

const styleOptions: { value: PrintStyle; title: string; text: string }[] = [
  { value: "traditional", title: "Traditional", text: "The classic list from the old system: plain serif type, age group first." },
  { value: "modern", title: "Modern", text: "Large event names and heat numbers, event first." },
];

const layoutOptions: { value: Layout; title: string; text: string }[] = [
  { value: "booklet", title: "Booklet sheets", text: "A4 landscape, two pages per side. Print double-sided, fold, staple." },
  { value: "pages", title: "Single pages", text: "A5 pages in reading order. Good for checking or a PDF." },
];

/** Pick a style and layout, check the preview, then print. */
export default function PrintBooklet() {
  const { data } = useSWR<PublicSchedule>("/api/public/schedule", fetcher);
  const [style, setStyle] = useState<PrintStyle>("traditional");
  const [layout, setLayout] = useState<Layout>("booklet");
  const [contents, setContents] = useState(false);

  const pages = useMemo(() => (data ? buildPages(data.heats, style, contents) : []), [data, style, contents]);
  const sheets = useMemo(() => impose(pages.length), [pages.length]);

  if (!data) return <Spinner />;
  if (!data.published) return <Empty title="The schedule isn't published yet">Nothing to print for now.</Empty>;

  const meta = { title: data.title, date: data.event_date };
  const page = (n: number) => <BookletPage page={pages[n - 1]} number={n} style={style} meta={meta} />;

  return (
    <div>
      {/* Page size for the printer. Margins are zero because the pages carry their own. */}
      <style>{`@page { size: ${layout === "booklet" ? "297mm 210mm" : "148.5mm 210mm"}; margin: 0; }`}</style>

      <div className="print:hidden">
        <div className="mb-6 flex items-center justify-between gap-3">
          <Link href="/schedule" className={buttonClass("ghost", "sm")}>
            <ArrowLeft className="size-4" /> Back to schedule
          </Link>
          <button onClick={() => window.print()} className={buttonClass("primary")}>
            <Printer className="size-4" /> Print
          </button>
        </div>

        <h1 className="mb-6 border-b border-signal pb-4 text-4xl font-light text-signal sm:text-5xl">Print booklet</h1>

        <div className="mb-8 grid gap-6 md:grid-cols-2">
          <Choice label="Style" value={style} onChange={setStyle} options={styleOptions} />
          <Choice label="Layout" value={layout} onChange={setLayout} options={layoutOptions} />
        </div>

        <label className="mb-8 flex w-fit cursor-pointer items-center gap-3 text-sm">
          <input type="checkbox" checked={contents} onChange={(e) => setContents(e.target.checked)} className="size-4 accent-ink" />
          <span>
            <span className="font-medium">Include a contents page</span>
            <span className="text-zinc-500"> (events with their start times and page numbers)</span>
          </span>
        </label>

        <div className="mb-8 rounded-md border border-zinc-200 bg-white p-4 text-sm text-zinc-600">
          <p className="font-medium text-ink">
            {pages.length} pages{layout === "booklet" ? ` on ${sheets.length} sheets of A4 (${sheets.length * 2} printed sides)` : ""}
          </p>
          {layout === "booklet" ? (
            <ol className="mt-2 list-decimal space-y-1 pl-5">
              <li>Print on A4, landscape, double-sided, <strong>flip on short edge</strong>. Scale 100% (actual size), no margins.</li>
              <li>Keep the sheets in order. Sheet 1 is the outside of the booklet.</li>
              <li>Fold the whole stack in half and staple along the fold. Page 1 is the front cover.</li>
            </ol>
          ) : (
            <p className="mt-1">Each A5 page prints on its own page, in reading order.</p>
          )}
        </div>
      </div>

      <div className="flex flex-wrap gap-x-8 gap-y-10 print:block">
        {layout === "booklet"
          ? sheets.flatMap((s, i) =>
              (["front", "back"] as const).map((side) => {
                const [l, r] = s[side];
                return (
                  <div key={`${i}-${side}`} className="print-side">
                    <p className="mb-2 font-mono text-[11px] font-semibold uppercase tracking-[0.14em] text-zinc-500 print:hidden">
                      Sheet {i + 1} · {side === "front" ? "front" : "back"} · pages {l} | {r}
                    </p>
                    <div className="sheet-zoom">
                      <div className="paper-shadow grid bg-white" style={{ width: "297mm", height: PAGE_H, gridTemplateColumns: `${PAGE_W} ${PAGE_W}` }}>
                        {page(l)}
                        {page(r)}
                      </div>
                    </div>
                  </div>
                );
              }),
            )
          : pages.map((_, i) => (
              <div key={i} className="print-side">
                <p className="mb-2 font-mono text-[11px] font-semibold uppercase tracking-[0.14em] text-zinc-500 print:hidden">Page {i + 1}</p>
                <div className="sheet-zoom">
                  <div className="paper-shadow" style={{ width: PAGE_W, height: PAGE_H }}>
                    {page(i + 1)}
                  </div>
                </div>
              </div>
            ))}
      </div>
    </div>
  );
}

function Choice<T extends string>({ label, value, onChange, options }: { label: string; value: T; onChange: (v: T) => void; options: { value: T; title: string; text: string }[] }) {
  return (
    <fieldset>
      <legend className="mb-2 font-mono text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-500">{label}</legend>
      <div className="grid gap-2">
        {options.map((o) => (
          <label key={o.value} className={cx("flex cursor-pointer items-start gap-3 rounded-md border p-3.5 transition-colors", value === o.value ? "border-ink bg-white" : "border-zinc-200 hover:border-zinc-400")}>
            <input type="radio" name={label} checked={value === o.value} onChange={() => onChange(o.value)} className="mt-1 size-4 accent-ink" />
            <span>
              <span className="block font-medium">{o.title}</span>
              <span className="block text-sm text-zinc-500">{o.text}</span>
            </span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}
