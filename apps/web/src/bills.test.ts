import { describe, expect, it } from "vitest";
import { billApiParams, billPath, billsPageUrl, idleLabel, parseBillPath, phaseLabel, readBillsState } from "./bills";

describe("proposições", () => {
  it("nomeia cada fase", () => {
    expect(phaseLabel("virou_norma")).toBe("Virou norma");
    expect(phaseLabel("enviado_ao_executivo")).toBe("Enviada ao Executivo");
    expect(phaseLabel("em_comissao")).toBe("Em comissão");
  });

  it("lê os filtros da URL com o tema ambiental como padrão da página", () => {
    const s = readBillsState("?fase=em_comissao&parado=180&q=poda&pagina=2");
    expect(s).toEqual({ q: "poda", theme: "meio_ambiente", phase: "em_comissao", idle: 180, kind: "", page: 2 });
    expect(readBillsState("?tema=")).toMatchObject({ theme: "", idle: 0, page: 1 });
    expect(readBillsState("?parado=abc&fase=inventada")).toMatchObject({ idle: 0, phase: "" });
  });

  it("monta a consulta da API e a URL da página", () => {
    const s = { q: "poda", theme: "meio_ambiente", phase: "em_comissao" as const, idle: 180, kind: "todos", page: 3 };
    expect(billApiParams(s, 20).toString()).toBe("q=poda&theme=meio_ambiente&phase=em_comissao&min_idle_days=180&kind=todos&limit=20&offset=40");
    expect(billsPageUrl(s)).toBe("/proposicoes?q=poda&fase=em_comissao&parado=180&tipo=todos&pagina=3");
    expect(billsPageUrl({ ...s, theme: "", phase: "", idle: 0, kind: "", page: 1 })).toBe("/proposicoes?q=poda&tema=");
    expect(billsPageUrl({ q: "", theme: "meio_ambiente", phase: "", idle: 0, kind: "", page: 1 })).toBe("/proposicoes");
  });

  it("liga o processo à sua página", () => {
    expect(billPath("5564/2025")).toBe("/proposicoes/5564-2025");
    expect(parseBillPath("/proposicoes/5564-2025")).toBe("5564-2025");
    expect(parseBillPath("/proposicoes")).toBeNull();
  });

  it("descreve os dias sem movimentação", () => {
    expect(idleLabel(0)).toBe("movimentada hoje");
    expect(idleLabel(1)).toBe("1 dia sem movimentação");
    expect(idleLabel(800)).toBe("800 dias sem movimentação (2 anos)");
  });
});
