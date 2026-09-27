import { describe, expect, it } from "vitest";
import { MunicipalCommitment } from "./api";
import { commitmentHeading, commitmentProcess, objectPreview } from "./municipal";

const commitment = {
  entity: "FUNDO MUNICIPAL DE SAUDE", year: 2025, number: "1453", date: "2025-08-05", process_kind: "Licitação", process: "2888/2024",
  modality: "Pregão Eletrônico PE 90003/2025",
} as MunicipalCommitment;

describe("municipal", () => {
  it("descreve o empenho e o processo", () => {
    expect(commitmentHeading(commitment)).toBe("Empenho 1453/2025 · 05/08/2025 · FUNDO MUNICIPAL DE SAUDE");
    expect(commitmentProcess(commitment)).toBe("processo 2888/2024 · licitação · Pregão Eletrônico PE 90003/2025");
    expect(commitmentProcess({ ...commitment, process_kind: "Outros/Não aplicável", modality: "Outros/Não Aplicável" })).toBe("processo 2888/2024");
  });

  it("encurta objeto longo", () => {
    expect(objectPreview("curto")).toBe("curto");
    expect(objectPreview("a".repeat(300))).toBe(`${"a".repeat(240)}…`);
  });
});
