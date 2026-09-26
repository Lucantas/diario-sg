import { FederalReport } from "./api";
import { formatYearMonth } from "./payments";

export interface TransferYear {
  year: number;
  total: number;
  byKind: { kind: string; value: number }[];
}

export function transferYears(transfers: FederalReport["transfers"]): TransferYear[] {
  const years = new Map<number, Map<string, number>>();
  for (const t of transfers) {
    const kinds = years.get(t.year) ?? new Map<string, number>();
    kinds.set(t.kind, (kinds.get(t.kind) ?? 0) + t.value_cents);
    years.set(t.year, kinds);
  }
  return [...years.entries()]
    .sort(([a], [b]) => b - a)
    .map(([year, kinds]) => {
      const byKind = [...kinds.entries()].map(([kind, value]) => ({ kind, value })).sort((a, b) => b.value - a.value);
      return { year, total: byKind.reduce((sum, k) => sum + k.value, 0), byKind };
    });
}

export function monthSpan(first: string, last: string): string {
  return first === last ? formatYearMonth(first) : `${formatYearMonth(first)} a ${formatYearMonth(last)}`;
}
