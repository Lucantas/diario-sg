import { PaymentYear } from "./api";
import { coverageLabel } from "./payments";
import { formatCents } from "./types";

export function PaymentsSection({ payments, coverage }: { payments: PaymentYear[]; coverage: { from: string; to: string } | null }) {
  if (!coverage) return null;
  return (
    <section className="payments" aria-labelledby="payments-heading">
      <h2 id="payments-heading" className="panel-heading">Pagamentos (TCE-RJ)</h2>
      {payments.length === 0 ? (
        <p className="count">Nenhum empenho para este CNPJ nos dados do TCE-RJ ({coverageLabel(coverage)}).</p>
      ) : (
        <div className="totals">
          <table>
            <thead>
              <tr><th scope="col">Ano</th><th scope="col">Empenhado</th><th scope="col">Liquidado</th><th scope="col">Pago</th></tr>
            </thead>
            <tbody>
              {payments.map((p) => (
                <tr key={p.year}>
                  <th scope="row" title={p.units.join("; ")}>{p.year}</th>
                  <td>{formatCents(p.committed_cents)}</td>
                  <td>{formatCents(p.liquidated_cents)}</td>
                  <td>{formatCents(p.paid_cents)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="fineprint">
        Empenhos dos municípios no portal de dados abertos do TCE-RJ, pagos de {coverageLabel(coverage)} (2020 só traz o
        empenhado). Valores negativos são anulações. O TCE não diz a que contrato cada pagamento se refere; unidades
        pagadoras: passe o mouse sobre o ano.
      </p>
    </section>
  );
}
