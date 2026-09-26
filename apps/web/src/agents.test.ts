import { describe, expect, it } from "vitest";
import { PoliticalAgent, PoliticalAgentsReport } from "./api";
import { agentSubtitle, diarioSearchUrl, inOffice, roleLabel, visibleAgents } from "./agents";

function agent(p: Partial<PoliticalAgent>): PoliticalAgent {
  return { body: "prefeitura", role: "secretario", name: "FULANO", offices: ["SECRETARIA MUNICIPAL DE SAUDE"], first: "2025-01", last: "2026-08", months: [], ...p };
}

const coverage: PoliticalAgentsReport["coverage"] = [
  { body: "prefeitura", from: "2010-10", to: "2026-08" },
  { body: "camara", from: "2017-01", to: "2026-08" },
];

describe("agentes políticos", () => {
  it("nomeia cada cargo", () => {
    expect(roleLabel("vice_prefeito")).toBe("Vice-prefeito");
    expect(roleLabel("procurador_geral")).toBe("Procurador-Geral do Município");
  });

  it("considera em exercício quem está no último mês da folha do órgão", () => {
    expect(inOffice(agent({}), coverage)).toBe(true);
    expect(inOffice(agent({ last: "2024-12" }), coverage)).toBe(false);
  });

  it("filtra por cargo, nome e exercício", () => {
    const agents = [agent({ name: "ANA" }), agent({ name: "BIA", last: "2020-01" }), agent({ name: "CAIO", role: "vereador", body: "camara", parliamentary_name: "CAIÃO" })];
    expect(visibleAgents(agents, coverage, { role: "", query: "", past: false }).map((a) => a.name)).toEqual(["ANA", "CAIO"]);
    expect(visibleAgents(agents, coverage, { role: "vereador", query: "caiao", past: true }).map((a) => a.name)).toEqual(["CAIO"]);
    expect(visibleAgents(agents, coverage, { role: "", query: "bia", past: true }).map((a) => a.name)).toEqual(["BIA"]);
  });

  it("descreve a lotação e o partido", () => {
    expect(agentSubtitle(agent({ role: "vereador", parliamentary_name: "CACAU", party: "MDB", offices: ["VEREADOR CACAU"] }))).toBe("CACAU · MDB");
    expect(agentSubtitle(agent({ offices: ["A", "B"] }))).toBe("A; B");
  });

  it("busca o nome exato no Diário", () => {
    expect(diarioSearchUrl("JOÃO DA SILVA")).toBe("/?q=%22JO%C3%83O+DA+SILVA%22");
  });
});
