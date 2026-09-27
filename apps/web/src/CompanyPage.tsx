import { useEffect, useState } from "react";
import { ActType, CompanyResponse, getCompany } from "./api";
import { Result } from "./components";
import { DiarioSanctionsSection } from "./DiarioSanctionsSection";
import { EntityAlert } from "./EntityAlert";
import { RegistrySection } from "./RegistrySection";
import { MunicipalCommitmentsSection } from "./MunicipalCommitmentsSection";
import { MuralSection } from "./MuralSection";
import { PaymentsSection } from "./PaymentsSection";
import { PNCPSection } from "./PNCPSection";
import { StalledWorksSection } from "./StalledWorksSection";
import { AmendmentsSection } from "./AmendmentsSection";
import { SanctionsSection } from "./SanctionsSection";
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
        <p className="eyebrow">Empresa · {formatCnpj(cnpj)}</p>
        <h1>{data?.registry?.name || formatCnpj(cnpj)}</h1>
      </header>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <>
          <RegistrySection registry={data.registry} month={data.registry_month} />
          <SanctionsSection cnpj={data.cnpj} sanctions={data.sanctions} listedOn={data.sanctions_listed_on} />
          <DiarioSanctionsSection sanctions={data.diario_sanctions} />
          <PaymentsSection payments={data.payments} coverage={data.payments_coverage} />
          <MunicipalCommitmentsSection supplier={data.municipal_commitments} />
          <MuralSection mural={data.procurements} />
          <PNCPSection contracts={data.pncp_contracts} />
          <StalledWorksSection works={data.stalled_works} />
          <AmendmentsSection payments={data.amendment_payments} />
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

          <EntityAlert kind="cnpj" value={cnpj} label={formatCnpj(cnpj)} />
        </>
      )}
    </main>
  );
}
