import { useEffect, useState } from "react";
import { Finding, Pattern, listPatterns } from "./api";
import { Result } from "./components";
import { patternSearchHref } from "./patterns";

export function PatternsPage() {
  const [patterns, setPatterns] = useState<Pattern[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    listPatterns()
      .then((res) => { if (!cancelled) setPatterns(res.items); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, []);

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Diário SG</p>
        <h1>Padrões para verificar</h1>
      </header>
      <p className="notice">
        Padrões para verificar, não irregularidades: cada caso só diz que os atos seguem a regra descrita. Confira
        sempre a edição original.
      </p>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && patterns === null && <p className="count">Carregando…</p>}

      {patterns?.map((p) => (
        <section key={p.id} className="pattern" aria-labelledby={`padrao-${p.id}`}>
          <h2 id={`padrao-${p.id}`}>{p.title}</h2>
          <p>{p.rule}</p>
          <p className="fineprint">{p.caveat}</p>
          <p className="count">
            {casesLabel(p.findings.length)}
          </p>
          {p.findings.map((f, i) => <FindingCard key={`${i}-${f.title}`} finding={f} />)}
        </section>
      ))}
    </main>
  );
}

function FindingCard({ finding }: { finding: Finding }) {
  return (
    <article className="finding">
      <h3>{finding.title}</h3>
      <p>{finding.detail}</p>
      {finding.search && <p><a href={patternSearchHref(finding.search)}>Ver os atos na busca</a></p>}
      {finding.link && <p><a href={finding.link.url} target="_blank" rel="noreferrer">{finding.link.label}</a></p>}
      {finding.acts.length > 0 && (
        <ol className="timeline">
          {finding.acts.map((h) => <Result key={h.id} hit={h} />)}
        </ol>
      )}
    </article>
  );
}

function casesLabel(n: number) {
  if (n === 0) return "Nenhum caso na base hoje.";
  return n === 1 ? "1 caso na base." : `${n} casos na base.`;
}
