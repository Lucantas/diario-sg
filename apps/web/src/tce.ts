import { formatIsoDate } from "./registry";

export function diarioSearchHref(query: string): string {
  return `/?${new URLSearchParams({ q: query })}`;
}

export function condemnationsLabel(n: number): string {
  return n === 1 ? "1 condenação" : `${n} condenações`;
}

export function stalledPeriod(started: string | null, stalled: string | null): string {
  const start = formatIsoDate(started);
  const stop = formatIsoDate(stalled);
  if (start && stop) return `iniciada em ${start}, paralisada em ${stop}`;
  if (stop) return `paralisada em ${stop}`;
  return start ? `iniciada em ${start}` : "datas não informadas";
}

const LAST_PERIOD = 6;

export function rreoPeriodLabel(period: number): string {
  return period === LAST_PERIOD ? "ano fechado" : `até o ${period}º bimestre`;
}

export function coverageLabel(basisPoints: number): string {
  return `${(basisPoints / 100).toLocaleString("pt-BR", { maximumFractionDigits: 1 })}%`;
}
