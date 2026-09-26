import { describe, expect, it } from "vitest";
import { formatCount, parseStaffUnit, staffApiPath, staffHref, staffMonthLabel } from "./staff";

describe("staff", () => {
  it("leva a unidade da página para a API", () => {
    const unit = parseStaffUnit("?unidade=C%C3%82MARA+S%C3%83O+GON%C3%87ALO");
    expect(unit).toBe("CÂMARA SÃO GONÇALO");
    expect(staffApiPath(unit)).toBe("/v1/panels/staff?unit=C%C3%82MARA%20S%C3%83O%20GON%C3%87ALO");
    expect(staffHref("")).toBe("/pessoal");
    expect(staffApiPath("")).toBe("/v1/panels/staff");
  });

  it("formata o mês e as contagens", () => {
    expect(staffMonthLabel("2025-01")).toBe("jan/2025");
    expect(formatCount(0)).toBe("—");
    expect(formatCount(15623)).toBe("15.623");
  });
});
