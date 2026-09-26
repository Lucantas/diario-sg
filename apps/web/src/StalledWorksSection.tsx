import { StalledWork } from "./api";
import { stalledPeriod } from "./tce";
import { formatCents } from "./types";

export function StalledWorksList({ works, showContractor = false }: { works: StalledWork[]; showContractor?: boolean }) {
  return (
    <ul className="pncp-list">
      {works.map((w, i) => (
        <li key={`${i}-${w.contract}`}>
          <p>
            <strong>Contrato {w.contract}</strong>
            {showContractor && (
              <>
                {" · "}
                {w.cnpj ? <a href={`/empresa/${w.cnpj}`}>{w.contractor}</a> : w.contractor}
              </>
            )}
          </p>
          <p>
            {formatCents(w.total_cents)} contratados, {formatCents(w.paid_cents)} pagos · {stalledPeriod(w.started_at, w.stalled_at)}
          </p>
          <p className="count">
            {[w.organ, w.function, w.reason, w.contract_status && `contrato ${w.contract_status.toLowerCase()}`].filter(Boolean).join(" · ")}
          </p>
        </li>
      ))}
    </ul>
  );
}

export function StalledWorksSection({ works }: { works: StalledWork[] }) {
  if (works.length === 0) return null;
  return (
    <section className="pncp" aria-labelledby="stalled-heading">
      <h2 id="stalled-heading" className="panel-heading">Obras paralisadas (TCE-RJ)</h2>
      <StalledWorksList works={works} />
      <p className="fineprint">
        Obras que o TCE-RJ lista como paralisadas em São Gonçalo, com o que o município informou ao tribunal. Veja todas em{" "}
        <a href="/tce">TCE-RJ</a>.
      </p>
    </section>
  );
}
