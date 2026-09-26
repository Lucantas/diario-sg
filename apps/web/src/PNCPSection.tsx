import { PNCPContract } from "./api";
import { pncpHeading, pncpTerm } from "./pncp";
import { formatIsoDate } from "./registry";
import { formatCents } from "./types";

export function PNCPSection({ contracts }: { contracts: PNCPContract[] }) {
  if (contracts.length === 0) return null;
  return (
    <section className="pncp" aria-labelledby="pncp-heading">
      <h2 id="pncp-heading" className="panel-heading">Contratos no PNCP</h2>
      <ul className="pncp-list">
        {contracts.map((c) => (
          <li key={c.control_number}>
            <p>
              <a href={c.url} target="_blank" rel="noreferrer">{pncpHeading(c)}</a>
              {" · "}
              <strong>{formatCents(c.value_cents)}</strong>
            </p>
            <p>{c.object}</p>
            <p className="count">
              {c.unit} · assinado em {formatIsoDate(c.signed_at) || "data não informada"} · vigência {pncpTerm(c)}
            </p>
          </li>
        ))}
      </ul>
      <p className="fineprint">
        Contratos registrados pelo município no Portal Nacional de Contratações Públicas (Lei 14.133). O município registra
        no PNCP só parte dos contratos, quase todos de 2024 em diante.
      </p>
    </section>
  );
}
