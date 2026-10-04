import { describe, expect, it } from "vitest";
import { SECTION_GROUPS, isCurrentSection } from "./sections";

describe("seções do site", () => {
  it("agrupa as nove seções em Painéis, Câmara e Dados e ferramentas", () => {
    expect(SECTION_GROUPS.map((g) => g.title)).toEqual(["Painéis", "Câmara", "Dados e ferramentas"]);
    expect(SECTION_GROUPS.flatMap((g) => g.links.map((l) => l.href))).toEqual([
      "/paineis", "/pessoal", "/tce", "/federal", "/agentes", "/proposicoes", "/dados", "/mcp", "/padroes",
    ]);
  });

  it("dá descrição a todo link", () => {
    for (const link of SECTION_GROUPS.flatMap((g) => g.links)) expect(link.description).not.toBe("");
  });

  it("marca a seção atual, inclusive nas páginas de uma proposição", () => {
    expect(isCurrentSection("/dados", "/dados")).toBe(true);
    expect(isCurrentSection("/proposicoes", "/proposicoes/1877-2023")).toBe(true);
    expect(isCurrentSection("/dados", "/")).toBe(false);
  });
});
