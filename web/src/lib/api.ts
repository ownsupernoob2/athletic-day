// Thin client for the Go API (proxied through Next at /api).

export type Role = "staff" | "organiser";

export type User = {
  id: number;
  name: string;
  email: string;
  role: Role;
  group_code: string;
  house: string;
};

export type AthleticEvent = {
  id: number;
  code: string;
  name: string;
  kind: "track" | "field";
  csv_header: string;
  position: number;
  active: boolean;
  entry_count: number;
};

export type Student = {
  id: number;
  first_name: string;
  last_name: string;
  dob: string;
  age: number;
  age_group: string;
  group_code: string;
  house: string;
  added_by_staff: boolean;
  from_kamar: boolean;
  updated_at: string;
  events: number[];
};

export type Settings = {
  title: string;
  event_date: string;
  min_events: number;
  heat_size: number;
  published: boolean;
  generated_at: string;
  day_start: string;
  day_end: string;
  entries_changed_at: string;
};

export type KamarStatus = {
  configured: boolean;
  linked: number;
  last: { at: string; sync: string; counts: { added: number; updated: number; skipped: number; bad_group: number } } | null;
};

export type GroupStatus = {
  code: string;
  house: string;
  staff: string[];
  students: number;
  complete: number;
  started: number;
  last_update: string;
};

export type HeatStudent = {
  id: number;
  first_name: string;
  last_name: string;
  group_code: string;
  house: string;
  age_group: string;
};

export type Heat = {
  id: number;
  event_id: number;
  event_code: string;
  event_name: string;
  kind: "track" | "field";
  age_group: string;
  number: number;
  time: string;
  students: HeatStudent[];
};

export type Unplaced = { event_id: number; age_group: string; student: HeatStudent };

export type ScheduleData = { settings: Settings; heats: Heat[]; unplaced: Unplaced[] };

export type PublicSchedule = { published: boolean; title: string; event_date: string; heats: Heat[] };

export type StaffUser = { id: number; name: string; email: string; role: Role; group_code: string; house: string };

export class ApiError extends Error {
  constructor(message: string, public status: number) {
    super(message);
  }
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(url, {
      method,
      headers: body instanceof FormData || body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : body instanceof FormData ? body : JSON.stringify(body),
      credentials: "same-origin",
    });
  } catch {
    throw new ApiError("Can't reach the server. Check your connection and try again.", 0);
  }
  const data = await res.json().catch(() => null);
  if (!res.ok) throw new ApiError(data?.error ?? "Something went wrong", res.status);
  return data as T;
}

export const api = {
  get: <T,>(url: string) => request<T>("GET", url),
  post: <T,>(url: string, body?: unknown) => request<T>("POST", url, body ?? {}),
  put: <T,>(url: string, body?: unknown) => request<T>("PUT", url, body ?? {}),
  patch: <T,>(url: string, body?: unknown) => request<T>("PATCH", url, body ?? {}),
  del: <T,>(url: string) => request<T>("DELETE", url),
  upload: <T,>(url: string, form: FormData) => request<T>("POST", url, form),
};

export const fetcher = <T,>(url: string) => api.get<T>(url);

export function errorMessage(e: unknown) {
  return e instanceof Error ? e.message : "Something went wrong";
}

// ---------- display helpers ----------

export const houseStyles: Record<string, { dot: string; badge: string }> = {
  Stuart: { dot: "bg-[#d6334a]", badge: "bg-white text-ink ring-ink/15" },
  Tudor: { dot: "bg-[#1f8a5b]", badge: "bg-white text-ink ring-ink/15" },
  Windsor: { dot: "bg-[#2563c9]", badge: "bg-white text-ink ring-ink/15" },
  Hanover: { dot: "bg-[#d99a00]", badge: "bg-white text-ink ring-ink/15" },
};
export const houseNames = ["Stuart", "Tudor", "Windsor", "Hanover"];
export const ageGroups = ["Under 14", "Under 15", "Under 16", "Under 19", "Over 19"];

export function formatDate(iso: string) {
  if (!iso) return "";
  const d = new Date(iso + (iso.length === 10 ? "T00:00:00" : ""));
  return d.toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" });
}

export function timeAgo(iso: string) {
  if (!iso) return "";
  const s = (Date.now() - new Date(iso).getTime()) / 1000;
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)} min ago`;
  if (s < 86400) return `${Math.floor(s / 3600)} h ago`;
  return `${Math.floor(s / 86400)} d ago`;
}
