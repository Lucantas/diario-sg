import { Registry } from "./api";
import { formatAddress, formatIsoDate, formatRegistryMonth } from "./registry";
import { formatCents } from "./types";

const PARTNER_KIND = { pessoa_juridica: "Pessoa jurídica", pessoa_fisica: "Pessoa física", estrangeiro: "Estrangeiro" } as const;

export function RegistrySection({ registry, month }: { registry: Registry | null; month: string | null }) {
  if (!registry) {
    return month ? (
      <p className="notice">
        CNPJ não encontrado no cadastro da Receita de {formatRegistryMonth(month)}. Pode ser erro de digitação no Diário:
        confira o número na edição original.
      </p>
    ) : null;
  }
  const since = formatIsoDate(registry.status_since);
  return (
    <section className="registry" aria-labelledby="registry-heading">
      <h2 id="registry-heading" className="panel-heading">Cadastro na Receita</h2>
      <dl className="registry-facts">
        {registry.trade_name && <><dt>Nome fantasia</dt><dd>{registry.trade_name}</dd></>}
        <dt>Situação</dt><dd>{registry.status}{since && ` desde ${since}`}</dd>
        <dt>Abertura</dt><dd>{formatIsoDate(registry.opened_at) || "não informada"}{registry.headquarters ? " (matriz)" : " (filial)"}</dd>
        <dt>Natureza jurídica</dt><dd>{registry.legal_nature}</dd>
        <dt>Porte</dt><dd>{registry.size}</dd>
        <dt>Capital social</dt><dd>{formatCents(registry.capital_cents)}</dd>
        <dt>Atividade principal</dt><dd>{registry.main_activity.code} {registry.main_activity.description}</dd>
        {registry.other_activities.length > 0 && (
          <><dt>Outras atividades</dt><dd>{registry.other_activities.map((a) => `${a.code} ${a.description}`).join("; ")}</dd></>
        )}
        <dt>Endereço</dt><dd>{formatAddress(registry)}</dd>
      </dl>
      {registry.partners.length > 0 && (
        <>
          <h3 className="registry-subheading">Sócios</h3>
          <ul className="partners">
            {registry.partners.map((p, i) => (
              <li key={`${p.name}-${i}`}>
                <strong>{p.name}</strong> · {p.role}
                <span className="fineprint-inline">
                  {" "}· {PARTNER_KIND[p.kind]}{p.document && ` ${p.document}`}{p.since && ` · desde ${formatIsoDate(p.since)}`}
                </span>
              </li>
            ))}
          </ul>
        </>
      )}
      <p className="fineprint">
        Dados abertos do CNPJ da Receita Federal, de {formatRegistryMonth(registry.month)}. O CPF de sócio vem mascarado
        pela própria Receita.
      </p>
    </section>
  );
}
