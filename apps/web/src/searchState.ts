import { ActType, Source } from "./api";
import { SOURCE_LABEL, TYPE_LABEL } from "./types";

export interface SearchState {
  q: string;
  source: Source | "";
  type: ActType | "";
  organ: string;
  from: string;
  to: string;
  min: string;
  max: string;
  page: number;
}

export const EMPTY_STATE: SearchState = { q: "", source: "", type: "", organ: "", from: "", to: "", min: "", max: "", page: 1 };

const KEYS = { q: "q", source: "fonte", type: "tipo", organ: "orgao", from: "de", to: "ate", min: "valor_min", max: "valor_max" } as const;

export function stateFromQuery(search: string): SearchState {
  const p = new URLSearchParams(search);
  const type = p.get(KEYS.type) ?? "";
  const source = p.get(KEYS.source) ?? "";
  const page = Number(p.get("pagina"));
  return {
    q: p.get(KEYS.q) ?? "",
    source: source in SOURCE_LABEL ? (source as Source) : "",
    type: type in TYPE_LABEL ? (type as ActType) : "",
    organ: p.get(KEYS.organ) ?? "",
    from: p.get(KEYS.from) ?? "",
    to: p.get(KEYS.to) ?? "",
    min: p.get(KEYS.min) ?? "",
    max: p.get(KEYS.max) ?? "",
    page: Number.isInteger(page) && page > 1 ? page : 1,
  };
}

export function withSource(s: SearchState, value: string): SearchState {
  const source = value in SOURCE_LABEL ? (value as Source) : "";
  return { ...s, source, organ: source === "diario_camara" ? "" : s.organ, page: 1 };
}

export function queryFromState(s: SearchState): string {
  const p = new URLSearchParams();
  for (const [field, key] of Object.entries(KEYS) as [keyof typeof KEYS, string][]) {
    if (s[field]) p.set(key, s[field]);
  }
  if (s.page > 1) p.set("pagina", String(s.page));
  const text = p.toString();
  return text ? `?${text}` : "";
}

export function parseBRL(text: string): string | null {
  const t = text.replace(/R\$/i, "").trim();
  if (!/^(\d{1,3}(\.\d{3})+|\d+)(,\d{1,2})?$/.test(t)) return null;
  return t.replace(/\./g, "").replace(",", ".");
}

export function hasSearch(s: SearchState) {
  return Boolean(s.q || s.source || s.type || s.organ || s.from || s.to || s.min || s.max);
}

export function apiParams(s: SearchState, limit: number): URLSearchParams | null {
  const p = new URLSearchParams({ q: s.q, limit: String(limit), offset: String((s.page - 1) * limit) });
  if (s.source) p.set("source", s.source);
  if (s.type) p.set("type", s.type);
  if (s.organ) p.set("organ", s.organ);
  if (s.from) p.set("from", s.from);
  if (s.to) p.set("to", s.to);
  for (const [field, key] of [["min", "min_value"], ["max", "max_value"]] as const) {
    if (!s[field]) continue;
    const value = parseBRL(s[field]);
    if (value === null) return null;
    p.set(key, value);
  }
  return p;
}

function unpagedParams(s: SearchState): URLSearchParams | null {
  const p = apiParams(s, 1);
  if (!p) return null;
  p.delete("limit");
  p.delete("offset");
  return p;
}

export function exportUrl(s: SearchState, format: "csv" | "json"): string | null {
  const p = unpagedParams(s);
  if (!p) return null;
  p.set("format", format);
  return `/api/v1/acts/export?${p}`;
}

export function feedUrl(s: SearchState): string | null {
  const p = unpagedParams(s);
  return p && `/api/v1/feeds/acts?${p}`;
}
