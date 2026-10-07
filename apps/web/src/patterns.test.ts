import { describe, expect, it } from "vitest";
import { Pattern } from "./api";
import { highlights, patternSearchHref } from "./patterns";

describe("patternSearchHref", () => {
  it("abre a busca do site com tipo, período e diário", () => {
    expect(patternSearchHref({ type: "nomeacao", from: "2020-08-01", to: "2020-08-31", source: "diario_prefeitura" }))
      .toBe("/?tipo=nomeacao&de=2020-08-01&ate=2020-08-31&fonte=diario_prefeitura");
  });
});

describe("casos em destaque na home", () => {
  const finding = (title: string) => ({ title, detail: `detalhe de ${title}`, acts: [], search: null, link: null });
  const pattern = (id: string, n: number): Pattern => ({
    id, title: `padrão ${id}`, rule: "", caveat: "", findings: Array.from({ length: n }, (_, i) => finding(`${id}-${i}`)),
  });

  it("pega o primeiro caso de cada padrão que tem caso, até o limite", () => {
    const got = highlights([pattern("vazio", 0), pattern("a", 3), pattern("b", 1), pattern("c", 2)], 2);

    expect(got).toEqual([
      { pattern: "padrão a", title: "a-0", detail: "detalhe de a-0", count: "3 casos na base", href: "/padroes#padrao-a" },
      { pattern: "padrão b", title: "b-0", detail: "detalhe de b-0", count: "1 caso na base", href: "/padroes#padrao-b" },
    ]);
  });
});
