import { useEffect, useState } from "react";
import { ActType, CompanyResponse, getCompany } from "./api";
import { DetailHeader, DetailLayout, PageNav, PageSection } from "./DetailLayout";
import { formatIsoDate, isActive } from "./registry";
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
import { YearBars } from "./YearBars";
import { countByYear } from "./yearBars";

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
    <main className="page page-wide detail">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <DetailHeader
        eyebrow={<>Empresa · <span className="mono">{formatCnpj(cnpj)}</span></>}
        title={data?.registry?.name || formatCnpj(cnpj)}
      >
        {data && <CompanyBadges data={data} />}
      </DetailHeader>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <>
          <PageNav sections={COMPANY_SECTIONS} />
          <DetailLayout
            lead={
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
              </>
            }
            aside={<CompanySummary data={data} />}
          >
            <section className="results" aria-labelledby="timeline-heading">
              <h2 id="timeline-heading" className="panel-heading">Linha do tempo</h2>
              {data.acts.length === 0 && (
                <p className="count">Nenhum ato indexado cita este CNPJ.</p>
              )}
              <ol className="timeline">
                {data.acts.map((h) => <Result key={h.id} hit={h} />)}
              </ol>
            </section>
            <EntityAlert kind="cnpj" value={cnpj} label={formatCnpj(cnpj)} />
          </DetailLayout>
        </>
      )}
    </main>
  );
}

const COMPANY_SECTIONS: PageSection[] = [
  { id: "registry-heading", label: "Cadastro" },
  { id: "sanctions-heading", label: "Sanções" },
  { id: "diario-sanctions-heading", label: "Punições no Diário" },
  { id: "payments-heading", label: "Pagamentos" },
  { id: "municipal-heading", label: "Empenhos" },
  { id: "mural-heading", label: "Mural" },
  { id: "pncp-heading", label: "PNCP" },
  { id: "stalled-heading", label: "Obras paralisadas" },
  { id: "amendments-heading", label: "Emendas" },
  { id: "summary-heading", label: "Resumo" },
  { id: "timeline-heading", label: "Linha do tempo" },
  { id: "alerta", label: "Alerta" },
];

function CompanyBadges({ data }: { data: CompanyResponse }) {
  const registry = data.registry;
  const since = registry ? formatIsoDate(registry.status_since) : "";
  const inactive = registry && !isActive(registry.status);
  const listedSanctions = data.sanctions.some((s) => s.state === "no_cadastro");
  return (
    <>
      <p className="badges">
        {registry && (
          <span className={inactive ? "tag tag-critical" : "tag tag-bay"}>Situação cadastral: {registry.status}</span>
        )}
        {listedSanctions && <a href="#sanctions-heading" className="tag tag-critical">Sanção no cadastro da CGU</a>}
        <span className="tag tag-outline">
          {data.acts.length.toLocaleString("pt-BR")} {data.acts.length === 1 ? "ato" : "atos"} nos Diários
        </span>
      </p>
      {registry && inactive && (
        <p className="notice notice-error" role="alert">
          Situação cadastral: {registry.status}{since && ` desde ${since}`}{registry.status_reason && ` (${registry.status_reason})`}.
        </p>
      )}
    </>
  );
}

function CompanySummary({ data }: { data: CompanyResponse }) {
  return (
    <section aria-labelledby="summary-heading">
      <h2 id="summary-heading" className="panel-heading">Resumo nos Diários</h2>
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
      <ActsByYear dates={data.acts.map((a) => a.published_at)} />
    </section>
  );
}

function ActsByYear({ dates }: { dates: string[] }) {
  const data = countByYear(dates);
  if (data.length < 2) return null;
  return (
    <div className="summary-chart">
      <h3 id="years-heading">Atos que citam a empresa, por ano</h3>
      <p className="fineprint">Fonte: Diários da Prefeitura e da Câmara. O ano corrente vai até a edição mais recente.</p>
      <YearBars data={data} caption="Atos que citam a empresa, por ano" />
    </div>
  );
}
