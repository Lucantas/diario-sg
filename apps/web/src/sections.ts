export interface SectionLink {
  href: string;
  label: string;
  description: string;
}

export interface SectionGroup {
  title: string;
  links: SectionLink[];
}

export const SECTION_GROUPS: SectionGroup[] = [
  {
    title: "Painéis",
    links: [
      { href: "/paineis", label: "Maiores fornecedores", description: "Ranking das empresas mais contratadas" },
      { href: "/pessoal", label: "Pessoal", description: "Vínculos e remuneração por mês" },
      { href: "/tce", label: "TCE-RJ", description: "Contas, obras paralisadas, débitos e multas" },
      { href: "/federal", label: "Dinheiro federal", description: "Transferências e emendas para São Gonçalo" },
      { href: "/agentes", label: "Agentes políticos", description: "Subsídios fixados e quem recebeu" },
    ],
  },
  {
    title: "Câmara",
    links: [{ href: "/proposicoes", label: "Proposições da Câmara", description: "Projetos, fase e andamento das leis" }],
  },
  {
    title: "Dados e ferramentas",
    links: [
      { href: "/dados", label: "Dados abertos", description: "A base inteira para baixar" },
      { href: "/mcp", label: "Pergunte pela sua IA (MCP)", description: "Conecte o Diário à sua IA" },
      { href: "/padroes", label: "Padrões para verificar", description: "Casos que seguem regras descritas" },
    ],
  },
];

export function isCurrentSection(href: string, path: string): boolean {
  return path === href || path.startsWith(href + "/");
}
