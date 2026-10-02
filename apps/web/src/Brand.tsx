type LogoSize = "sm" | "md" | "lg";

export function Logo({ size = "md", href }: { size?: LogoSize; href?: string }) {
  const mark = (
    <>
      <span>Diário</span>
      <span className="logo-block">SG</span>
    </>
  );
  if (!href) return <span className={`logo logo-${size}`} role="img" aria-label="Diário SG">{mark}</span>;
  return (
    <a className={`logo logo-${size}`} href={href} aria-label="Diário SG, página inicial">
      {mark}
    </a>
  );
}

export function SiteHeader() {
  return (
    <header className="site-header">
      <Logo size="md" href="/" />
    </header>
  );
}

const SECTIONS: [string, string][] = [
  ["/dados", "Dados abertos: a base inteira para baixar"],
  ["/mcp", "Pergunte pela sua IA (MCP)"],
  ["/padroes", "Padrões para verificar"],
  ["/paineis", "Maiores fornecedores"],
  ["/pessoal", "Pessoal"],
  ["/tce", "TCE-RJ"],
  ["/federal", "Dinheiro federal"],
  ["/agentes", "Agentes políticos"],
  ["/proposicoes", "Proposições da Câmara"],
];

export function SiteFooter() {
  const current = window.location.pathname;
  return (
    <footer className="site-footer">
      <nav aria-label="Seções">
        {SECTIONS.map(([href, label]) => (
          <a key={href} href={href} aria-current={current === href ? "page" : undefined}>{label}</a>
        ))}
      </nav>
      <p className="fineprint">
        Projeto independente, sem ligação com a Prefeitura nem com a Câmara de São Gonçalo. Confira sempre a edição original.
      </p>
    </footer>
  );
}
