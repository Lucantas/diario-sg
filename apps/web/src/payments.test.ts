import { describe, expect, it } from "vitest";
import { coverageLabel, formatYearMonth } from "./payments";

describe("pagamentos", () => {
  it("formata mês e ano", () => {
    expect(formatYearMonth("2021-01")).toBe("jan/2021");
    expect(coverageLabel({ from: "2021-01", to: "2026-08" })).toBe("jan/2021 a ago/2026");
  });
});
