import { describe, expect, it } from "vitest";
import { monthSpan, transferYears } from "./federal";

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
});
