import { useEffect, useState } from "react";
import { AgentRole, PoliticalAgent, PoliticalAgentsReport, getPoliticalAgents } from "./api";
import { AgentFilter, ROLES, agentSubtitle, diarioSearchUrl, roleLabel, visibleAgents } from "./agents";
import { monthSpan } from "./federal";
import { coverageLabel, formatYearMonth } from "./payments";
import { formatCents } from "./types";

const BODY_NAMES = { prefeitura: "Prefeitura", camara: "Câmara" } as const;

export function AgentsPage() {
  const [data, setData] = useState<PoliticalAgentsReport | null>(null);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState<AgentFilter>({ role: "", query: "", past: false });

  useEffect(() => {
    let cancelled = false;
    getPoliticalAgents()
      .then((res) => { if (!cancelled) setData(res); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, []);

  const agents = data ? visibleAgents(data.agents, data.coverage, filter) : [];

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Painéis</p>
        <h1>Agentes políticos</h1>
      </header>
      <p className="notice">
        Prefeito, vice, secretários municipais, Procurador-Geral e vereadores, com a remuneração mês a mês publicada pela
        Prefeitura e pela Câmara nos portais de transparência. Só agentes políticos aparecem com nome; os demais servidores
        estão em agregados na página <a href="/pessoal">Pessoal</a>. A Prefeitura informa só o valor bruto; na Câmara,
        dezembro traz o 13º junto. As folhas não trazem CPF inteiro nem matrícula: o nome é a única ligação entre elas.
      </p>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <>
          <section className="totals" aria-labelledby="norms-heading">
            <h2 id="norms-heading" className="panel-heading">Subsídio fixado</h2>
            <table>
              <thead><tr><th scope="col">Cargo</th><th scope="col">Período</th><th scope="col">Subsídio mensal</th><th scope="col">Norma</th></tr></thead>
              <tbody>
                {data.norms.map((n) => (
                  <tr key={n.role}>
                    <th scope="row">{roleLabel(n.role)}</th>
                    <td>{n.from_year} a {n.to_year}</td>
                    <td>{formatCents(n.value_cents)}</td>
                    <td><a href={`/?${new URLSearchParams({ q: n.search, fonte: n.diario }).toString()}`}>{n.norm}</a></td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="count">
              Folhas carregadas: {data.coverage.map((c) => `${BODY_NAMES[c.body]}, ${coverageLabel(c)}`).join("; ") || "nenhuma"}.
            </p>
          </section>

          <section className="pncp" aria-labelledby="agents-heading">
            <h2 id="agents-heading" className="panel-heading">Quem recebeu</h2>
            <div className="panel-filters organ-filter">
              <label>
                Cargo{" "}
                <select value={filter.role} onChange={(e) => setFilter({ ...filter, role: e.target.value as AgentRole | "" })}>
                  <option value="">Todos</option>
                  {ROLES.map((r) => <option key={r} value={r}>{roleLabel(r)}</option>)}
                </select>
              </label>
              <label>
                Nome ou lotação{" "}
                <input type="search" value={filter.query} onChange={(e) => setFilter({ ...filter, query: e.target.value })} />
              </label>
              <label>
                <input type="checkbox" checked={filter.past} onChange={(e) => setFilter({ ...filter, past: e.target.checked })} />{" "}
                Incluir quem já saiu
              </label>
            </div>
            <p className="count">{agents.length === 1 ? "1 agente" : `${agents.length} agentes`}.</p>
            <ul className="pncp-list">
              {agents.map((a) => <AgentItem key={`${a.body}-${a.role}-${a.name}`} agent={a} />)}
            </ul>
          </section>
        </>
      )}
    </main>
  );
}

function AgentItem({ agent }: { agent: PoliticalAgent }) {
  const latest = agent.months[agent.months.length - 1];
  const hasNet = agent.months.some((m) => m.net_cents !== null);
  return (
    <li>
      <details>
        <summary>
          <strong>{agent.name}</strong> · {roleLabel(agent.role)}
          <span className="fineprint-inline"> · {monthSpan(agent.first, agent.last)}</span>
        </summary>
        <p className="count">{agentSubtitle(agent)}</p>
        <p><a href={diarioSearchUrl(agent.name)}>Buscar o nome no Diário</a></p>
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th scope="col">Mês</th><th scope="col">Lotação</th><th scope="col">Bruto</th>
                {hasNet && <><th scope="col">Descontos</th><th scope="col">Líquido</th></>}
              </tr>
            </thead>
            <tbody>
              {[...agent.months].reverse().map((m) => (
                <tr key={`${m.month}-${m.office}`}>
                  <th scope="row">{formatYearMonth(m.month)}</th>
                  <td className="wrap">{m.office}</td>
                  <td>{formatCents(m.gross_cents)}</td>
                  {hasNet && <><td>{m.discount_cents === null ? "—" : formatCents(m.discount_cents)}</td><td>{m.net_cents === null ? "—" : formatCents(m.net_cents)}</td></>}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
      {latest && <p className="count">Último mês: {formatCents(latest.gross_cents)} bruto.</p>}
    </li>
  );
}
