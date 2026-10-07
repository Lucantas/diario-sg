import { describe, expect, it } from "vitest";
import { ActHit } from "./api";
import { canToggleExact, editionLabel, isExactPhrase, relaxations, resultsHeading, toggleExact } from "./results";
import { EMPTY_STATE } from "./searchState";

const edition = "3f2b8c1e-4d5a-4b6c-8d7e-9f0a1b2c3d4e";
const hit = { edition_number: "1772", is_extra: false, published_at: "2026-09-23" } as ActHit;

describe("título dos resultados", () => {
  it("diz o termo buscado e marca a frase exata", () => {
    expect(resultsHeading({ ...EMPTY_STATE, q: "locação de veículos" }, 1284, hit))
      .toEqual({ count: "1.284 atos encontrados", scope: "para “locação de veículos”" });
    expect(resultsHeading({ ...EMPTY_STATE, q: "\"locação de veículos\"" }, 1, hit))
      .toEqual({ count: "1 ato encontrado", scope: "para “locação de veículos”, frase exata" });
  });

  it("sem termo, conta pelo tipo em todas as edições", () => {
    expect(resultsHeading({ ...EMPTY_STATE, type: "nomeacao" }, 212408, hit))
      .toEqual({ count: "212.408 nomeações", scope: "em todas as edições" });
  });

  it("dentro de uma edição, diz qual e de quando", () => {
    expect(resultsHeading({ ...EMPTY_STATE, edition, type: "nomeacao" }, 2, hit))
      .toEqual({ count: "2 nomeações", scope: "na edição 1.772, de 23 de setembro de 2026" });
    expect(resultsHeading({ ...EMPTY_STATE, edition }, 0, undefined)).toEqual({ count: "0 atos", scope: "nesta edição" });
    expect(editionLabel(hit)).toBe("Edição 1.772");
    expect(editionLabel(undefined)).toBe("Uma edição");
  });
});

describe("frase exata", () => {
  it("só aparece com duas palavras ou mais e sem operadores", () => {
    expect(canToggleExact("locação de veículos")).toBe(true);
    expect(canToggleExact("\"locação de veículos\"")).toBe(true);
    expect(canToggleExact("merenda")).toBe(false);
    expect(canToggleExact("merenda OU alimentação")).toBe(false);
    expect(canToggleExact("limpeza -urbana")).toBe(false);
  });

  it("põe e tira as aspas", () => {
    expect(toggleExact("locação de veículos")).toBe("\"locação de veículos\"");
    expect(toggleExact("\"locação de veículos\"")).toBe("locação de veículos");
    expect(isExactPhrase(" \"a b\" ")).toBe(true);
  });
});

describe("alternativas para a busca vazia", () => {
  it("tira um filtro por vez, começando pela frase exata", () => {
    const s = { ...EMPTY_STATE, q: "\"locação de veículos\"", type: "contrato" as const, organ: "SEMSA" };

    const got = relaxations(s, "Uma edição");

    expect(got.map((r) => r.label)).toEqual([
      "Sem exigir a frase exata", "Sem o filtro Contratos", "Sem o filtro SEMSA", "Com qualquer uma das palavras: locação OU veículos",
    ]);
    expect(got[0].state.q).toBe("locação de veículos");
    expect(got[1].state).toEqual({ ...s, type: "" });
  });

  it("com duas palavras ou mais, sugere qualquer uma delas", () => {
    const got = relaxations({ ...EMPTY_STATE, q: "locação veículos" }, "Uma edição");

    expect(got).toEqual([{ label: "Com qualquer uma das palavras: locação OU veículos", state: { ...EMPTY_STATE, q: "locação OU veículos" } }]);
  });

  it("não sugere nada para uma palavra só", () => {
    expect(relaxations({ ...EMPTY_STATE, q: "xyzw" }, "Uma edição")).toEqual([]);
  });

  it("nomeia a edição do filtro", () => {
    expect(relaxations({ ...EMPTY_STATE, q: "merenda", edition }, "Edição 1.772").map((r) => r.label)).toEqual(["Sem o filtro Edição 1.772"]);
  });
});
