export interface SectionLink {
  href: string;
  label: string;
}

export interface SectionGroup {
  title: string;
  links: SectionLink[];
}

export const SECTION_GROUPS: SectionGroup[] = [
  {
    title: "Pessoas",
    links: [
      { href: "/?tipo=nomeacao", label: "Quem foi nomeado ou exonerado" },
      { href: "/pessoal", label: "Quantos servidores a prefeitura tem e quanto custam" },
      { href: "/agentes", label: "Quanto ganham prefeito, secretários e vereadores" },
    ],
  },
  {
    title: "Dinheiro",
    links: [
      { href: "/paineis", label: "Quem a prefeitura mais contrata" },
      { href: "/federal", label: "Quanto dinheiro federal chega a São Gonçalo" },
      { href: "/tce", label: "O que o TCE-RJ diz das contas da prefeitura" },
    ],
  },
  {
    title: "Leis",
    links: [
      { href: "/proposicoes", label: "Que projetos tramitam na Câmara" },
      { href: "/?tipo=decreto", label: "Que decretos a prefeitura publicou" },
      { href: "/?tipo=lei", label: "Que leis foram sancionadas" },
    ],
  },
  {
    title: "Para verificar",
    links: [
      { href: "/padroes#padrao-fracionamento_dispensa", label: "Dispensas que, somadas, passam do limite" },
      { href: "/padroes#padrao-aditivo_acima_do_limite", label: "Aditivos acima de 25% do contrato" },
      { href: "/padroes#padrao-empresa_nova_contratada", label: "Empresas contratadas logo depois de abertas" },
    ],
  },
];

export const DEV_GROUP: SectionGroup = {
  title: "Para quem programa",
  links: [
    { href: "/dados", label: "Dados abertos" },
    { href: "/mcp", label: "Servidor MCP" },
  ],
};

export function isCurrentSection(href: string, path: string): boolean {
  if (href.includes("?") || href.includes("#")) return false;
  return path === href || path.startsWith(href + "/");
}
