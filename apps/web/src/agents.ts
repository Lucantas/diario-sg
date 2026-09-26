import { AgentRole, PoliticalAgent, PoliticalAgentsReport } from "./api";

const ROLE_LABELS: Record<AgentRole, string> = {
  prefeito: "Prefeito",
  vice_prefeito: "Vice-prefeito",
  secretario: "Secretário municipal",
  procurador_geral: "Procurador-Geral do Município",
  vereador: "Vereador",
};

export const ROLES = Object.keys(ROLE_LABELS) as AgentRole[];

export function roleLabel(role: AgentRole): string {
  return ROLE_LABELS[role];
}

export function inOffice(agent: PoliticalAgent, coverage: PoliticalAgentsReport["coverage"]): boolean {
  return coverage.some((c) => c.body === agent.body && c.to === agent.last);
}

export interface AgentFilter {
  role: AgentRole | "";
  query: string;
  past: boolean;
}

function fold(s: string): string {
  return s.normalize("NFD").replace(/[̀-ͯ]/g, "").toUpperCase();
}

export function visibleAgents(agents: PoliticalAgent[], coverage: PoliticalAgentsReport["coverage"], f: AgentFilter): PoliticalAgent[] {
  const q = fold(f.query.trim());
  return agents.filter((a) =>
    (f.role === "" || a.role === f.role) &&
    (f.past || inOffice(a, coverage)) &&
    (q === "" || [a.name, a.parliamentary_name ?? "", ...a.offices].some((s) => fold(s).includes(q))));
}

export function agentSubtitle(a: PoliticalAgent): string {
  if (a.role === "vereador") return [a.parliamentary_name, a.party].filter(Boolean).join(" · ");
  return a.offices.join("; ");
}

export function diarioSearchUrl(name: string): string {
  return `/?${new URLSearchParams({ q: `"${name}"` }).toString()}`;
}
