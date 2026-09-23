import { describe, expect, it } from "vitest";
import {
  EMPTY_STATE, SearchState, apiParams, exportUrl, feedUrl, hasSearch, parseBRL, queryFromState, stateFromQuery,
} from "./searchState";

describe("parseBRL", () => {
  it("aceita formato brasileiro e inteiro", () => {
    expect(parseBRL("1.500,50")).toBe("1500.50");
    expect(parseBRL("1500")).toBe("1500");
    expect(parseBRL("1500,5")).toBe("1500.5");
    expect(parseBRL(" R$ 20.000 ")).toBe("20000");
  });

  it("recusa o que não é valor", () => {
    expect(parseBRL("")).toBeNull();
    expect(parseBRL("abc")).toBeNull();
    expect(parseBRL("1,2,3")).toBeNull();
    expect(parseBRL("-5")).toBeNull();
    expect(parseBRL("1.50")).toBeNull();
  });
});

describe("URL da busca", () => {
  it("vai e volta sem perder filtros", () => {
    const s: SearchState = {
      q: "limpeza OU coleta", type: "contrato", organ: "SEMED", from: "2024-01-01",
      to: "2024-12-31", min: "1.000,00", max: "", page: 3,
    };

    expect(stateFromQuery(queryFromState(s))).toEqual(s);
  });

  it("usa nomes em português e omite o vazio", () => {
    expect(queryFromState({ ...EMPTY_STATE, q: "merenda", organ: "SEMED" })).toBe("?q=merenda&orgao=SEMED");
    expect(queryFromState(EMPTY_STATE)).toBe("");
  });

  it("ignora página inválida e tipo desconhecido", () => {
    expect(stateFromQuery("?pagina=-2&tipo=bobagem")).toEqual(EMPTY_STATE);
    expect(stateFromQuery("?pagina=2.5").page).toBe(1);
  });
});

describe("apiParams", () => {
  it("converte valores e página para a API", () => {
    const p = apiParams({ ...EMPTY_STATE, q: "x", min: "1.500,50", page: 2 }, 20)!;

    expect(p.get("min_value")).toBe("1500.50");
    expect(p.get("offset")).toBe("20");
    expect(p.get("limit")).toBe("20");
    expect(p.has("max_value")).toBe(false);
  });

  it("recusa valor inválido", () => {
    expect(apiParams({ ...EMPTY_STATE, max: "abc" }, 20)).toBeNull();
  });
});

describe("hasSearch", () => {
  it("é verdadeiro com termo ou qualquer filtro", () => {
    expect(hasSearch(EMPTY_STATE)).toBe(false);
    expect(hasSearch({ ...EMPTY_STATE, organ: "FMS" })).toBe(true);
    expect(hasSearch({ ...EMPTY_STATE, page: 2 })).toBe(false);
  });
});

describe("exportUrl", () => {
  it("leva os filtros sem paginação", () => {
    const url = exportUrl({ ...EMPTY_STATE, q: "merenda", organ: "SEMED", min: "1.000", page: 3 }, "csv")!;
    const p = new URLSearchParams(url.split("?")[1]);

    expect(url.startsWith("/api/v1/acts/export?")).toBe(true);
    expect(p.get("format")).toBe("csv");
    expect(p.get("q")).toBe("merenda");
    expect(p.get("organ")).toBe("SEMED");
    expect(p.get("min_value")).toBe("1000");
    expect(p.has("limit") || p.has("offset")).toBe(false);
  });

  it("não monta link com valor inválido", () => {
    expect(exportUrl({ ...EMPTY_STATE, min: "abc" }, "json")).toBeNull();
  });
});

describe("feedUrl", () => {
  it("leva os filtros sem paginação", () => {
    const url = feedUrl({ ...EMPTY_STATE, q: "merenda", organ: "SEMED", page: 4 })!;
    const p = new URLSearchParams(url.split("?")[1]);

    expect(url.startsWith("/api/v1/feeds/acts?")).toBe(true);
    expect(p.get("q")).toBe("merenda");
    expect(p.get("organ")).toBe("SEMED");
    expect(p.has("limit") || p.has("offset") || p.has("format")).toBe(false);
  });

  it("não monta link com valor inválido", () => {
    expect(feedUrl({ ...EMPTY_STATE, max: "x" })).toBeNull();
  });
});
