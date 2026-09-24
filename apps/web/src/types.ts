import { ActType, Phase, Source } from "./api";

export const SOURCE_LABEL: Record<Source, string> = {
  diario_prefeitura: "Prefeitura",
  diario_camara: "Câmara",
};

export const SOURCE_NAME: Record<Source, string> = {
  diario_prefeitura: "Diário Oficial do Município de São Gonçalo",
  diario_camara: "Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo",
};

export const TYPE_LABEL: Record<ActType, string> = {
  nomeacao: "Nomeação", exoneracao: "Exoneração", contrato: "Contrato",
  aditivo: "Aditivo", licitacao: "Licitação", dispensa: "Sem licitação",
  decreto: "Decreto", lei: "Lei", portaria: "Portaria", resolucao: "Resolução",
  despacho: "Despacho", edital: "Edital", ata: "Ata", corrigenda: "Corrigenda",
  prestacao_contas: "Prestação de contas", outro: "Outro",
};

export const PHASES: Phase[] = [
  "licitacao", "homologacao", "ata_registro_precos", "dispensa", "contrato",
  "fiscal", "aditivo", "ajuste_contas", "rescisao", "outro",
];

export const PHASE_LABEL: Record<Phase, string> = {
  licitacao: "Licitação",
  homologacao: "Homologação",
  ata_registro_precos: "Ata de registro de preços",
  dispensa: "Dispensa ou inexigibilidade",
  contrato: "Contrato",
  fiscal: "Fiscal do contrato",
  aditivo: "Aditivo ou apostilamento",
  ajuste_contas: "Ajuste de contas",
  rescisao: "Rescisão ou distrato",
  outro: "Outros atos",
};

export function formatCnpj(digits: string) {
  const d = digits.replace(/\D/g, "");
  if (d.length !== 14) return digits;
  return `${d.slice(0, 2)}.${d.slice(2, 5)}.${d.slice(5, 8)}/${d.slice(8, 12)}-${d.slice(12)}`;
}

export function formatCents(cents: number) {
  return (cents / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}
