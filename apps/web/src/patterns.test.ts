import { describe, expect, it } from "vitest";
import { highlights, patternSearchHref } from "./patterns";

describe("patternSearchHref", () => {
  it("abre a busca do site com tipo, período e diário", () => {
    expect(patternSearchHref({ type: "nomeacao", from: "2020-08-01", to: "2020-08-31", source: "diario_prefeitura" }))
      .toBe("/?tipo=nomeacao&de=2020-08-01&ate=2020-08-31&fonte=diario_prefeitura");
  });
});

describe("casos em destaque na home", () => {
  it("conta os casos e leva ao padrão na página de padrões", () => {
    const got = highlights([
      { pattern_id: "a", pattern_title: "padrão a", title: "a-0", detail: "detalhe", findings: 3 },
      { pattern_id: "b", pattern_title: "padrão b", title: "b-0", detail: "detalhe", findings: 1 },
    ]);

    expect(got).toEqual([
      { pattern: "padrão a", title: "a-0", detail: "detalhe", count: "3 casos na base", href: "/padroes#padrao-a" },
      { pattern: "padrão b", title: "b-0", detail: "detalhe", count: "1 caso na base", href: "/padroes#padrao-b" },
    ]);
  });
});
