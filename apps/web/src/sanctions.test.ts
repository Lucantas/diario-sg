import { describe, expect, it } from "vitest";
import { Sanction } from "./api";
import { listedOnLabel, sanctionOrgan, sanctionPeriod, sanctionStateLabel, sanctionsSummary } from "./sanctions";

const base: Sanction = {
  register: "CEIS", code: "1", cnpj: "28926250000176", name: "EMPRESA X", category: "Suspensão",
  starts_at: "2025-10-24", ends_at: "2027-10-23", published_at: "2025-10-27", process: "1/2025",
  organ: "PREFEITURA DO RIO DE JANEIRO", organ_uf: "RJ", sphere: "MUNICIPAL", scope: "No órgão sancionador",
  legal_basis: "LEI 14133", fine_cents: null, first_seen: "2026-09-25", last_seen: "2026-09-25", state: "no_cadastro",
};

describe("sanções", () => {
  it("descreve o estado de cada sanção", () => {
    expect(sanctionStateLabel(base)).toBe("No cadastro, até 23/10/2027");
    expect(sanctionStateLabel({ ...base, ends_at: null })).toBe("No cadastro, sem data final");
    expect(sanctionStateLabel({ ...base, state: "prazo_encerrado", ends_at: "2026-01-31" })).toBe("Prazo encerrado em 31/01/2026");
    expect(sanctionStateLabel({ ...base, state: "fora_do_cadastro", last_seen: "2026-09-20" })).toBe("Saiu do cadastro depois de 20/09/2026");
  });

  it("mostra o período com as datas que houver", () => {
    expect(sanctionPeriod(base)).toBe("24/10/2025 a 23/10/2027");
    expect(sanctionPeriod({ ...base, ends_at: null })).toBe("desde 24/10/2025");
    expect(sanctionPeriod({ ...base, starts_at: null, ends_at: null })).toBe("sem período informado");
  });

  it("junta o órgão, a UF e a esfera", () => {
    expect(sanctionOrgan(base)).toBe("PREFEITURA DO RIO DE JANEIRO (RJ, municipal)");
    expect(sanctionOrgan({ ...base, organ_uf: "", sphere: "" })).toBe("PREFEITURA DO RIO DE JANEIRO");
    expect(sanctionOrgan({ ...base, organ: "Prefeitura de Joinville (SC)", organ_uf: "SC" })).toBe("Prefeitura de Joinville (SC) (municipal)");
  });

  it("resume quantas estão no cadastro", () => {
    expect(sanctionsSummary([base])).toBe("1 sanção no cadastro. Clique em cada uma para ver o processo e a fundamentação.");
    expect(sanctionsSummary([base, { ...base, state: "prazo_encerrado" }])).toBe("2 sanções, 1 no cadastro. Clique em cada uma para ver o processo e a fundamentação.");
  });

  it("junta as datas dos arquivos sem repetir", () => {
    expect(listedOnLabel({ CEIS: "2026-09-25", CNEP: "2026-09-25" })).toBe("25/09/2026");
    expect(listedOnLabel({ CEIS: "2026-09-25", CNEP: "2026-09-24" })).toBe("24/09/2026 e 25/09/2026");
  });
});
