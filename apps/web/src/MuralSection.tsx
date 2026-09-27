import { MuralMatches } from "./api";
import { formatIsoDate } from "./registry";
import { formatCents } from "./types";

const SHOWN = 10;

export function MuralSection({ mural }: { mural: MuralMatches }) {
  if (mural.procurements.length === 0 && mural.contracts.length === 0) return null;
  return (
    <section className="pncp" aria-labelledby="mural-heading">
      <h2 id="mural-heading" className="panel-heading">Licitações e contratos no mural da Prefeitura</h2>
      <p className="count">
        Ligados pelo número do processo dos empenhos desta empresa no portal da Prefeitura. O mural não traz CNPJ e o número
        não tem o órgão: confira o fornecedor e o objeto.
      </p>
      <ul className="pncp-list">
        {mural.contracts.slice(0, SHOWN).map((c, i) => (
          <li key={`c-${i}-${c.procurement_id}`}>
            <p>
              <strong>{c.instrument || "Contrato"} · processo {c.process}</strong>
              {c.value_cents > 0 && <> · {formatCents(c.value_cents)}</>}
            </p>
            <p>{c.object}</p>
            <p className="count">
              {[c.supplier, c.modality, c.notice && `edital ${c.notice}`].filter(Boolean).join(" · ")}
              {c.document_url && <> · <a href={c.document_url}>documento</a></>}
            </p>
          </li>
        ))}
        {mural.procurements.slice(0, SHOWN).map((p) => (
          <li key={`p-${p.list}-${p.id}`}>
            <p><strong>{p.modality} · processo {p.process}</strong> · {p.status}</p>
            <p>{p.object}</p>
            <p className="count">
              {[p.notice && `edital ${p.notice}`, p.opens_at && `abertura em ${formatIsoDate(p.opens_at)}`].filter(Boolean).join(" · ")}
              {" · "}<a href={p.url}>no mural</a>
            </p>
          </li>
        ))}
      </ul>
    </section>
  );
}
