import { useEffect, useState } from "react";
import { TCEOversight, getOversight } from "./api";
import { formatIsoDate } from "./registry";
import { StalledWorksList } from "./StalledWorksSection";
import { condemnationsLabel, diarioSearchHref } from "./tce";
import { formatCents } from "./types";

export function TCEPage() {
  const [data, setData] = useState<TCEOversight | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    getOversight()
      .then((res) => { if (!cancelled) setData(res); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, []);

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Controle externo</p>
        <h1>TCE-RJ</h1>
      </header>
      <p className="notice">
        O que o Tribunal de Contas do Estado publica em dados abertos sobre São Gonçalo: o parecer prévio das contas de
        governo, as obras paralisadas e os débitos e multas aplicados às unidades do município. Os dados abertos não trazem o
        nome de quem foi condenado; o processo do TCE é o do tribunal, não o da Prefeitura.
      </p>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <>
          <section className="totals" aria-labelledby="accounts-heading">
            <h2 id="accounts-heading" className="panel-heading">Contas de governo</h2>
            {data.accounts.length === 0 ? <p className="count">Nenhum parecer carregado.</p> : (
              <table>
                <thead>
                  <tr><th scope="col">Exercício</th><th scope="col">Parecer prévio</th><th scope="col">Processo</th><th scope="col">Prefeito</th></tr>
                </thead>
                <tbody>
                  {data.accounts.map((a) => (
                    <tr key={a.year}>
                      <th scope="row">{a.year}</th><td>{a.opinion}</td><td>{a.process}</td><td>{a.responsible}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>

          <section className="pncp" aria-labelledby="works-heading">
            <h2 id="works-heading" className="panel-heading">Obras paralisadas</h2>
            {data.works.length === 0 ? <p className="count">Nenhuma obra paralisada na lista do TCE-RJ.</p> : <StalledWorksList works={data.works} showContractor />}
          </section>

          <section className="pncp" aria-labelledby="penalties-heading">
            <h2 id="penalties-heading" className="panel-heading">Débitos e multas</h2>
            <p className="count">
              {data.penalties.length} processos, {formatCents(data.penalties.reduce((sum, p) => sum + p.total_cents, 0))}.
            </p>
            <ul className="pncp-list">
              {data.penalties.map((p) => (
                <li key={p.process}>
                  <p>
                    <strong>Processo TCE-RJ {p.process}</strong> · {formatCents(p.total_cents)} em {condemnationsLabel(p.condemnations.length)}
                  </p>
                  <p className="count">
                    {[p.organs.join(", "), p.natures.join(", ").toLowerCase(), p.last_session && `sessão de ${formatIsoDate(p.last_session)}`]
                      .filter(Boolean).join(" · ")}
                  </p>
                  {p.search && <p><a href={diarioSearchHref(p.search)}>Procurar o processo no Diário</a></p>}
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </main>
  );
}
