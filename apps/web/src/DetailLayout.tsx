import { ReactNode, useEffect, useState } from "react";

export interface PageSection {
  id: string;
  label: string;
}

export function DetailHeader({ eyebrow, title, children }: { eyebrow: ReactNode; title: ReactNode; children?: ReactNode }) {
  return (
    <header className="card detail-header">
      <p className="eyebrow">{eyebrow}</p>
      {title && <h1>{title}</h1>}
      {children}
    </header>
  );
}

export function PageNav({ sections }: { sections: PageSection[] }) {
  const [present, setPresent] = useState<PageSection[]>([]);

  useEffect(() => {
    setPresent(sections.filter((s) => document.getElementById(s.id)));
  }, [sections]);

  if (present.length < 2) return null;
  return (
    <nav className="page-nav" aria-label="Nesta página">
      {present.map((s) => <a key={s.id} href={`#${s.id}`} className="pill pill-sm">{s.label}</a>)}
    </nav>
  );
}

export function DetailLayout({ lead, aside, children }: { lead: ReactNode; aside: ReactNode; children?: ReactNode }) {
  return (
    <div className="detail-layout">
      <div className="detail-lead">{lead}</div>
      <aside className="detail-aside">{aside}</aside>
      {children && <div className="detail-rest">{children}</div>}
    </div>
  );
}
