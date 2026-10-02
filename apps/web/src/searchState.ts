import { ActType, AlertFilters, Source } from "./api";
import { SOURCE_LABEL, TYPE_LABEL } from "./types";

export interface SearchState {
  q: string;
  source: Source | "";
  type: ActType | "";
  organ: string;
  theme: string;
  from: string;
  to: string;
  min: string;
  max: string;
  page: number;
}

export const TYPE_OPTIONS: { value: ActType | ""; label: string }[] = [
  { value: "", label: "Tudo" },
  { value: "nomeacao", label: "Nomeações" },
  { value: "exoneracao", label: "Exonerações" },
  { value: "contrato", label: "Contratos" },
  { value: "aditivo", label: "Aditivos" },
  { value: "licitacao", label: "Licitações" },
  { value: "dispensa", label: "Sem licitação" },
  { value: "decreto", label: "Decretos" },
  { value: "lei", label: "Leis" },
  { value: "resolucao", label: "Resoluções" },
  { value: "prestacao_contas", label: "Prestações de contas" },
  { value: "licenca_ambiental", label: "Licenças ambientais" },
  { value: "despacho", label: "Despachos" },
  { value: "edital", label: "Editais" },
];

export const EMPTY_STATE: SearchState = { q: "", source: "", type: "", organ: "", theme: "", from: "", to: "", min: "", max: "", page: 1 };

const KEYS = { q: "q", source: "fonte", type: "tipo", organ: "orgao", theme: "tema", from: "de", to: "ate", min: "valor_min", max: "valor_max" } as const;

export function stateFromQuery(search: string): SearchState {
  const p = new URLSearchParams(search);
  const type = p.get(KEYS.type) ?? "";
  const source = asSource(p.get(KEYS.source) ?? "");
  const page = Number(p.get("pagina"));
  return {
    q: p.get(KEYS.q) ?? "",
    source,
    type: Object.prototype.hasOwnProperty.call(TYPE_LABEL, type) ? (type as ActType) : "",
    organ: source === "diario_camara" ? "" : p.get(KEYS.organ) ?? "",
    theme: asTheme(p.get(KEYS.theme) ?? ""),
    from: p.get(KEYS.from) ?? "",
    to: p.get(KEYS.to) ?? "",
    min: p.get(KEYS.min) ?? "",
    max: p.get(KEYS.max) ?? "",
    page: Number.isInteger(page) && page > 1 ? page : 1,
  };
}

function asSource(value: string): Source | "" {
  return Object.prototype.hasOwnProperty.call(SOURCE_LABEL, value) ? (value as Source) : "";
}

export const THEME_LABEL: Record<string, string> = { meio_ambiente: "Meio ambiente" };

function asTheme(value: string): string {
  return Object.prototype.hasOwnProperty.call(THEME_LABEL, value) ? value : "";
}

export function withSource(s: SearchState, value: string): SearchState {
  const source = asSource(value);
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
  return Boolean(s.q || s.source || s.type || s.organ || s.theme || s.from || s.to || s.min || s.max);
}

export function apiParams(s: SearchState, limit: number): URLSearchParams | null {
  const p = new URLSearchParams({ q: s.q, limit: String(limit), offset: String((s.page - 1) * limit) });
  if (s.source) p.set("source", s.source);
  if (s.type) p.set("type", s.type);
  if (s.organ) p.set("organ", s.organ);
  if (s.theme) p.set("theme", s.theme);
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

const MIN_ALERT_QUERY = 3;

export function alertFilters(s: SearchState): AlertFilters {
  return { source: s.source, type: s.type, organ: s.organ, theme: s.theme };
}

export function canAlert(s: SearchState): boolean {
  const hasFilter = Object.values(alertFilters(s)).some(Boolean);
  return s.q.length >= MIN_ALERT_QUERY || (s.q === "" && hasFilter);
}

export function alertFilterNames(s: SearchState): string {
  const f = alertFilters(s);
  return [
    f.type && TYPE_LABEL[f.type].toLowerCase(),
    f.organ,
    f.theme && THEME_LABEL[f.theme].toLowerCase(),
    f.source && `Diário da ${SOURCE_LABEL[f.source]}`,
  ].filter(Boolean).join(" · ");
}

export type FilterKey = "type" | "source" | "organ" | "theme" | "from" | "to" | "min" | "max";

export interface ActiveFilter {
  key: FilterKey;
  label: string;
}

function brDate(iso: string): string {
  const [y, m, d] = iso.split("-");
  return d && m && y ? `${d}/${m}/${y}` : iso;
}

export function activeFilters(s: SearchState): ActiveFilter[] {
  const labels: [FilterKey, string][] = [
    ["type", s.type && (TYPE_OPTIONS.find((t) => t.value === s.type)?.label ?? TYPE_LABEL[s.type])],
    ["source", s.source && `Diário da ${SOURCE_LABEL[s.source]}`],
    ["organ", s.organ],
    ["theme", s.theme && THEME_LABEL[s.theme]],
    ["from", s.from && `a partir de ${brDate(s.from)}`],
    ["to", s.to && `até ${brDate(s.to)}`],
    ["min", s.min && `valor a partir de R$ ${s.min}`],
    ["max", s.max && `valor até R$ ${s.max}`],
  ];
  return labels.filter(([, label]) => label).map(([key, label]) => ({ key, label }));
}

export function withoutFilter(s: SearchState, key: FilterKey): SearchState {
  return { ...s, [key]: "", page: 1 };
}

export function withoutFilters(s: SearchState): SearchState {
  return { ...EMPTY_STATE, q: s.q };
}
