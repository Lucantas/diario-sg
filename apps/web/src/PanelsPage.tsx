import { useEffect, useState } from "react";
import { SupplierPanel, SupplierRow, getSupplierPanel } from "./api";
import { Result } from "./components";
import { PanelParams, contractsLabel, formatCompactCents, panelApiPath, panelHref, parsePanelParams } from "./panels";
import { formatCents, formatCnpj } from "./types";

const TOP_ORGANS = 15;

export function PanelsPage() {
  const [params, setParams] = useState<PanelParams>(() => parsePanelParams(window.location.search));
  const [panel, setPanel] = useState<SupplierPanel | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const onPop = () => setParams(parsePanelParams(window.location.search));
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  useEffect(() => {
    let cancelled = false;
    setError("");
    getSupplierPanel(panelApiPath(params))
      .then((res) => { if (!cancelled) setPanel(res); })
      .catch((err: Error) => {
        if (cancelled) return;
        setPanel(null);
        setError(err.message);
      });
    return () => { cancelled = true; };
  }, [params]);

  const current = panel ? { ...params, organ: panel.organ } : params;

  function go(next: PanelParams) {
    window.history.pushState(null, "", panelHref(next));
    setParams(next);
  }

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Painéis</p>
        <h1>Maiores fornecedores</h1>
      </header>
      <p className="notice">
        Valor declarado nos extratos do Diário, não o que foi pago. Cada contratação (atos da mesma empresa ligados
        pelo processo ou pelo contrato) conta uma vez, pelo maior valor de contrato. Ata de registro de preços é um
        teto e aparece à parte. Aditivos e prorrogações também aparecem à parte, no ano em que foram publicados: a
        prorrogação de 12 meses ou mais conta o valor do novo período; o aditivo de acréscimo, o acréscimo em reais ou
        o percentual declarado sobre o contratado; prorrogação mais curta (que repete o valor global da obra),
        prorrogação "sem ônus", supressão e aditivo sem valor de acréscimo não somam.
        Ficam de fora homologações (que trazem o valor do certame inteiro), editais, multas, cancelamentos e atos que
        citam mais de uma empresa. Confira sempre a edição original. O pago vem dos empenhos do TCE-RJ (de 2021 em
        diante), é o pago à empresa no ano por qualquer unidade da Prefeitura, sem ligação com um contrato específico, e
        não muda com o filtro de secretaria.
      </p>

      <Filters panel={panel} params={current} onChange={go} />

      {error && <p className="notice notice-error">{error}</p>}
      {!error && panel === null && <p className="count">Carregando…</p>}

      {panel && (
        <>
          <dl className="summary">
            <div><dt>Empresas</dt><dd>{panel.suppliers.toLocaleString("pt-BR")}</dd></div>
            <div><dt>Contratações</dt><dd>{panel.contracts.toLocaleString("pt-BR")}</dd></div>
            <div className="summary-money"><dt>Contratado</dt><dd>{formatCompactCents(panel.contracted_cents)}</dd></div>
            <div className="summary-money"><dt>Em atas</dt><dd>{formatCompactCents(panel.registered_cents)}</dd></div>
            <div className="summary-money"><dt>Aditivos e prorrogações</dt><dd>{formatCompactCents(panel.amended_cents)}</dd></div>
            <div className="summary-money"><dt>Pago (TCE-RJ)</dt><dd>{formatCompactCents(panel.paid_cents)}</dd></div>
          </dl>

          <section aria-labelledby="ranking">
            <h2 id="ranking" className="panel-heading">
              {panel.items.length < panel.suppliers ? `As ${panel.items.length} maiores` : "Empresas"}
            </h2>
            {panel.items.length === 0 && <p className="count">Nenhuma contratação com valor neste recorte.</p>}
            <ol className="suppliers">
              {panel.items.map((row) => <SupplierCard key={row.cnpj} row={row} />)}
            </ol>
          </section>

          <Totals
            showPaid={!current.organ}
            title={current.organ ? `Por ano, em ${current.organ}` : "Por ano"}
            rows={panel.years.map((y) => ({ key: String(y.year), label: String(y.year), amounts: y, active: current.year === y.year,
              onClick: () => go({ ...current, year: current.year === y.year ? null : y.year }) }))}
          />
          <Totals
            title={current.year ? `Por secretaria, em ${current.year}` : "Por secretaria"}
            rows={panel.organs.slice(0, TOP_ORGANS).map((o) => ({ key: o.organ, label: o.organ, hint: o.organ_name, amounts: o,
              active: current.organ === o.organ, onClick: () => go({ ...current, organ: current.organ === o.organ ? "" : o.organ }) }))}
          />
        </>
      )}
    </main>
  );
}

function Filters({ panel, params, onChange }: { panel: SupplierPanel | null; params: PanelParams; onChange: (p: PanelParams) => void }) {
  const years = panel?.years.map((y) => y.year) ?? [];
  if (params.year !== null && !years.includes(params.year)) years.push(params.year);
  const organs = panel?.organs.map((o) => ({ organ: o.organ, name: o.organ_name })) ?? [];
  if (params.organ && !organs.some((o) => o.organ === params.organ)) organs.push({ organ: params.organ, name: "" });
  return (
    <div className="panel-filters">
      <div className="organ-filter">
        <label htmlFor="panel-year">Ano</label>
        <select id="panel-year" value={params.year ?? ""}
          onChange={(e) => onChange({ ...params, year: e.target.value ? Number(e.target.value) : null })}>
          <option value="">Todos os anos</option>
          {years.sort((a, b) => b - a).map((y) => <option key={y} value={y}>{y}</option>)}
        </select>
      </div>
      <div className="organ-filter">
        <label htmlFor="panel-organ">Secretaria</label>
        <select id="panel-organ" value={params.organ} onChange={(e) => onChange({ ...params, organ: e.target.value })}>
          <option value="">Todas as secretarias</option>
          {organs.map((o) => (
            <option key={o.organ} value={o.organ}>{o.name ? `${o.organ} · ${o.name}` : o.organ}</option>
          ))}
        </select>
      </div>
    </div>
  );
}

function SupplierCard({ row }: { row: SupplierRow }) {
  const period = row.first.slice(0, 4) === row.last.slice(0, 4) ? row.first.slice(0, 4) : `${row.first.slice(0, 4)} a ${row.last.slice(0, 4)}`;
  return (
    <li className="supplier">
      <h3><a href={`/empresa/${row.cnpj}`} title="Ver todos os atos desta empresa">{row.name || formatCnpj(row.cnpj)}</a></h3>
      {row.name && <p className="supplier-cnpj">{formatCnpj(row.cnpj)}</p>}
      <p className="supplier-values">
        {row.contracted_cents > 0 && <span><strong>{formatCents(row.contracted_cents)}</strong> contratados</span>}
        {row.registered_cents > 0 && <span><strong>{formatCents(row.registered_cents)}</strong> em atas de registro de preços</span>}
        {row.amended_cents > 0 && <span><strong>{formatCents(row.amended_cents)}</strong> em aditivos e prorrogações</span>}
        {row.paid_cents > 0 && <span><strong>{formatCents(row.paid_cents)}</strong> pagos (TCE-RJ)</span>}
      </p>
      <p className="fineprint">
        {contractsLabel(row.contracts)}, {period}
        {row.organs.length > 0 && ` · ${row.organs.join(", ")}`}
      </p>
      {row.largest && (
        <details className="largest">
          <summary>Ato de maior valor</summary>
          <ol className="timeline"><Result hit={row.largest} /></ol>
        </details>
      )}
    </li>
  );
}

interface TotalRow {
  key: string;
  label: string;
  hint?: string;
  amounts: { contracts: number; contracted_cents: number; registered_cents: number; amended_cents: number; paid_cents?: number };
  active: boolean;
  onClick: () => void;
}

function Totals({ title, rows, showPaid = false }: { title: string; rows: TotalRow[]; showPaid?: boolean }) {
  if (rows.length === 0) return null;
  return (
    <section className="totals" aria-label={title}>
      <h2 className="panel-heading">{title}</h2>
      <table>
        <thead>
          <tr>
            <th scope="col"></th><th scope="col">Contratado</th><th scope="col">Em atas</th><th scope="col">Aditivos</th>
            {showPaid && <th scope="col">Pago</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.key}>
              <th scope="row">
                <button type="button" className="link-button" aria-pressed={r.active} title={r.hint || undefined} onClick={r.onClick}>
                  {r.label}
                </button>
              </th>
              <td>{formatCompactCents(r.amounts.contracted_cents)}</td>
              <td>{formatCompactCents(r.amounts.registered_cents)}</td>
              <td>{formatCompactCents(r.amounts.amended_cents)}</td>
              {showPaid && <td>{r.amounts.paid_cents ? formatCompactCents(r.amounts.paid_cents) : "—"}</td>}
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
