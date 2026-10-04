import { FormEvent, ReactNode, useEffect, useRef, useState } from "react";
import { SECTION_GROUPS, SectionLink, isCurrentSection } from "./sections";

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

export function SearchIcon({ size = 18 }: { size?: number }) {
  return (
    <svg aria-hidden="true" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" className="search-icon">
      <circle cx="11" cy="11" r="7" />
      <path d="M20 20l-3.5-3.5" />
    </svg>
  );
}

interface ControlledSearch {
  value: string;
  onChange: (q: string) => void;
  onSubmit: (e: FormEvent) => void;
  loading: boolean;
}

export function HeaderSearch({ controlled }: { controlled?: ControlledSearch }) {
  const inputProps = controlled
    ? { value: controlled.value, onChange: (e: React.ChangeEvent<HTMLInputElement>) => controlled.onChange(e.target.value) }
    : { name: "q" };
  return (
    <form className="header-search" action="/" role="search" onSubmit={controlled?.onSubmit}>
      <SearchIcon />
      <label htmlFor="q-head" className="visually-hidden">Buscar nos Diários Oficiais</label>
      <input id="q-head" type="search" placeholder="Nome, CNPJ, empresa ou assunto" autoComplete="off" {...inputProps} />
      <button type="submit" className="primary" disabled={controlled?.loading}>
        {controlled?.loading ? "Buscando" : "Buscar"}
      </button>
    </form>
  );
}

export function SiteHeader({ search }: { search?: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false);
  return (
    <>
      <header className="site-header">
        <div className="site-header-inner">
          <Logo size="md" href="/" />
          {search}
          <span className="site-header-spacer" />
          <HeaderNav />
          <button type="button" className="menu-button" aria-haspopup="dialog" aria-expanded={menuOpen}
            onClick={() => setMenuOpen(true)}>
            <span className="burger" aria-hidden="true"><span /><span /><span /></span>
            Menu
          </button>
        </div>
      </header>
      <MenuDrawer open={menuOpen} onClose={() => setMenuOpen(false)} />
    </>
  );
}

const [PANELS, ...OTHER_GROUPS] = SECTION_GROUPS;

const SHORT_LABEL: Record<string, string> = {
  "/proposicoes": "Proposições",
  "/dados": "Dados abertos",
  "/mcp": "MCP",
  "/padroes": "Padrões",
};

function currentProps(href: string) {
  return isCurrentSection(href, window.location.pathname) ? { "aria-current": "page" as const } : {};
}

function HeaderNav() {
  return (
    <nav className="header-nav" aria-label="Seções">
      <PanelsDropdown />
      {OTHER_GROUPS.flatMap((g) => g.links).map((l) => (
        <a key={l.href} href={l.href} className="header-link" {...currentProps(l.href)}>{SHORT_LABEL[l.href] ?? l.label}</a>
      ))}
    </nav>
  );
}

function PanelsDropdown() {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onPointer(e: MouseEvent) {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointer);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div className="panels-dropdown" ref={ref}>
      <button type="button" className="header-link" aria-expanded={open} aria-controls="panels-menu"
        onClick={() => setOpen(!open)}>
        {PANELS.title}
        <span className="chevron" aria-hidden="true" />
      </button>
      {open && (
        <ul id="panels-menu" className="panels-menu">
          {PANELS.links.map((l) => <li key={l.href}><DescribedLink link={l} /></li>)}
        </ul>
      )}
    </div>
  );
}

function DescribedLink({ link }: { link: SectionLink }) {
  return (
    <a href={link.href} className="described-link" {...currentProps(link.href)}>
      <span className="described-link-label">{link.label}</span>
      <span className="described-link-desc">{link.description}</span>
    </a>
  );
}

function MenuDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog ref={ref} className="menu-drawer" aria-label="Menu" onClose={onClose}
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="menu-drawer-head">
        <Logo size="sm" />
        <button type="button" className="icon-button" aria-label="Fechar menu" onClick={onClose}>×</button>
      </div>
      <nav aria-label="Seções" className="menu-drawer-nav">
        <a href="/" className="menu-drawer-home" {...(window.location.pathname === "/" ? { "aria-current": "page" as const } : {})}>
          <SearchIcon />
          Buscar nos Diários
        </a>
        {SECTION_GROUPS.map((g) => (
          <div key={g.title} className="menu-drawer-group">
            <p className="group-title">{g.title}</p>
            {g.links.map((l) => <DescribedLink key={l.href} link={l} />)}
          </div>
        ))}
      </nav>
    </dialog>
  );
}

export function SectionCards() {
  return (
    <section className="section-cards" aria-label="Seções">
      {SECTION_GROUPS.map((g) => (
        <div key={g.title} className="card section-card">
          <h2 className="group-title">{g.title}</h2>
          <ul>
            {g.links.map((l) => (
              <li key={l.href}>
                <a href={l.href}>
                  <span className="described-link-text">
                    <span className="described-link-label">{l.label}</span>
                    <span className="described-link-desc">{l.description}</span>
                  </span>
                  <span aria-hidden="true" className="arrow">→</span>
                </a>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="site-footer-inner">
        <div className="site-footer-brand">
          <Logo size="sm" />
          <p>Diários Oficiais da Prefeitura e da Câmara Municipal de São Gonçalo. Confira sempre a edição original.</p>
          <p className="fineprint">Projeto independente, sem ligação com a Prefeitura nem com a Câmara de São Gonçalo.</p>
        </div>
        {SECTION_GROUPS.map((g) => (
          <nav key={g.title} aria-label={g.title}>
            <p className="group-title">{g.title}</p>
            {g.links.map((l) => <a key={l.href} href={l.href} {...currentProps(l.href)}>{l.label}</a>)}
          </nav>
        ))}
      </div>
    </footer>
  );
}
