import { formatYearMonth } from "./payments";

export function staffApiPath(unit: string): string {
  return unit ? `/v1/panels/staff?unit=${encodeURIComponent(unit)}` : "/v1/panels/staff";
}

export function staffHref(unit: string): string {
  return unit ? `/pessoal?unidade=${encodeURIComponent(unit)}` : "/pessoal";
}

export function parseStaffUnit(search: string): string {
  return new URLSearchParams(search).get("unidade")?.trim() ?? "";
}

export function staffMonthLabel(month: string): string {
  return formatYearMonth(month);
}

export function formatCount(n: number): string {
  return n === 0 ? "—" : n.toLocaleString("pt-BR");
}
