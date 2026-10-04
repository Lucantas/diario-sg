import { useEffect, useState } from "react";
import { EntityKind, EntityResponse, Related, getEntity } from "./api";
import { Result } from "./components";
import { DetailHeader, DetailLayout } from "./DetailLayout";
import { EntityAlert } from "./EntityAlert";
import { entityPath, groupByOrgan, selectedOrgan } from "./entity";
import { PHASES, PHASE_LABEL } from "./types";

const ENTITY_EYEBROW: Record<EntityKind, string> = {
  processo: "Processo",
  contrato: "Contrato",
};

export function EntityPage({ kind, slug }: { kind: EntityKind; slug: string }) {
  const [data, setData] = useState<EntityResponse | null>(null);
  const [error, setError] = useState("");
  const [organ, setOrgan] = useState(() => new URLSearchParams(window.location.search).get("orgao") ?? "");

  useEffect(() => {
    let cancelled = false;
    setData(null);
    setError("");
    setOrgan(new URLSearchParams(window.location.search).get("orgao") ?? "");
    getEntity(kind, slug)
      .then((res) => { if (!cancelled) setData(res); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, [kind, slug]);

  function chooseOrgan(next: string) {
    setOrgan(next);
    const params = new URLSearchParams(window.location.search);
    if (next) params.set("orgao", next);
    else params.delete("orgao");
    const qs = params.toString();
    window.history.replaceState(null, "", window.location.pathname + (qs ? `?${qs}` : ""));
  }

  const groups = data ? groupByOrgan(data.acts, data.organs, organ) : [];
  const selected = data ? selectedOrgan(data.organs, organ) : "";

  return (
    <main className="page page-wide detail">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <DetailHeader eyebrow={ENTITY_EYEBROW[kind]} title={data?.label}>
        {data?.warnings.map((w) => <p key={w} className="notice">{w}</p>)}
      </DetailHeader>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <DetailLayout
          lead={
            <>
              <div className="timeline-head">
                <h2 className="panel-heading">Linha do tempo</h2>
                {data.organs.length > 1 && (
                  <label className="pill-select">
                    <span>Órgão</span>
                    <select value={selected} onChange={(e) => chooseOrgan(e.target.value)}>
                      <option value="">Todos ({data.total_acts})</option>
                      {data.organs.filter((o) => o.organ).map((o) => (
                        <option key={o.organ} value={o.organ}>{o.organ} ({o.acts})</option>
                      ))}
                    </select>
                  </label>
                )}
              </div>

              {data.total_acts > data.acts.length && (
                <p className="fineprint">
                  Mostrando os {data.acts.length} atos mais recentes de {data.total_acts}.
                </p>
              )}

              {data.acts.length === 0 && (
                <p className="count">Nenhum ato indexado cita este número.</p>
              )}

              {groups.map((group) => (
                <section
                  key={group.organ || "sem-orgao"}
                  className="results"
                  aria-label={`Linha do tempo${group.organ ? ` — ${group.organ}` : ""}`}
                >
                  {groups.length > 1 && <h3 className="group-title">{organHeading(data, group.organ)}</h3>}
                  <ol className="timeline">
                    {group.acts.map((h) => <Result key={h.id} hit={h} />)}
                  </ol>
                </section>
              ))}

              <EntityAlert kind={kind} value={data.label} label={data.label} />
            </>
          }
          aside={
            <>
              <section aria-labelledby="summary-heading">
                <h2 id="summary-heading" className="panel-heading">Resumo</h2>
                <dl className="summary">
                  <div>
                    <dt>Atos</dt>
                    <dd>{data.total_acts}</dd>
                  </div>
                  {PHASES.filter((p) => data.count_by_phase[p]).map((p) => (
                    <div key={p}>
                      <dt>{PHASE_LABEL[p]}</dt>
                      <dd>{data.count_by_phase[p]}</dd>
                    </div>
                  ))}
                </dl>
              </section>
              {data.related.length > 0 && (
                <section className="related" aria-labelledby="related-heading">
                  <h2 id="related-heading" className="panel-heading">Citados junto</h2>
                  <ul>
                    {data.related.map((r) => (
                      <li key={`${r.kind}-${r.slug}`}>
                        <a href={relatedHref(r)}>{r.label}</a>
                        <span>{r.acts} {r.acts === 1 ? "ato" : "atos"}</span>
                      </li>
                    ))}
                  </ul>
                </section>
              )}
            </>
          }
        />
      )}
    </main>
  );
}

function organHeading(data: EntityResponse, organ: string) {
  if (!organ) return "Sem órgão";
  const o = data.organs.find((x) => x.organ === organ);
  return o?.organ_name ? `${organ} · ${o.organ_name}` : organ;
}

function relatedHref(r: Related) {
  return r.kind === "cnpj" ? `/empresa/${r.key}` : entityPath(r.kind, r.slug);
}
