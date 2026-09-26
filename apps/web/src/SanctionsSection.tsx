import { Sanction, SanctionRegister } from "./api";
import { formatIsoDate } from "./registry";
import { listedOnLabel, sanctionOrgan, sanctionPeriod, sanctionStateLabel, sanctionsSummary, sanctionTerms } from "./sanctions";
import { formatCents, formatCnpj } from "./types";

type ListedOn = Partial<Record<SanctionRegister, string>>;

const REGISTER_NAME = {
  CEIS: "CEIS (empresas inidôneas e suspensas)",
  CNEP: "CNEP (empresas punidas pela Lei Anticorrupção)",
  CEPIM: "CEPIM (entidades sem fins lucrativos impedidas de receber transferências da União por convênio)",
} as const;

export function SanctionsSection({ cnpj, sanctions, listedOn }: { cnpj: string; sanctions: Sanction[]; listedOn: ListedOn }) {
  const consulted = listedOnLabel(listedOn);
  if (!consulted) return null;
  return (
    <section className="sanctions" aria-labelledby="sanctions-heading">
      <h2 id="sanctions-heading" className="panel-heading">Sanções (CGU)</h2>
      {sanctions.length === 0 && <p className="count">Nenhuma sanção no CEIS, no CNEP nem no CEPIM em {consulted}.</p>}
      {sanctions.length > 0 && <p className="count">{sanctionsSummary(sanctions)}</p>}
      <ul className="sanction-list">
        {sanctions.map((s) => (
          <li key={`${s.register}-${s.code}`} className={s.state === "no_cadastro" ? "sanction sanction-listed" : "sanction"}>
            <details>
              <summary>
                <strong>{s.category}</strong> · {s.register} · {sanctionStateLabel(s)}
                <span className="fineprint-inline"> · {s.organ}</span>
              </summary>
              <p className="fineprint">{REGISTER_NAME[s.register]}</p>
              <dl className="registry-facts">
                <dt>{sanctionTerms(s.register).organ}</dt><dd>{sanctionOrgan(s)}</dd>
                <dt>Período</dt><dd>{sanctionPeriod(s)}</dd>
                <dt>Abrangência</dt><dd>{s.scope}</dd>
                {s.process && <><dt>Processo</dt><dd>{s.process}</dd></>}
                {s.published_at && <><dt>Publicação</dt><dd>{formatIsoDate(s.published_at)}</dd></>}
                {s.fine_cents !== null && <><dt>Multa</dt><dd>{formatCents(s.fine_cents)}</dd></>}
                <dt>{sanctionTerms(s.register).basis}</dt><dd className="sanction-basis">{s.legal_basis}</dd>
              </dl>
            </details>
            {s.cnpj !== cnpj && <p className="fineprint">Aplicada a outro estabelecimento da mesma empresa: {formatCnpj(s.cnpj)}.</p>}
          </li>
        ))}
      </ul>
      <p className="fineprint">
        Cadastros da Controladoria-Geral da União no Portal da Transparência, arquivo de {consulted}. A sanção vale
        conforme a abrangência declarada pelo órgão sancionador; o CEPIM só impede convênio com a União, não
        contrato com o Município. Confira o processo antes de publicar.
      </p>
    </section>
  );
}
