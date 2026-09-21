import { ActType } from "./api";

export const TYPE_LABEL: Record<ActType, string> = {
  nomeacao: "Nomeação", exoneracao: "Exoneração", contrato: "Contrato",
  aditivo: "Aditivo", licitacao: "Licitação", dispensa: "Sem licitação",
  decreto: "Decreto", lei: "Lei", portaria: "Portaria", resolucao: "Resolução",
  despacho: "Despacho", edital: "Edital", ata: "Ata", outro: "Outro",
};

export function formatCnpj(digits: string) {
  const d = digits.replace(/\D/g, "");
  if (d.length !== 14) return digits;
  return `${d.slice(0, 2)}.${d.slice(2, 5)}.${d.slice(5, 8)}/${d.slice(8, 12)}-${d.slice(12)}`;
}

export function formatCents(cents: number) {
  return (cents / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}
