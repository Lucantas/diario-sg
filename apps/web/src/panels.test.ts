import { describe, expect, it } from "vitest";
import { contractsLabel, formatCompactCents, panelApiPath, panelHref, parsePanelParams } from "./panels";

describe("parsePanelParams", () => {
  it("lê ano e órgão do endereço", () => {
    expect(parsePanelParams("?ano=2024&orgao=semed")).toEqual({ year: 2024, organ: "SEMED" });
  });

  it("descarta ano inválido", () => {
    expect(parsePanelParams("?ano=abc").year).toBeNull();
    expect(parsePanelParams("?ano=1999").year).toBeNull();
    expect(parsePanelParams("?ano=2024.5").year).toBeNull();
    expect(parsePanelParams("").year).toBeNull();
  });
});

describe("panelHref e panelApiPath", () => {
  it("põem só os filtros escolhidos", () => {
    expect(panelHref({ year: null, organ: "" })).toBe("/paineis");
    expect(panelHref({ year: 2024, organ: "FMS" })).toBe("/paineis?ano=2024&orgao=FMS");
    expect(panelApiPath({ year: null, organ: "SEMED" })).toBe("/v1/panels/suppliers?organ=SEMED");
  });
});

describe("formatCompactCents", () => {
  it("abrevia milhares e milhões, mas não valores pequenos", () => {
    expect(formatCompactCents(10556000000)).toMatch(/^R\$\s105,6\smi$/);
    expect(formatCompactCents(258)).toMatch(/^R\$\s2,58$/);
    expect(formatCompactCents(0)).toMatch(/^R\$\s0,00$/);
  });
});

describe("contractsLabel", () => {
  it("diz quantas contratações ou que só há aditivos", () => {
    expect(contractsLabel(0)).toBe("Só aditivos e prorrogações");
    expect(contractsLabel(1)).toBe("1 contratação");
    expect(contractsLabel(3)).toBe("3 contratações");
  });
});
