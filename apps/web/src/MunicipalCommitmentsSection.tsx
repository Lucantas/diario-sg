import { MunicipalSupplier } from "./api";
import { commitmentHeading, commitmentProcess, objectPreview } from "./municipal";
import { formatCents } from "./types";

const SHOWN = 10;

export function MunicipalCommitmentsSection({ supplier }: { supplier: MunicipalSupplier }) {
  if (supplier.years.length === 0) return null;
  return (
    <section className="payments" aria-labelledby="municipal-heading">
      <h2 id="municipal-heading" className="panel-heading">Empenhos no portal da Prefeitura</h2>
      <div className="totals">
        <table>
          <thead>
            <tr><th scope="col">Ano</th><th scope="col">Empenhos</th><th scope="col">Empenhado</th><th scope="col">Pago</th></tr>
          </thead>
          <tbody>
            {supplier.years.map((y) => (
              <tr key={y.year}>
                <th scope="row">{y.year}</th><td>{y.commitments}</td><td>{formatCents(y.committed_cents)}</td><td>{formatCents(y.paid_cents)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ul className="pncp-list">
        {supplier.recent.slice(0, SHOWN).map((c) => (
          <li key={`${c.entity}-${c.year}-${c.number}`}>
            <p><strong>{commitmentHeading(c)}</strong></p>
            <p>{objectPreview(c.object)}</p>
            <p className="count">
              {[commitmentProcess(c), `empenhado ${formatCents(c.committed_cents)}`, `pago ${formatCents(c.paid_cents)}`].filter(Boolean).join(" · ")}
            </p>
          </li>
        ))}
      </ul>
      <p className="fineprint">
        Empenhos do portal da transparência da Prefeitura (todas as entidades, de 2017 em diante), com o valor acumulado do
        empenho; os {Math.min(SHOWN, supplier.recent.length)} mais recentes. O número do processo é o do portal, sem o prefixo do órgão.
      </p>
    </section>
  );
}
