import { useEffect, useState } from "react";
import { BillDetail, getBill } from "./api";
import { idleLabel, phaseLabel } from "./bills";

const CERTAINTY_TEXT = {
  exata: "a página do processo no SICAM aponta a lei",
  forte: "o autor da lei, na consulta de leis da Prefeitura, cita este projeto",
  fraca: "ligação fraca",
} as const;

function day(iso: string | null): string {
  return iso ? iso.slice(0, 10).split("-").reverse().join("/") : "";
}

function dateTime(iso: string): string {
  return new Date(iso).toLocaleString("pt-BR", { timeZone: "America/Sao_Paulo", dateStyle: "short", timeStyle: "short" });
}

export function BillPage({ process }: { process: string }) {
  const [bill, setBill] = useState<BillDetail | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    getBill(process)
      .then((res) => { if (!cancelled) setBill(res); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, [process]);

  return (
    <main className="page">
      <p className="crumb"><a href="/proposicoes">← Proposições</a></p>
      {error && <p className="notice notice-error">{error}</p>}
      {!error && bill === null && <p className="count">Carregando…</p>}
      {bill && (
        <>
          <header className="masthead">
            <p className="eyebrow">{bill.kind} · processo {bill.process}</p>
            <h1>{bill.document}</h1>
          </header>
          <p>{bill.summary}</p>
          <dl className="registry-facts">
            <dt>Autoria</dt><dd>{bill.authors}</dd>
            {bill.presented_on && <><dt>Apresentação</dt><dd>{day(bill.presented_on)}</dd></>}
            <dt>Fase</dt><dd>{phaseLabel(bill.phase)} ({idleLabel(bill.days_idle)})</dd>
            <dt>Situação no SICAM</dt><dd>{bill.status}</dd>
            <dt>Onde está</dt><dd>{bill.current_body}</dd>
            <dt>Última movimentação</dt><dd>{bill.last_movement}</dd>
          </dl>
          <p className="count">
            Fonte: <a href={bill.url}>página do processo no SICAM</a>, lida em {day(bill.fetched_at)}. A página original é que vale.
          </p>

          {bill.laws.length > 0 && (
            <section aria-labelledby="law-heading">
              <h2 id="law-heading" className="panel-heading">Lei que resultou do projeto</h2>
              <ul className="pncp-list">
                {bill.laws.map((l) => (
                  <li key={l.number}>
                    <p>
                      <strong>Lei {l.number}</strong>{l.url && <> · <a href={l.url}>texto</a></>}
                      {" "}· <a href={`/?${new URLSearchParams({ q: l.diario_search }).toString()}`}>atos do Diário que citam a lei</a>
                    </p>
                    {l.summary && <p>{l.summary}</p>}
                    <p className="count">Ligação {l.certainty}: {CERTAINTY_TEXT[l.certainty]}.</p>
                  </li>
                ))}
              </ul>
            </section>
          )}

          <section aria-labelledby="opinions-heading">
            <h2 id="opinions-heading" className="panel-heading">Pareceres das comissões</h2>
            {bill.opinions.length === 0 ? <p className="count">Nenhum parecer registrado.</p> : (
              <div className="table-scroll">
                <table>
                  <thead><tr><th scope="col">Comissão</th><th scope="col">Relator</th><th scope="col">Resultado</th><th scope="col">Data</th></tr></thead>
                  <tbody>
                    {bill.opinions.map((o, i) => (
                      <tr key={i}><td className="wrap">{o.committee}</td><td>{o.rapporteur}</td><td>{o.result}</td><td>{day(o.on)}</td></tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section aria-labelledby="events-heading">
            <h2 id="events-heading" className="panel-heading">Tramitação</h2>
            <ol className="pncp-list">
              {bill.events.map((e, i) => (
                <li key={i}>
                  <p className="count">{dateTime(e.at)} · {e.label}{e.sector && ` · ${e.sector}`}</p>
                  <p>{e.text}</p>
                </li>
              ))}
            </ol>
          </section>
        </>
      )}
    </main>
  );
}
