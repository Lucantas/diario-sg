import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { Organ, SearchResponse, listOrgans, searchActs, subscribe } from "./api";
import { AlertForm } from "./AlertForm";
import { Logo } from "./Brand";
import { Result } from "./components";
import { FilterBar, FiltersDialog, MoreFilters, ScopeSelects, SearchForm } from "./SearchFilters";
import {
  FilterKey, SearchState, activeFilters, alertFilterNames, alertFilters, apiParams, canAlert, exportUrl, feedUrl, hasSearch,
  queryFromState, stateFromQuery, withoutFilter, withoutFilters,
} from "./searchState";

const PAGE_SIZE = 20;

const EXPORT_LIMIT = 10000;

const INVALID_VALUE = "Valor inválido. Escreva só números, como 1.500 ou 1.500,50.";

function hasAdvancedFilters(s: SearchState) {
  return Boolean(s.from || s.to || s.min || s.max);
}

export function SearchPage() {
  const [state, setState] = useState<SearchState>(() => stateFromQuery(window.location.search));
  const [draft, setDraft] = useState<SearchState>(state);
  const [organs, setOrgans] = useState<Organ[]>([]);
  const [result, setResult] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [filtersOpen, setFiltersOpen] = useState(false);
  const resultsRef = useRef<HTMLElement>(null);

  const load = useCallback(async (s: SearchState) => {
    const params = apiParams(s, PAGE_SIZE);
    if (!params) {
      setError(INVALID_VALUE);
      return;
    }
    setLoading(true);
    setError("");
    try {
      setResult(await searchActs(params));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    listOrgans().then((res) => setOrgans(res.items)).catch(() => setOrgans([]));
  }, []);

  useEffect(() => {
    const initial = stateFromQuery(window.location.search);
    if (hasSearch(initial)) load(initial);
    function onPop() {
      const s = stateFromQuery(window.location.search);
      setState(s);
      setDraft(s);
      setError("");
      if (hasSearch(s)) load(s);
      else setResult(null);
    }
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, [load]);

  function go(next: SearchState): boolean {
    if (apiParams(next, PAGE_SIZE) === null) {
      setError(INVALID_VALUE);
      return false;
    }
    setState(next);
    setDraft(next);
    window.history.pushState(null, "", "/" + queryFromState(next));
    load(next);
    return true;
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    go({ ...draft, q: draft.q.trim(), page: 1 });
  }

  function goToPage(page: number) {
    go({ ...state, page });
    resultsRef.current?.scrollIntoView({ block: "start" });
  }

  const totalPages = result ? Math.max(1, Math.ceil(result.total / PAGE_SIZE)) : 1;
  const isHome = !hasSearch(state);
  const searchForm = <SearchForm draft={draft} setDraft={setDraft} onSubmit={onSubmit} loading={loading} size={isHome ? "lg" : "md"} />;
  const errorNotice = error && <p className="notice notice-error" role="alert">{error}</p>;
  const applied = isHome ? draft : state;
  const chips = activeFilters(applied);

  function removeFilter(key: FilterKey) {
    if (isHome) setDraft(withoutFilter(draft, key));
    else go(withoutFilter(state, key));
  }

  function closeFilters() {
    setFiltersOpen(false);
    if (!isHome) setDraft({ ...state, q: draft.q });
  }

  function applyFilters() {
    const next = { ...draft, q: draft.q.trim(), page: 1 };
    if (apiParams(next, PAGE_SIZE) === null) {
      setError(INVALID_VALUE);
      return;
    }
    setError("");
    if (isHome || go(next)) setFiltersOpen(false);
  }

  const filterBar = (className?: string) => (
    <FilterBar filters={chips} onOpen={() => setFiltersOpen(true)} onRemove={removeFilter} className={className} />
  );
  const dialog = (
    <FiltersDialog
      open={filtersOpen}
      value={draft}
      onChange={setDraft}
      organs={organs}
      onClose={closeFilters}
      onClear={() => setDraft(withoutFilters(draft))}
      onApply={applyFilters}
      applyLabel={isHome ? "Aplicar filtros" : "Aplicar e buscar"}
      canClear={activeFilters(draft).length > 0}
      error={filtersOpen && errorNotice}
    />
  );

  if (isHome) {
    return (
      <main className="page search-home">
        <Logo size="lg" href="/" />
        <h1>Diários Oficiais de São Gonçalo, pesquisáveis.</h1>
        <p className="lede">
          Nomeações, contratos, licitações e decretos publicados pela prefeitura e pela
          Câmara Municipal, com busca por nome, empresa ou assunto.
        </p>
        {searchForm}
        <SyntaxHint />
        {filterBar()}
        {!filtersOpen && errorNotice}
        {dialog}
      </main>
    );
  }

  return (
    <main className="page page-wide search-results">
      <h1 className="visually-hidden">Resultados da busca nos Diários Oficiais</h1>
      <div className="results-top">
        <Logo size="md" href="/" />
        {searchForm}
      </div>
      {filterBar("filter-bar-mobile")}
      <div className="results-layout">
        <aside className="filters" aria-label="Filtros">
          <div className="filters-head">
            <h2>Filtros</h2>
            {chips.length > 0 && (
              <button type="button" className="link-button" onClick={() => go(withoutFilters(state))}>Limpar</button>
            )}
          </div>
          <ScopeSelects value={state} onChange={(next) => go({ ...next, q: draft.q.trim() })} organs={organs} />
          <MoreFilters
            value={draft}
            onChange={setDraft}
            onApply={() => go({ ...draft, q: draft.q.trim(), page: 1 })}
            loading={loading}
            open={hasAdvancedFilters(state)}
          />
        </aside>
        <section className="results" aria-live="polite" ref={resultsRef}>
          {!filtersOpen && errorNotice}
          {result === null && loading && <p className="count">Carregando…</p>}
          {result !== null && (
            <>
              <div className="results-heading">
                <h2>
                  {result.total === 0
                    ? "Nenhum ato encontrado"
                    : `${result.total.toLocaleString("pt-BR")} ${result.total === 1 ? "ato encontrado" : "atos encontrados"}`}
                </h2>
                {result.total > 0 && <ExportLinks state={state} total={result.total} />}
              </div>
              {result.total === 0 && <p className="count">Tente outro termo ou remova filtros.</p>}
              <ol className="act-list">
                {result.items.map((h) => <Result key={h.id} hit={h} />)}
              </ol>
              {totalPages > 1 && (
                <nav className="pager" aria-label="Páginas">
                  <button type="button" disabled={loading || state.page <= 1}
                    onClick={() => goToPage(state.page - 1)}>Anterior</button>
                  <span>Página {state.page.toLocaleString("pt-BR")} de {totalPages.toLocaleString("pt-BR")}</span>
                  <button type="button" disabled={loading || state.page >= totalPages}
                    onClick={() => goToPage(state.page + 1)}>Próxima</button>
                </nav>
              )}
              {canAlert(state) ? <SearchAlert state={state} /> : <FeedLink state={state} />}
            </>
          )}
        </section>
      </div>
      {dialog}
    </main>
  );
}

function SyntaxHint() {
  return (
    <p className="hint">
      Entre aspas ("josé da silva") só a frase exata. OU junta termos (merenda OU alimentação);{" "}
      -termo exclui (limpeza -urbana).
    </p>
  );
}

function SearchAlert({ state }: { state: SearchState }) {
  const filters = alertFilterNames(state);
  const title = state.q ? `Avisar quando “${state.q}” aparecer de novo` : "Avisar quando sair ato novo com estes filtros";
  const scope = state.q ? "mencionar este termo" : "trouxer um ato assim";
  const filterNote = filters ? ` Filtros do alerta: ${filters}.` : "";
  return (
    <AlertForm
      title={title}
      description={`Você recebe um e-mail no dia em que uma nova edição ${scope}.${filterNote} Período e valor não entram no alerta.`}
      onSubscribe={(email) => subscribe(email, state.q, alertFilters(state))}
    >
      <FeedLink state={state} />
    </AlertForm>
  );
}

function ExportLinks({ state, total }: { state: SearchState; total: number }) {
  const csv = exportUrl(state, "csv");
  const json = exportUrl(state, "json");
  if (!csv || !json) return null;
  return (
    <p className="export">
      Baixar o resultado: <a href={csv} download>CSV</a> · <a href={json} download>JSON</a>
      {total > EXPORT_LIMIT && ` (só os ${EXPORT_LIMIT.toLocaleString("pt-BR")} primeiros atos)`}
    </p>
  );
}

function FeedLink({ state }: { state: SearchState }) {
  const path = feedUrl(state);
  if (!path) return null;
  return (
    <p className="feed">
      Prefere RSS? <a href={window.location.origin + path}>Assine o feed desta busca</a>: os 50 atos mais recentes,
      sem precisar de e-mail.
    </p>
  );
}
