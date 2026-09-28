import { FormEvent, useEffect, useState } from "react";
import { Bill, BillPhase, BillsResponse, listBills } from "./api";
import { BillsState, DEFAULT_THEME, PHASES, billApiParams, billPath, billsPageUrl, idleLabel, phaseLabel, readBillsState } from "./bills";

const PER_PAGE = 20;
const IDLE_OPTIONS = [0, 90, 180, 365, 730];

export function BillsPage() {
  const [state, setState] = useState<BillsState>(() => readBillsState(window.location.search));
  const [draft, setDraft] = useState(state.q);
  const [data, setData] = useState<BillsResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setData(null);
    setError("");
    window.history.replaceState(null, "", billsPageUrl(state));
    listBills(billApiParams(state, PER_PAGE))
      .then((res) => { if (!cancelled) setData(res); })
      .catch((err: Error) => { if (!cancelled) setError(err.message); });
    return () => { cancelled = true; };
  }, [state]);

  function update(patch: Partial<BillsState>) {
    setState({ ...state, ...patch, page: patch.page ?? 1 });
  }

  function onSearch(e: FormEvent) {
    e.preventDefault();
    update({ q: draft.trim() });
  }

  const pages = data ? Math.ceil(data.total / PER_PAGE) : 0;

  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Câmara Municipal</p>
        <h1>Proposições e andamento das leis</h1>
      </header>
      <p className="notice">
        Projetos de lei, mensagens do Executivo e outras proposições da Câmara, com a tramitação lida da área pública do
        sistema de processo legislativo (SICAM). A fase é deduzida da tramitação por regra; confira na página de cada
        processo. O voto de cada vereador não está publicado em formato aberto; o placar aparece na tramitação.
      </p>

      <form className="panel-filters organ-filter" onSubmit={onSearch}>
        <label>
          Ementa ou autor{" "}
          <input type="search" value={draft} onChange={(e) => setDraft(e.target.value)} />
        </label>
        <label>
          Tema{" "}
          <select value={state.theme} onChange={(e) => update({ theme: e.target.value })}>
            <option value={DEFAULT_THEME}>Meio ambiente</option>
            <option value="">Todos os temas</option>
          </select>
        </label>
        <label>
          Fase{" "}
          <select value={state.phase} onChange={(e) => update({ phase: e.target.value as BillPhase | "" })}>
            <option value="">Todas</option>
            {PHASES.map((p) => <option key={p} value={p}>{phaseLabel(p)}{data?.by_phase[p] ? ` (${data.by_phase[p]})` : ""}</option>)}
          </select>
        </label>
        <label>
          Sem movimentação há{" "}
          <select value={state.idle} onChange={(e) => update({ idle: Number(e.target.value) })}>
            {IDLE_OPTIONS.map((d) => <option key={d} value={d}>{d === 0 ? "qualquer tempo" : `${d} dias ou mais`}</option>)}
          </select>
        </label>
        <label>
          <input type="checkbox" checked={state.kind === "todos"} onChange={(e) => update({ kind: e.target.checked ? "todos" : "" })} />{" "}
          Incluir indicações e moções
        </label>
        <button type="submit" className="primary">Buscar</button>
      </form>

      {error && <p className="notice notice-error">{error}</p>}
      {!error && data === null && <p className="count">Carregando…</p>}

      {data && (
        <section className="pncp" aria-labelledby="bills-heading">
          <h2 id="bills-heading" className="panel-heading">
            {data.total === 1 ? "1 proposição" : `${data.total} proposições`}
          </h2>
          {data.theme_rule && (
            <details className="fineprint">
              <summary>Como o tema é definido</summary>
              <p>{data.theme_rule}</p>
            </details>
          )}
          <ul className="pncp-list">
            {data.items.map((b) => <BillItem key={b.process} bill={b} />)}
          </ul>
          {pages > 1 && (
            <nav className="pager" aria-label="Páginas">
              <button type="button" className="secondary" disabled={state.page <= 1} onClick={() => update({ page: state.page - 1 })}>Anterior</button>
              <span>Página {state.page} de {pages.toLocaleString("pt-BR")}</span>
              <button type="button" className="secondary" disabled={state.page >= pages} onClick={() => update({ page: state.page + 1 })}>Próxima</button>
            </nav>
          )}
        </section>
      )}
    </main>
  );
}

function BillItem({ bill }: { bill: Bill }) {
  return (
    <li>
      <p>
        <a href={billPath(bill.process)}><strong>{bill.document}</strong></a>
        <span className="fineprint-inline"> · {phaseLabel(bill.phase)} · {idleLabel(bill.days_idle)}</span>
      </p>
      <p>{bill.summary}</p>
      <p className="count">
        {bill.authors}
        {bill.presented_on && ` · apresentada em ${bill.presented_on.split("-").reverse().join("/")}`}
        {bill.current_body && ` · ${bill.current_body}`}
        {bill.laws.map((l) => ` · ${l.kind_name} ${l.number}`).join("")}
      </p>
    </li>
  );
}
