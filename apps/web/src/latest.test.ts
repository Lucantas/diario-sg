import { describe, expect, it } from "vitest";
import { LatestEdition } from "./api";
import { formatCompactCents } from "./panels";
import { actsLabel, editionHref, editionTitle, homeEdition, longDate, relativeDay, typeCountLabel } from "./latest";

const plain = (s: string) => s.replace(/\s/g, " ");

const edition = (over: Partial<LatestEdition>): LatestEdition => ({
  gazette_id: "3f2b8c1e-4d5a-4b6c-8d7e-9f0a1b2c3d4e", source: "diario_prefeitura", source_name: "", published_at: "2026-10-07",
  edition_number: "1612", is_extra: false, total_acts: 41, types: [], ...over,
});

describe("edição da home", () => {
  it("escolhe a primeira edição que tem atos", () => {
    const empty = edition({ gazette_id: "a", total_acts: 0 });
    const full = edition({ gazette_id: "b" });

    expect(homeEdition([empty, full])?.gazette_id).toBe("b");
    expect(homeEdition([empty])).toBeNull();
  });

  it("formata o título com milhar, extra e edição sem número", () => {
    expect(editionTitle(edition({}))).toBe("Edição 1.612");
    expect(editionTitle(edition({ edition_number: "1613", is_extra: true }))).toBe("Edição 1.613 (extra)");
    expect(editionTitle(edition({ edition_number: "" }))).toBe("Edição sem número");
  });

  it("conta cada tipo no singular ou plural e soma o valor quando há", () => {
    expect(typeCountLabel("nomeacao", 14, 0)).toBe("14 nomeações");
    expect(typeCountLabel("decreto", 1, 0)).toBe("1 decreto");
    expect(plain(typeCountLabel("contrato", 3, 210_000_000))).toBe("3 contratos, R$ 2,1 mi");
    expect(typeCountLabel("aditivo", 2, 48_000_000)).toBe(`2 aditivos, ${formatCompactCents(48_000_000)}`);
  });

  it("chama o link de todos os atos pelo total", () => {
    expect(actsLabel(41)).toBe("Todos os 41 atos desta edição");
    expect(actsLabel(1)).toBe("o único ato desta edição");
  });

  it("diz o dia relativo a hoje", () => {
    const today = new Date(2026, 9, 7, 15, 0);

    expect(relativeDay("2026-10-07", today)).toBe("hoje");
    expect(relativeDay("2026-10-06", today)).toBe("ontem");
    expect(relativeDay("2026-10-02", today)).toBe("sexta");
    expect(relativeDay("2026-09-20", today)).toBe("20 de setembro de 2026");
    expect(longDate("2026-01-05")).toBe("5 de janeiro de 2026");
  });

  it("leva para a busca dentro da edição, com ou sem tipo", () => {
    expect(editionHref("abc")).toBe("/?edicao=abc");
    expect(editionHref("abc", "nomeacao")).toBe("/?tipo=nomeacao&edicao=abc");
  });
});
