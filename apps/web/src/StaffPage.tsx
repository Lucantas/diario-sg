import { useEffect, useState } from "react";
import { StaffPanel, getStaffPanel } from "./api";
import { formatCompactCents } from "./panels";
import { formatCount, parseStaffUnit, staffApiPath, staffHref, staffMonthLabel } from "./staff";

export function StaffPage() {
  const [unit, setUnit] = useState(() => parseStaffUnit(window.location.search));
  const [panel, setPanel] = useState<StaffPanel | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const onPop = () => setUnit(parseStaffUnit(window.location.search));
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  useEffect(() => {
    let cancelled = false;
    setError("");
    getStaffPanel(staffApiPath(unit))
      .then((res) => { if (!cancelled) setPanel(res); })
      .catch((err: Error) => {
        if (cancelled) return;
        setPanel(null);
        setError(err.message);
      });
    return () => { cancelled = true; };
  }, [unit]);

  function choose(next: string) {
    window.history.pushState(null, "", staffHref(next));
    setUnit(next);
  }

  const diario = panel?.diario_source === "diario_camara" ? "Diário da Câmara" : "Diário da Prefeitura";

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Painéis</p>
        <h1>Pessoal</h1>
      </header>
      <p className="notice">
        Vínculos e remuneração do mês por situação funcional, como o município informa ao TCE-RJ (de janeiro de 2024 em
        diante), somados por unidade. Não há dado de pessoa. A classificação é a informada pelo município e muda: de abril
        de 2024 a abril de 2025, os comissionados da Prefeitura aparecem em "Outros" e os agentes políticos não aparecem; desde
        abril de 2024 não há contratados por tempo determinado.
        As nomeações e exonerações são os atos publicados no Diário no mês (um ato pode nomear várias pessoas).
      </p>

      <div className="panel-filters">
        <div className="organ-filter">
          <label htmlFor="staff-unit">Unidade</label>
          <select id="staff-unit" value={unit} onChange={(e) => choose(e.target.value)}>
            <option value="">Todas as unidades</option>
            {(panel?.units ?? (unit ? [unit] : [])).map((u) => <option key={u} value={u}>{u}</option>)}
          </select>
        </div>
      </div>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && panel === null && <p className="count">Carregando…</p>}
      {panel && panel.months.length === 0 && <p className="count">Nenhum dado de pessoal carregado.</p>}

      {panel && panel.months.length > 0 && (
        <section className="totals staff" aria-labelledby="staff-heading">
          <h2 id="staff-heading" className="panel-heading">Vínculos por mês</h2>
          <table>
            <thead>
              <tr>
                <th scope="col">Mês</th>
                {panel.groups.map((g) => <th scope="col" key={g.group}>{g.label}</th>)}
                <th scope="col">Total</th>
                <th scope="col">Remuneração</th>
                <th scope="col" title={diario}>Nomeações</th>
                <th scope="col" title={diario}>Exonerações</th>
              </tr>
            </thead>
            <tbody>
              {panel.months.map((m) => (
                <tr key={m.month}>
                  <th scope="row">{staffMonthLabel(m.month)}</th>
                  {m.groups.map((g, i) => (
                    <td key={panel.groups[i].group} title={g.headcount ? formatCompactCents(g.remuneration_cents) : undefined}>
                      {formatCount(g.headcount)}
                    </td>
                  ))}
                  <td>{formatCount(m.headcount)}</td>
                  <td>{formatCompactCents(m.remuneration_cents)}</td>
                  <td>{formatCount(m.appointments)}</td>
                  <td>{formatCount(m.dismissals)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="fineprint">
            Remuneração de cada grupo: passe o mouse sobre o número. Nomeações e exonerações: atos do {diario}; o mês
            corrente não entra.
          </p>
        </section>
      )}
    </main>
  );
}
