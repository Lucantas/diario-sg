import { describe, expect, it } from "vitest";
import { patternSearchHref } from "./patterns";

describe("patternSearchHref", () => {
  it("abre a busca do site com tipo, período e diário", () => {
    expect(patternSearchHref({ type: "nomeacao", from: "2020-08-01", to: "2020-08-31", source: "diario_prefeitura" }))
      .toBe("/?tipo=nomeacao&de=2020-08-01&ate=2020-08-31&fonte=diario_prefeitura");
  });
});
