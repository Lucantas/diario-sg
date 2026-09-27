import { FederalReport, SpecialTransfer } from "./api";
import { formatCompactCents } from "./panels";
import { formatIsoDate } from "./registry";
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

const PLAN_STATUS: Record<string, string> = {
  CIENTE: "aceita pelo município",
  IMPEDIDO: "impedida",
};

export function specialStatus(status: string): string {
  return PLAN_STATUS[status] ?? status.toLowerCase().replace(/_/g, " ");
}

export function specialPayment(st: SpecialTransfer): string {
  if (st.paid_cents === 0) return "nada pago";
  const paid = `${formatCompactCents(st.paid_cents)} pago`;
  return st.last_paid_at ? `${paid} (última ordem bancária em ${formatIsoDate(st.last_paid_at)})` : paid;
}

export function specialExecution(st: SpecialTransfer): string {
  if (!st.report_kind || !st.report_at) return "sem relatório de gestão";
  return `relatório ${st.report_kind.toLowerCase()} de ${formatIsoDate(st.report_at)}: ${formatCompactCents(st.executed_cents)} executado, ` +
    `${formatCompactCents(st.pending_cents)} pendente`;
}
