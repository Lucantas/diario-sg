import { BillPhase } from "./api";

const PHASE_LABELS: Record<BillPhase, string> = {
  virou_lei: "Virou lei",
  vetado: "Vetada",
  rejeitado: "Rejeitada",
  retirado: "Retirada pelo autor",
  arquivado: "Arquivada",
  enviado_ao_executivo: "Enviada ao Executivo",
  aprovado: "Aprovada",
  em_votacao: "Em votação",
  em_comissao: "Em comissão",
  apresentado: "Apresentada",
};

export const PHASES = Object.keys(PHASE_LABELS) as BillPhase[];

export const DEFAULT_THEME = "meio_ambiente";

export function phaseLabel(p: BillPhase): string {
  return PHASE_LABELS[p];
}

export interface BillsState {
  q: string;
  theme: string;
  phase: BillPhase | "";
  idle: number;
  kind: string;
  page: number;
}

function positive(s: string | null, fallback: number): number {
  const n = Number(s);
  return Number.isInteger(n) && n > 0 ? n : fallback;
}

export function readBillsState(search: string): BillsState {
  const p = new URLSearchParams(search);
  const phase = p.get("fase") ?? "";
  return {
    q: p.get("q") ?? "",
    theme: p.has("tema") ? p.get("tema") ?? "" : DEFAULT_THEME,
    phase: (PHASES as string[]).includes(phase) ? (phase as BillPhase) : "",
    idle: positive(p.get("parado"), 0),
    kind: p.get("tipo") ?? "",
    page: positive(p.get("pagina"), 1),
  };
}

export function billApiParams(s: BillsState, perPage: number): URLSearchParams {
  const p = new URLSearchParams();
  if (s.q) p.set("q", s.q);
  if (s.theme) p.set("theme", s.theme);
  if (s.phase) p.set("phase", s.phase);
  if (s.idle > 0) p.set("min_idle_days", String(s.idle));
  if (s.kind) p.set("kind", s.kind);
  p.set("limit", String(perPage));
  p.set("offset", String((s.page - 1) * perPage));
  return p;
}

export function billsPageUrl(s: BillsState): string {
  const p = new URLSearchParams();
  if (s.q) p.set("q", s.q);
  if (s.theme !== DEFAULT_THEME) p.set("tema", s.theme);
  if (s.phase) p.set("fase", s.phase);
  if (s.idle > 0) p.set("parado", String(s.idle));
  if (s.kind) p.set("tipo", s.kind);
  if (s.page > 1) p.set("pagina", String(s.page));
  const qs = p.toString();
  return qs ? `/proposicoes?${qs}` : "/proposicoes";
}

export function billPath(process: string): string {
  return `/proposicoes/${process.replace("/", "-")}`;
}

export function parseBillPath(path: string): string | null {
  const m = path.match(/^\/proposicoes\/(\d+-\d{4})$/);
  return m ? m[1] : null;
}

const DAYS_PER_YEAR = 365;

export function idleLabel(days: number): string {
  if (days <= 0) return "movimentada hoje";
  const base = days === 1 ? "1 dia sem movimentação" : `${days} dias sem movimentação`;
  const years = Math.floor(days / DAYS_PER_YEAR);
  if (years === 0) return base;
  return `${base} (${years === 1 ? "1 ano" : `${years} anos`})`;
}
