import { AmendmentPayment } from "./api";
import { formatYearMonth } from "./payments";
import { formatCents } from "./types";

export function AmendmentsSection({ payments }: { payments: AmendmentPayment[] }) {
  if (payments.length === 0) return null;
  const total = payments.reduce((sum, p) => sum + p.value_cents, 0);
  return (
    <section className="totals" aria-labelledby="amendments-heading">
      <h2 id="amendments-heading" className="panel-heading">Emendas parlamentares (CGU)</h2>
      <p className="count">{payments.length === 1 ? "1 pagamento" : `${payments.length} pagamentos`}, {formatCents(total)}.</p>
      <table>
        <thead>
          <tr><th scope="col">Mês</th><th scope="col">Autor</th><th scope="col">Emenda</th><th scope="col">Valor</th></tr>
        </thead>
        <tbody>
          {payments.map((p, i) => (
            <tr key={`${i}-${p.code}-${p.month}`}>
              <th scope="row">{formatYearMonth(p.month)}</th>
              <td>{p.author}</td>
              <td>{p.code}</td>
              <td>{formatCents(p.value_cents)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="fineprint">
        Pagamentos de emenda parlamentar federal à empresa, segundo o Portal da Transparência. Veja todas as emendas de São
        Gonçalo em <a href="/federal">Dinheiro federal</a>.
      </p>
    </section>
  );
}
