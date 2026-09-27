import { describe, expect, it } from "vitest";
import { SpecialTransfer } from "./api";
import { monthSpan, specialExecution, specialPayment, specialStatus, transferYears } from "./federal";

describe("federal", () => {
  it("soma as transferências por ano e tipo", () => {
    const years = transferYears([
      { year: 2025, kind: "Legais", function: "Saúde", value_cents: 10 },
      { year: 2026, kind: "Legais", function: "Saúde", value_cents: 5 },
      { year: 2025, kind: "Constitucionais", function: "Encargos", value_cents: 30 },
      { year: 2025, kind: "Legais", function: "Educação", value_cents: 1 },
    ]);
    expect(years.map((y) => y.year)).toEqual([2026, 2025]);
    expect(years[1].total).toBe(41);
    expect(years[1].byKind).toEqual([{ kind: "Constitucionais", value: 30 }, { kind: "Legais", value: 11 }]);
  });

  it("descreve o período dos pagamentos", () => {
    expect(monthSpan("2025-01", "2025-01")).toBe("jan/2025");
    expect(monthSpan("2024-03", "2025-01")).toBe("mar/2024 a jan/2025");
  });

  it("resume pagamento e execução da transferência especial", () => {
    const plain = (text: string) => text.replace(/\s/g, " ");
    const st = {
      status: "CIENTE", paid_cents: 267000000, last_paid_at: "2023-03-29", report_kind: "Final", report_at: "2024-12-30",
      executed_cents: 241778012, pending_cents: 25221988,
    } as SpecialTransfer;
    expect(specialStatus(st.status)).toBe("aceita pelo município");
    expect(specialStatus("EM_ANALISE")).toBe("em analise");
    expect(plain(specialPayment(st))).toBe("R$ 2,7 mi pago (última ordem bancária em 29/03/2023)");
    expect(specialPayment({ ...st, paid_cents: 0 })).toBe("nada pago");
    expect(plain(specialExecution(st))).toBe("relatório final de 30/12/2024: R$ 2,4 mi executado, R$ 252,2 mil pendente");
    expect(specialExecution({ ...st, report_kind: "" })).toBe("sem relatório de gestão");
  });
});
