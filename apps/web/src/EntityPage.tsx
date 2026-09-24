import { useEffect, useState } from "react";
import { EntityKind, EntityResponse, Related, getEntity } from "./api";
import { Result } from "./components";
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
  const pressedOrgan = data ? selectedOrgan(data.organs, organ) : "";

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">{ENTITY_EYEBROW[kind]}</p>
        {data && <h1>{data.label}</h1>}
      </header>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <>
          {data.warnings.map((w) => <p key={w} className="notice">{w}</p>)}

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

          {data.total_acts > data.acts.length && (
            <p className="fineprint">
              Mostrando os {data.acts.length} atos mais recentes de {data.total_acts}.
            </p>
          )}

          {data.organs.length > 1 && (
            <div className="organ-filter" role="group" aria-label="Filtrar por órgão">
              <button type="button" aria-pressed={pressedOrgan === ""} onClick={() => chooseOrgan("")}>
                Todos
              </button>
              {data.organs.map((o) => (
                <button
                  key={o.organ || "sem-orgao"}
                  type="button"
                  aria-pressed={pressedOrgan === o.organ}
                  onClick={() => chooseOrgan(o.organ)}
                >
                  {o.organ || "Sem órgão"} ({o.acts})
                </button>
              ))}
            </div>
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
              {groups.length > 1 && <h2>{organHeading(data, group.organ)}</h2>}
              <ol className="timeline">
                {group.acts.map((h) => <Result key={h.id} hit={h} />)}
              </ol>
            </section>
          ))}

          {data.related.length > 0 && (
            <section className="related">
              <h2>Citados junto</h2>
              <ul>
                {data.related.map((r) => (
                  <li key={`${r.kind}-${r.slug}`}>
                    <a href={relatedHref(r)}>{r.label}</a> ({r.acts})
                  </li>
                ))}
              </ul>
            </section>
          )}
        </>
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
