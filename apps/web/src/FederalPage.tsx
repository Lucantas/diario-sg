import { useEffect, useState } from "react";
import { FederalReport, getFederal } from "./api";
import { monthSpan, transferYears } from "./federal";
import { formatCompactCents } from "./panels";
import { formatCents, formatCnpj } from "./types";

const TOP_FAVORED = 30;

export function FederalPage() {
  const [data, setData] = useState<FederalReport | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    getFederal()
      .then((res) => { if (!cancelled) setData(res); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, []);

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Painéis</p>
        <h1>Dinheiro federal</h1>
      </header>
      <p className="notice">
        Transferências da União para São Gonçalo e emendas parlamentares federais, segundo o Portal da Transparência (CGU).
        As transferências vão para o Município, os fundos municipais e entidades da cidade. As emendas aparecem de dois
        jeitos: as destinadas a São Gonçalo (com o autor e a ação) e os pagamentos de emenda a pessoas jurídicas com sede na
        cidade, que podem vir de emendas destinadas a outro lugar. Pagamentos a pessoa física ficam de fora.
      </p>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <>
          <section className="totals" aria-labelledby="transfers-heading">
            <h2 id="transfers-heading" className="panel-heading">Transferências por ano</h2>
            {data.transfers.length === 0 ? <p className="count">Nenhuma transferência carregada.</p> : (
              <table>
                <thead><tr><th scope="col">Ano</th><th scope="col">Total</th><th scope="col">Por tipo</th></tr></thead>
                <tbody>
                  {transferYears(data.transfers).map((y) => (
                    <tr key={y.year}>
                      <th scope="row">{y.year}</th>
                      <td>{formatCompactCents(y.total)}</td>
                      <td className="wrap">{y.byKind.map((k) => `${k.kind}: ${formatCompactCents(k.value)}`).join(" · ")}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>

          <section className="totals" aria-labelledby="amendments-heading">
            <h2 id="amendments-heading" className="panel-heading">Emendas destinadas a São Gonçalo</h2>
            <table>
              <thead>
                <tr><th scope="col">Ano</th><th scope="col">Autor</th><th scope="col">Ação</th><th scope="col">Empenhado</th><th scope="col">Pago</th></tr>
              </thead>
              <tbody>
                {data.amendments.map((a, i) => (
                  <tr key={`${i}-${a.code}`}>
                    <th scope="row">{a.year}</th>
                    <td className="wrap">{a.author}</td>
                    <td className="wrap" title={a.kind}>{a.action}</td>
                    <td>{formatCompactCents(a.committed_cents)}</td>
                    <td>{formatCompactCents(a.paid_cents)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section className="pncp" aria-labelledby="favored-heading">
            <h2 id="favored-heading" className="panel-heading">Quem recebeu pagamento de emenda em São Gonçalo</h2>
            <p className="count">
              {data.favored.length} pessoas jurídicas; as {Math.min(TOP_FAVORED, data.favored.length)} que mais receberam.
            </p>
            <ul className="pncp-list">
              {data.favored.slice(0, TOP_FAVORED).map((f) => (
                <li key={f.cnpj}>
                  <p><a href={`/empresa/${f.cnpj}`}>{f.name || formatCnpj(f.cnpj)}</a> · <strong>{formatCents(f.value_cents)}</strong></p>
                  <p className="count">
                    {f.payments === 1 ? "1 pagamento" : `${f.payments} pagamentos`}, {monthSpan(f.first, f.last)} · {f.authors.join(", ")}
                  </p>
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </main>
  );
}
