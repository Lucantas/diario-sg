import { useEffect, useState } from "react";
import { ActType, CompanyResponse, getCompany } from "./api";
import { Result } from "./components";
import { TYPE_LABEL, formatCents, formatCnpj } from "./types";

export function CompanyPage({ cnpj }: { cnpj: string }) {
  const [data, setData] = useState<CompanyResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    getCompany(cnpj)
      .then((res) => { if (!cancelled) setData(res); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, [cnpj]);

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Empresa</p>
        <h1>{formatCnpj(cnpj)}</h1>
      </header>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <>
          <dl className="summary">
            <div>
              <dt>Atos</dt>
              <dd>{data.acts.length}</dd>
            </div>
            <div>
              <dt>Valores citados</dt>
              <dd>{formatCents(data.total_value_cents)}</dd>
            </div>
            {(Object.keys(data.count_by_type) as ActType[]).sort().map((t) => (
              <div key={t}>
                <dt>{TYPE_LABEL[t]}</dt>
                <dd>{data.count_by_type[t]}</dd>
              </div>
            ))}
          </dl>
          <p className="fineprint">
            A soma junta todos os valores em reais encontrados nos atos (mensal, global,
            unitário), sem distinguir o que cada um significa. Confira sempre a edição original.
          </p>

          <section className="results" aria-label="Linha do tempo">
            {data.acts.length === 0 && (
              <p className="count">Nenhum ato indexado cita este CNPJ.</p>
            )}
            <ol className="timeline">
              {data.acts.map((h) => <Result key={h.id} hit={h} />)}
            </ol>
          </section>
        </>
      )}
    </main>
  );
}
