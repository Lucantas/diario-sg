import { ActType, LatestEdition } from "./api";
import { formatCompactCents } from "./panels";
import { EMPTY_STATE, queryFromState } from "./searchState";

const TYPE_COUNT: Record<ActType, [string, string]> = {
  nomeacao: ["nomeação", "nomeações"], exoneracao: ["exoneração", "exonerações"], contrato: ["contrato", "contratos"],
  aditivo: ["aditivo", "aditivos"], licitacao: ["licitação", "licitações"], dispensa: ["dispensa de licitação", "dispensas de licitação"],
  decreto: ["decreto", "decretos"], lei: ["lei", "leis"], portaria: ["portaria", "portarias"], resolucao: ["resolução", "resoluções"],
  despacho: ["despacho", "despachos"], edital: ["edital", "editais"], ata: ["ata", "atas"], corrigenda: ["corrigenda", "corrigendas"],
  prestacao_contas: ["prestação de contas", "prestações de contas"], licenca_ambiental: ["licença ambiental", "licenças ambientais"],
  outro: ["outro ato", "outros atos"],
};

const WEEKDAYS = ["domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"];
const MONTHS = ["janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"];
const DAY_MS = 86_400_000;
const WEEK_DAYS = 7;

export function homeEdition(items: LatestEdition[]): LatestEdition | null {
  return items.find((e) => e.total_acts > 0) ?? null;
}

export function editionTitle(e: Pick<LatestEdition, "edition_number" | "is_extra">): string {
  const n = Number(e.edition_number);
  const number = e.edition_number && Number.isFinite(n) ? n.toLocaleString("pt-BR") : e.edition_number;
  const base = number ? `Edição ${number}` : "Edição sem número";
  return e.is_extra ? `${base} (extra)` : base;
}

export function typeCountLabel(type: ActType, acts: number, valueCents: number): string {
  const [one, many] = TYPE_COUNT[type];
  const label = `${acts.toLocaleString("pt-BR")} ${acts === 1 ? one : many}`;
  return valueCents > 0 ? `${label}, ${formatCompactCents(valueCents)}` : label;
}

export function actsLabel(n: number): string {
  return n === 1 ? "o único ato desta edição" : `Todos os ${n.toLocaleString("pt-BR")} atos desta edição`;
}

export function relativeDay(isoDay: string, today: Date): string {
  const [y, m, d] = isoDay.split("-").map(Number);
  const day = Date.UTC(y, m - 1, d);
  const now = Date.UTC(today.getFullYear(), today.getMonth(), today.getDate());
  const diff = Math.round((now - day) / DAY_MS);
  if (diff === 0) return "hoje";
  if (diff === 1) return "ontem";
  if (diff > 1 && diff < WEEK_DAYS) return WEEKDAYS[new Date(day).getUTCDay()];
  return longDate(isoDay);
}

export function longDate(isoDay: string): string {
  const [y, m, d] = isoDay.split("-").map(Number);
  return `${d} de ${MONTHS[m - 1]} de ${y}`;
}

export function editionHref(gazetteId: string, type: ActType | "" = ""): string {
  return "/" + queryFromState({ ...EMPTY_STATE, edition: gazetteId, type });
}
