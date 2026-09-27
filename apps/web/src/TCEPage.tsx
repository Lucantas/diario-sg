import { useEffect, useState } from "react";
import { TCEOversight, getOversight } from "./api";
import { formatIsoDate } from "./registry";
import { formatCompactCents } from "./panels";
import { StalledWorksList } from "./StalledWorksSection";
import { condemnationsLabel, coverageLabel, diarioSearchHref, rreoPeriodLabel } from "./tce";
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

          <section className="totals" aria-labelledby="fiscal-heading">
            <h2 id="fiscal-heading" className="panel-heading">Total de controle</h2>
            <p className="count">
              O total que a Prefeitura declara ao Tesouro Nacional no RREO (SICONFI) ao lado do pago no portal da transparência
              da Prefeitura e da soma dos empenhos que o TCE-RJ publica. Quando o TCE cobre menos de 90% do que foi pago no ano fechado, faltam empenhos na base do tribunal, e os
              totais por credor ficam abaixo do real.
            </p>
            {data.fiscal_control.length === 0 ? <p className="count">Nenhum RREO carregado.</p> : (
              <table>
                <thead>
                  <tr>
                    <th scope="col">Exercício</th><th scope="col">Empenhado (RREO)</th><th scope="col">Pago (RREO)</th>
                    <th scope="col">Pago (portal)</th><th scope="col">Pago (TCE-RJ)</th><th scope="col">Cobertura TCE</th>
                  </tr>
                </thead>
                <tbody>
                  {data.fiscal_control.map((f) => (
                    <tr key={f.year}>
                      <th scope="row"><a href={f.source_url}>{f.year}</a><br /><span className="count">{rreoPeriodLabel(f.period)}</span></th>
                      <td>{formatCompactCents(f.committed_cents)}</td><td>{formatCompactCents(f.paid_cents)}</td>
                      <td>{f.portal_paid_cents === null ? "—" : formatCompactCents(f.portal_paid_cents)}</td>{f.tce_loaded ? (
                        <><td>{formatCompactCents(f.tce_paid_cents)}</td><td>{coverageLabel(f.paid_coverage_bp)}{f.low_coverage && " · incompleto"}</td></>
                      ) : <td colSpan={2}>sem empenhos carregados</td>}
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
