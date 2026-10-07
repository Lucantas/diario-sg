import { describe, expect, it } from "vitest";
import { ActHit, Bill, CompanyResponse, EntityResponse, Organ, Suggestion } from "./api";
import {
  cnpjDigits, companyFicha, entityFicha, exactMatch, looksLikeNumber, matchOrgan, organFicha, suggestGroups,
} from "./discovery";

const plain = (s: string) => s.replace(/\s/g, " ");
const company: Suggestion = { kind: "cnpj", key: "12345678000190", label: "12.345.678/0001-90", name: "EXEMPLO LOCAÇÕES LTDA", acts: 107 };
const process: Suggestion = { kind: "processo", key: "43212024", label: "4.321/2024", name: "", acts: 5 };
const organs: Organ[] = [
  { acronym: "SEMSA", name: "Secretaria Municipal de Saúde", acts: 48112 },
  { acronym: "SEMED", name: "Secretaria Municipal de Educação", acts: 31904 },
];
const hit = (published_at: string) => ({ published_at } as ActHit);

describe("reconhecer o que foi digitado", () => {
  it("lê CNPJ com ou sem máscara", () => {
    expect(cnpjDigits("12.345.678/0001-90")).toBe("12345678000190");
    expect(cnpjDigits("12345678000190")).toBe("12345678000190");
    expect(cnpjDigits("4.321/2024")).toBeNull();
    expect(cnpjDigits("cnpj 12345678000190")).toBeNull();
  });

  it("trata como número o que começa com dígito ou com processo/contrato", () => {
    expect(looksLikeNumber("4.321/2024")).toBe(true);
    expect(looksLikeNumber("processo nº 4.321/2024")).toBe(true);
    expect(looksLikeNumber("processo de licitação")).toBe(false);
    expect(looksLikeNumber("limpeza")).toBe(false);
  });

  it("escolhe a sugestão exata: o CNPJ digitado ou o primeiro processo ou contrato", () => {
    const other = { ...company, key: "99999999000199" };
    expect(exactMatch("12.345.678/0001-90", [other, company])).toBe(company);
    expect(exactMatch("4.321/2024", [process])).toBe(process);
    expect(exactMatch("limpeza", [company])).toBeNull();
  });

  it("acha o órgão pela sigla, pelo nome ou por parte longa do nome que só ele tem, sem acento", () => {
    expect(matchOrgan("semsa", organs)?.acronym).toBe("SEMSA");
    expect(matchOrgan("secretaria municipal de saude", organs)?.acronym).toBe("SEMSA");
    expect(matchOrgan("educação", organs)?.acronym).toBe("SEMED");
    expect(matchOrgan("saúde", organs)).toBeNull();
    expect(matchOrgan("secretaria municipal", organs)).toBeNull();
  });
});

describe("ficha acima dos resultados", () => {
  it("resume a empresa com situação, atos, valores e pagamentos", () => {
    const c = {
      registry: { name: "EXEMPLO LOCAÇÕES LTDA", status: "ATIVA" }, total_value_cents: 410_000_000,
      acts: [hit("2026-03-01"), hit("2011-05-02")], payments: [{ paid_cents: 200_000_000 }, { paid_cents: 130_000_000 }],
    } as unknown as CompanyResponse;

    const f = companyFicha(company, c);

    expect(f.kind).toBe("Empresa · CNPJ 12.345.678/0001-90");
    expect(f.title).toBe("EXEMPLO LOCAÇÕES LTDA");
    expect(f.facts.map(plain)).toEqual([
      "Situação cadastral: ativa", "107 atos nos Diários, de 2011 a 2026", "R$ 4,1 mi citados em atos", "R$ 3,3 mi pagos (TCE-RJ)",
    ]);
    expect(f.href).toBe("/empresa/12345678000190");
  });

  it("sem a ficha completa, mostra o que a sugestão já sabe", () => {
    expect(companyFicha(company, null).facts).toEqual(["107 atos nos Diários"]);
  });

  it("resume o processo com período, órgãos e o caminho da contratação", () => {
    const e = {
      total_acts: 5, acts: [hit("2025-03-10"), hit("2024-01-15")], organs: [{ organ: "SEMSA" }, { organ: "SEMAD" }],
      count_by_phase: { licitacao: 1, contrato: 1, fiscal: 1, aditivo: 2 },
    } as unknown as EntityResponse;

    const f = entityFicha(process, e);

    expect(f).toEqual({
      kind: "Processo administrativo", title: "4.321/2024",
      facts: ["5 atos, de jan. 2024 a mar. 2025", "SEMSA e SEMAD", "Licitação → contrato → fiscal do contrato → aditivo ou apostilamento"],
      cta: "Ver a linha do tempo →", href: "/processo/4.321-2024",
    });
  });

  it("leva o órgão para a busca filtrada", () => {
    expect(organFicha(organs[0])).toEqual({
      kind: "Órgão da Prefeitura", title: "SEMSA · Secretaria Municipal de Saúde", facts: ["48.112 atos nos Diários"],
      cta: "Filtrar a busca por este órgão →", href: "/?orgao=SEMSA",
    });
  });
});

describe("sugestões enquanto digita", () => {
  it("agrupa empresas, números, órgãos, tipos e proposições, até três por grupo", () => {
    const bills = [{ process: "1877/2023", document: "PL 087/2023", summary: "Coleta seletiva" }] as Bill[];

    const got = suggestGroups("sa", [company, process], organs, bills);

    expect(got.map((g) => g.title)).toEqual(["Empresas", "Processos e contratos", "Órgãos", "Proposições"]);
    expect(got[0].items[0]).toEqual({ label: "EXEMPLO LOCAÇÕES LTDA", meta: "12.345.678/0001-90 · 107 atos", href: "/empresa/12345678000190" });
    expect(got[1].items[0]).toEqual({ label: "Processo 4.321/2024", meta: "5 atos", href: "/processo/4.321-2024" });
    expect(got[2].items.map((i) => i.href)).toEqual(["/?orgao=SEMSA"]);
    expect(got[3].items[0].href).toBe("/proposicoes/1877-2023");
  });

  it("acha tipos de ato pelo nome, sem acento", () => {
    const got = suggestGroups("nomeacoes", [], [], []);

    expect(got).toEqual([{ title: "Tipos de ato", items: [{ label: "Nomeações", meta: "", href: "/?tipo=nomeacao" }] }]);
  });

  it("não sugere nada com menos de duas letras", () => {
    expect(suggestGroups("s", [company], organs, [])).toEqual([]);
  });
});
