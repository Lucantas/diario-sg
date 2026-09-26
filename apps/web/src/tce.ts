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
