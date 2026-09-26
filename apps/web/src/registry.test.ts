import { describe, expect, it } from "vitest";
import { formatAddress, formatIsoDate, formatRegistryMonth, isActive } from "./registry";

describe("formatAddress", () => {
  it("junta as partes que existem", () => {
    expect(formatAddress({ street: "RUA DOUTOR NILO PECANHA", number: "120", complement: "SALA 2", district: "CENTRO", city: "SAO GONCALO", uf: "RJ", zip: "24445300" }))
      .toBe("RUA DOUTOR NILO PECANHA, 120, SALA 2, CENTRO, SAO GONCALO/RJ, CEP 24445-300");
  });

  it("pula número, complemento e bairro vazios", () => {
    expect(formatAddress({ street: "ESTRADA X", number: "", complement: "", district: "", city: "NITEROI", uf: "RJ", zip: "" }))
      .toBe("ESTRADA X, NITEROI/RJ");
  });
});

describe("isActive", () => {
  it("só a situação Ativa é ativa", () => {
    expect(isActive("Ativa")).toBe(true);
    expect(isActive("Baixada")).toBe(false);
    expect(isActive("Inapta")).toBe(false);
  });
});

describe("datas", () => {
  it("formata dia e mês de referência", () => {
    expect(formatIsoDate("2005-05-18")).toBe("18/05/2005");
    expect(formatIsoDate(null)).toBe("");
    expect(formatRegistryMonth("2026-09")).toBe("setembro de 2026");
  });
});
