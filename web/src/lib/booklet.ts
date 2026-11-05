// Booklet layout: splits the schedule into A5 pages, then orders those pages onto
// A4 landscape sheets for a folded-and-stapled (saddle-stitch) booklet.
import { ageGroups, type Heat } from "@/lib/api";

export type PrintStyle = "traditional" | "modern";

export type Header =
  | { kind: "title"; text: string; label: string; cont: boolean } // traditional: "Under 16 100m (4 Heats)"
  | { kind: "event"; text: string; label: string; cont: boolean } // modern: big event name
  | { kind: "age"; text: string }; // modern: age group label

export type Entry = { headers: Header[]; heat: Heat };
export type TocEntry = { label: string; page: number; time: string };

export type PageDesc =
  | { type: "cover" }
  | { type: "contents"; toc: TocEntry[] }
  | { type: "content"; entries: Entry[] }
  | { type: "blank" }
  | { type: "back" };

// ---- sizes in millimetres (A5 page, 9pt text) ----
export const LINE = 4.6;
const BODY = 176; // usable height of one page
const TRAD_TITLE = 9.5;
const EVENT = 15;
const EVENT_CONT = 9;
const AGE = 7.5;
const TOC_PER_PAGE = 76;

// Old style: the first 4 runners go in the left column, the rest in the right.
export const tradRows = (n: number) => Math.max(Math.min(n, 4), n - 4);
const tradHeat = (n: number) => tradRows(n) * LINE + 3.2;
// Modern: two columns of names, big heat number on the left.
const modernHeat = (n: number) => Math.max(Math.ceil(n / 2) * LINE, 12) + 4;

const headerHeight = (h: Header) => (h.kind === "title" ? TRAD_TITLE : h.kind === "age" ? AGE : h.cont ? EVENT_CONT : EVENT);

function plural(n: number, word: string) {
  return `${n} ${word}${n === 1 ? "" : "s"}`;
}

/** Old booklet went age group first, then event. Modern goes event first, then age group. */
function order(heats: Heat[], style: PrintStyle): Heat[] {
  const eventRank = new Map<number, number>();
  heats.forEach((h) => eventRank.has(h.event_id) || eventRank.set(h.event_id, eventRank.size));
  const age = (h: Heat) => ageGroups.indexOf(h.age_group);
  return [...heats].sort((a, b) =>
    style === "traditional"
      ? age(a) - age(b) || eventRank.get(a.event_id)! - eventRank.get(b.event_id)! || a.number - b.number
      : eventRank.get(a.event_id)! - eventRank.get(b.event_id)! || age(a) - age(b) || a.number - b.number,
  );
}

function headersFor(style: PrintStyle, h: Heat, prev: Heat | null, heats: Heat[]): Header[] {
  if (style === "traditional") {
    if (prev && prev.event_id === h.event_id && prev.age_group === h.age_group) return [];
    const count = heats.filter((x) => x.event_id === h.event_id && x.age_group === h.age_group).length;
    const label = `${h.age_group} ${h.event_name}`;
    return [{ kind: "title", label, text: `${label} (${plural(count, "Heat")})`, cont: prev === null && h.number > 1 }];
  }
  const out: Header[] = [];
  const sameEvent = !!prev && prev.event_id === h.event_id;
  if (!sameEvent) {
    const first = !heats.some((x) => x.event_id === h.event_id && (ageGroups.indexOf(x.age_group) < ageGroups.indexOf(h.age_group) || (x.age_group === h.age_group && x.number < h.number)));
    out.push({ kind: "event", label: h.event_name, text: h.event_name, cont: !first });
  }
  if (!sameEvent || prev!.age_group !== h.age_group) out.push({ kind: "age", text: h.age_group });
  return out;
}

/** Fill pages top to bottom; a heat is never split, and every new page re-states what it is continuing. */
export function paginate(heats: Heat[], style: PrintStyle): Entry[][] {
  const sorted = order(heats, style);
  const pages: Entry[][] = [];
  let cur: Entry[] = [];
  let used = 0;
  for (const h of sorted) {
    const heatH = style === "traditional" ? tradHeat(h.students.length) : modernHeat(h.students.length);
    let headers = headersFor(style, h, cur.length ? cur[cur.length - 1].heat : null, sorted);
    let need = headers.reduce((n, x) => n + headerHeight(x), 0) + heatH;
    if (cur.length && used + need > BODY) {
      pages.push(cur);
      cur = [];
      used = 0;
      headers = headersFor(style, h, null, sorted);
      need = headers.reduce((n, x) => n + headerHeight(x), 0) + heatH;
    }
    cur.push({ headers, heat: h });
    used += need;
  }
  if (cur.length) pages.push(cur);
  return pages;
}

/**
 * Whole booklet in reading order: cover, optional contents, schedule pages, blank pages as padding, back cover.
 * Total is always a multiple of 4 because one folded sheet = 4 pages.
 */
export function buildPages(heats: Heat[], style: PrintStyle, withContents: boolean): PageDesc[] {
  const content = paginate(heats, style);

  // Each block's contents line points at the page it starts on and carries its first heat's time.
  const labels: { label: string; at: number; time: string }[] = [];
  content.forEach((entries, i) =>
    entries.forEach((e) =>
      e.headers.forEach((h) => {
        if ((h.kind === "title" || h.kind === "event") && !h.cont) labels.push({ label: h.label, at: i, time: e.heat.time });
      }),
    ),
  );
  const contentsPages = withContents ? Math.max(1, Math.ceil(labels.length / TOC_PER_PAGE)) : 0;
  const firstContent = 1 + contentsPages; // pages before the schedule: cover (+ contents)
  const toc: TocEntry[] = labels.map((l) => ({ label: l.label, page: firstContent + l.at + 1, time: l.time }));

  const pages: PageDesc[] = [{ type: "cover" }];
  for (let i = 0; i < contentsPages; i++) pages.push({ type: "contents", toc: toc.slice(i * TOC_PER_PAGE, (i + 1) * TOC_PER_PAGE) });
  content.forEach((entries) => pages.push({ type: "content", entries }));
  while ((pages.length + 1) % 4 !== 0) pages.push({ type: "blank" });
  pages.push({ type: "back" });
  return pages;
}

export type Sheet = { front: [number, number]; back: [number, number] };

/**
 * Saddle-stitch imposition. For N pages (multiple of 4), sheet s (0 = outermost):
 *   front: [N-2s, 2s+1]      back: [2s+2, N-2s-1]      (page numbers, left | right)
 * e.g. N=8: sheet 1 front 8|1, back 2|7; sheet 2 front 6|3, back 4|5.
 */
export function impose(total: number): Sheet[] {
  const sheets: Sheet[] = [];
  for (let s = 0; s < total / 4; s++) {
    sheets.push({ front: [total - 2 * s, 2 * s + 1], back: [2 * s + 2, total - 2 * s - 1] });
  }
  return sheets;
}
