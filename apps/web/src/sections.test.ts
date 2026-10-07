import { describe, expect, it } from "vitest";
import { DEV_GROUP, SECTION_GROUPS, isCurrentSection } from "./sections";

describe("seções do site", () => {
  it("agrupa a navegação pelo que a pessoa quer saber", () => {
    expect(SECTION_GROUPS.map((g) => g.title)).toEqual(["Pessoas", "Dinheiro", "Leis", "Para verificar"]);
    expect(DEV_GROUP.links.map((l) => l.href)).toEqual(["/dados", "/mcp"]);
  });

  it("leva a todas as páginas do site", () => {
    const paths = [...SECTION_GROUPS, DEV_GROUP].flatMap((g) => g.links.map((l) => l.href.split(/[?#]/)[0]));
    for (const page of ["/pessoal", "/agentes", "/paineis", "/federal", "/tce", "/proposicoes", "/padroes", "/dados", "/mcp"]) {
      expect(paths).toContain(page);
    }
  });

  it("aponta os links de padrão para padrões que existem na API", () => {
    const ids = SECTION_GROUPS.flatMap((g) => g.links).map((l) => l.href.split("#padrao-")[1]).filter(Boolean);
    expect(ids).toEqual(["fracionamento_dispensa", "aditivo_acima_do_limite", "empresa_nova_contratada"]);
  });

  it("marca a seção atual, inclusive nas páginas de uma proposição", () => {
    expect(isCurrentSection("/dados", "/dados")).toBe(true);
    expect(isCurrentSection("/proposicoes", "/proposicoes/1877-2023")).toBe(true);
    expect(isCurrentSection("/dados", "/")).toBe(false);
  });

  it("não marca como atual link para busca filtrada nem para trecho de página", () => {
    expect(isCurrentSection("/?tipo=lei", "/")).toBe(false);
    expect(isCurrentSection("/padroes#padrao-fracionamento_dispensa", "/padroes")).toBe(false);
  });
});
