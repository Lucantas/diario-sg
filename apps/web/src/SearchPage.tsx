import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { Organ, SearchResponse, listOrgans, searchActs, subscribe } from "./api";
import { AlertForm } from "./AlertForm";
import { HeaderSearch, SectionCards, SiteHeader } from "./Brand";
import { Result } from "./components";
import { FilterBar, FilterPills, FiltersDialog, SearchForm } from "./SearchFilters";
import {
  FilterKey, SearchState, activeFilters, alertFilterNames, alertFilters, apiParams, canAlert, exportUrl, feedUrl, hasSearch,
  isRangeFilter, queryFromState, stateFromQuery, withoutFilter, withoutFilters,
} from "./searchState";

const PAGE_SIZE = 20;

const EXPORT_LIMIT = 10000;

const INVALID_VALUE = "Valor inválido. Escreva só números, como 1.500 ou 1.500,50.";

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
  const errorNotice = error && <p className="notice notice-error" role="alert">{error}</p>;
  const applied = isHome ? draft : state;
  const chips = activeFilters(applied);
  const rangeChips = chips.filter((c) => isRangeFilter(c.key));

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
      <>
        <SiteHeader />
        <main className="page page-wide search-home">
          <h1>Diários Oficiais de São Gonçalo, pesquisáveis.</h1>
          <p className="lede">
            Nomeações, contratos, licitações e decretos publicados pela prefeitura e pela
            Câmara Municipal, com busca por nome, empresa ou assunto.
          </p>
          <div className="search-home-form">
            <SearchForm draft={draft} setDraft={setDraft} onSubmit={onSubmit} loading={loading} />
            {filterBar()}
            {!filtersOpen && errorNotice}
            <SyntaxHint />
          </div>
          <SectionCards />
          {dialog}
        </main>
      </>
    );
  }

  const headerSearch = (
    <HeaderSearch controlled={{ value: draft.q, onChange: (q) => setDraft({ ...draft, q }), onSubmit, loading }} />
  );

  return (
    <>
      <SiteHeader search={headerSearch} />
      <main className="page page-wide search-results">
        <h1 className="visually-hidden">Resultados da busca nos Diários Oficiais</h1>
        <FilterPills
          value={state}
          onChange={(next) => go({ ...next, q: draft.q.trim() })}
          organs={organs}
          rangeCount={rangeChips.length}
          onOpenDialog={() => setFiltersOpen(true)}
          onClear={chips.length > 0 ? () => go(withoutFilters(state)) : null}
        />
        {rangeChips.length > 0 && <FilterBar filters={rangeChips} onRemove={removeFilter} className="filter-bar-desktop" />}
        {filterBar("filter-bar-mobile")}
        <section className="results" aria-live="polite" ref={resultsRef}>
          {!filtersOpen && errorNotice}
          {result === null && loading && <p className="count">Carregando…</p>}
          {result !== null && result.total === 0 && (
            <div className="card empty-state">
              <h2>Nenhum ato encontrado</h2>
              <p>Tente outro termo ou remova filtros.</p>
            </div>
          )}
          {result !== null && result.total > 0 && (
            <>
              <div className="results-heading">
                <h2>
                  {result.total.toLocaleString("pt-BR")} {result.total === 1 ? "ato encontrado" : "atos encontrados"}
                  {state.q && <span className="results-heading-query"> para “{state.q}”</span>}
                </h2>
                <ExportLinks state={state} total={result.total} />
              </div>
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
            </>
          )}
          {result !== null && (canAlert(state) ? <SearchAlert state={state} /> : <FeedLink state={state} />)}
        </section>
        {dialog}
      </main>
    </>
  );
}

const SYNTAX_TIPS: [string, string][] = [
  ['"josé da silva"', "Entre aspas: encontra só essa frase exata, nessa ordem."],
  ["merenda OU alimentação", "Com OU: encontra atos que tenham uma palavra ou a outra."],
  ["limpeza -urbana", "Com um traço antes: deixa de fora os atos com essa palavra."],
];

function SyntaxHint() {
  return (
    <div className="hint">
      <p>Dicas para refinar a busca:</p>
      <ul>
        {SYNTAX_TIPS.map(([example, text]) => (
          <li key={example}><code>{example}</code><span>{text}</span></li>
        ))}
      </ul>
    </div>
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
      <span>Baixar o resultado</span>
      <a className="pill pill-sm" href={csv} download>CSV</a>
      <a className="pill pill-sm" href={json} download>JSON</a>
      {total > EXPORT_LIMIT && <span>(só os {EXPORT_LIMIT.toLocaleString("pt-BR")} primeiros atos)</span>}
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
