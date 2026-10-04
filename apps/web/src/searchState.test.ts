import { describe, expect, it } from "vitest";
import {
  EMPTY_STATE, SearchState, activeFilters, isRangeFilter, alertFilterNames, alertFilters, apiParams, canAlert, exportUrl, feedUrl, hasSearch, parseBRL, queryFromState, stateFromQuery, withSource, withoutFilter, withoutFilters,
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
      q: "limpeza OU coleta", source: "diario_prefeitura", type: "contrato", organ: "SEMED", theme: "meio_ambiente", from: "2024-01-01",
      to: "2024-12-31", min: "1.000,00", max: "", page: 3,
    };

    expect(stateFromQuery(queryFromState(s))).toEqual(s);
  });

  it("usa nomes em português e omite o vazio", () => {
    expect(queryFromState({ ...EMPTY_STATE, q: "merenda", organ: "SEMED" })).toBe("?q=merenda&orgao=SEMED");
    expect(queryFromState(EMPTY_STATE)).toBe("");
  });

  it("ignora página inválida, tipo e diário desconhecidos", () => {
    expect(stateFromQuery("?pagina=-2&tipo=bobagem&fonte=tce")).toEqual(EMPTY_STATE);
    expect(stateFromQuery("?tipo=toString&fonte=constructor")).toEqual(EMPTY_STATE);
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

  it("filtra pelo diário", () => {
    expect(apiParams({ ...EMPTY_STATE, source: "diario_camara" }, 20)!.get("source")).toBe("diario_camara");
    expect(apiParams(EMPTY_STATE, 20)!.has("source")).toBe(false);
    expect(queryFromState({ ...EMPTY_STATE, source: "diario_camara" })).toBe("?fonte=diario_camara");
  });
});

describe("hasSearch", () => {
  it("é verdadeiro com termo ou qualquer filtro", () => {
    expect(hasSearch(EMPTY_STATE)).toBe(false);
    expect(hasSearch({ ...EMPTY_STATE, organ: "FMS" })).toBe(true);
    expect(hasSearch({ ...EMPTY_STATE, source: "diario_camara" })).toBe(true);
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

describe("link com Câmara e órgão", () => {
  it("descarta o órgão, que só vale para a Prefeitura", () => {
    expect(stateFromQuery("?fonte=diario_camara&orgao=SEMED")).toEqual({ ...EMPTY_STATE, source: "diario_camara" });
  });
});

describe("withSource", () => {
  it("tira o órgão ao escolher a Câmara e volta à primeira página", () => {
    const s = { ...EMPTY_STATE, organ: "SEMED", page: 3 };

    expect(withSource(s, "diario_camara")).toEqual({ ...EMPTY_STATE, source: "diario_camara" });
    expect(withSource(s, "diario_prefeitura")).toEqual({ ...EMPTY_STATE, source: "diario_prefeitura", organ: "SEMED" });
    expect(withSource(s, "")).toEqual({ ...EMPTY_STATE, organ: "SEMED" });
  });
});

describe("alerta da busca", () => {
  it("aceita só filtros, sem termo", () => {
    const s = { ...EMPTY_STATE, type: "licenca_ambiental" as const, organ: "SEMMATRAN" };

    expect(canAlert(s)).toBe(true);
    expect(alertFilters(s)).toEqual({ source: "", type: "licenca_ambiental", organ: "SEMMATRAN", theme: "" });
    expect(alertFilterNames(s)).toBe("licença ambiental · SEMMATRAN");
  });

  it("recusa termo curto e busca sem termo nem filtro", () => {
    expect(canAlert({ ...EMPTY_STATE, q: "ab", type: "contrato" })).toBe(false);
    expect(canAlert({ ...EMPTY_STATE, from: "2024-01-01" })).toBe(false);
    expect(canAlert({ ...EMPTY_STATE, q: "merenda" })).toBe(true);
  });

  it("descreve o diário", () => {
    expect(alertFilterNames({ ...EMPTY_STATE, source: "diario_camara" })).toBe("Diário da Câmara");
  });
});

describe("tema", () => {
  const s = { ...EMPTY_STATE, q: "loteamento", theme: "meio_ambiente" };

  it("vai para a URL, a API, a exportação, o RSS e o alerta", () => {
    expect(queryFromState(s)).toBe("?q=loteamento&tema=meio_ambiente");
    expect(apiParams(s, 20)?.get("theme")).toBe("meio_ambiente");
    expect(exportUrl(s, "csv")).toContain("theme=meio_ambiente");
    expect(feedUrl(s)).toContain("theme=meio_ambiente");
    expect(alertFilterNames(s)).toBe("meio ambiente");
    expect(hasSearch({ ...EMPTY_STATE, theme: "meio_ambiente" })).toBe(true);
  });

  it("ignora tema desconhecido", () => {
    expect(stateFromQuery("?tema=saude").theme).toBe("");
  });
});

describe("filtros ativos", () => {
  const full: SearchState = {
    q: "merenda", source: "diario_prefeitura", type: "contrato", organ: "SEMSA", theme: "meio_ambiente",
    from: "2024-01-05", to: "2024-12-31", min: "1.000", max: "50.000,00", page: 3,
  };

  it("lista cada filtro com o rótulo do chip, sem a busca", () => {
    expect(activeFilters(full).map((f) => f.label)).toEqual([
      "Contratos", "Diário da Prefeitura", "SEMSA", "Meio ambiente",
      "a partir de 05/01/2024", "até 31/12/2024", "valor a partir de R$ 1.000", "valor até R$ 50.000,00",
    ]);
  });

  it("separa período e valor dos filtros de escopo", () => {
    expect(activeFilters(full).filter((f) => isRangeFilter(f.key)).map((f) => f.key)).toEqual(["from", "to", "min", "max"]);
  });

  it("não lista nada sem filtro", () => {
    expect(activeFilters({ ...EMPTY_STATE, q: "merenda" })).toEqual([]);
  });

  it("remove um filtro e volta para a primeira página", () => {
    const next = withoutFilter(full, "organ");
    expect(next.organ).toBe("");
    expect(next.type).toBe("contrato");
    expect(next.page).toBe(1);
  });

  it("limpa os filtros e mantém a busca", () => {
    expect(withoutFilters(full)).toEqual({ ...EMPTY_STATE, q: "merenda" });
  });
});
