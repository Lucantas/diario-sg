import { describe, expect, it } from "vitest";
import { PNCPContract } from "./api";
import { pncpHeading, pncpTerm } from "./pncp";

const contract: PNCPContract = {
  control_number: "28636579000100-2-000010/2025",
  url: "https://pncp.gov.br/app/contratos/28636579000100/2025/10",
  org_cnpj: "28636579000100",
  unit: "SECRETARIA MUNICIPAL DE SAÚDE",
  kind: "Contrato (termo inicial)",
  number: "015/2025",
  process: "12345/2024",
  object: "Serviço de limpeza",
  value_cents: 123456789,
  signed_at: "2025-03-10",
  starts_at: "2025-03-11",
  ends_at: "2026-03-10",
};

describe("pncp", () => {
  it("mostra a vigência com início e fim", () => {
    expect(pncpTerm(contract)).toBe("11/03/2025 a 10/03/2026");
  });

  it("diz quando a vigência não veio", () => {
    expect(pncpTerm({ ...contract, starts_at: null, ends_at: null })).toBe("vigência não informada");
  });

  it("junta tipo, número e processo no título", () => {
    expect(pncpHeading(contract)).toBe("Contrato (termo inicial) 015/2025, processo 12345/2024");
    expect(pncpHeading({ ...contract, process: "" })).toBe("Contrato (termo inicial) 015/2025");
  });
});
