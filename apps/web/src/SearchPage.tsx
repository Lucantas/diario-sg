import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { ActType, Organ, SearchResponse, listOrgans, searchActs, subscribe } from "./api";
import { Result } from "./components";
import { SearchState, apiParams, exportUrl, feedUrl, hasSearch, queryFromState, stateFromQuery } from "./searchState";

const PAGE_SIZE = 20;

const EXPORT_LIMIT = 10000;

const INVALID_VALUE = "Valor inválido. Escreva só números, como 1.500 ou 1.500,50.";

const TYPES: { value: ActType | ""; label: string }[] = [
  { value: "", label: "Tudo" },
  { value: "nomeacao", label: "Nomeações" },
  { value: "exoneracao", label: "Exonerações" },
  { value: "contrato", label: "Contratos" },
  { value: "aditivo", label: "Aditivos" },
  { value: "licitacao", label: "Licitações" },
  { value: "dispensa", label: "Sem licitação" },
  { value: "decreto", label: "Decretos" },
  { value: "despacho", label: "Despachos" },
  { value: "edital", label: "Editais" },
];

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

  function go(next: SearchState) {
    if (apiParams(next, PAGE_SIZE) === null) {
      setError(INVALID_VALUE);
      return;
    }
    setState(next);
    setDraft(next);
    window.history.pushState(null, "", "/" + queryFromState(next));
    load(next);
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

  return (
    <main className="page">
      <header className="masthead">
        <h1>Diário Oficial de São Gonçalo, pesquisável.</h1>
        <p className="lede">
          Nomeações, contratos, licitações e decretos publicados pela prefeitura,
          com busca por nome, empresa ou assunto.
        </p>
      </header>

      <form className="search" onSubmit={onSubmit} role="search">
        <label htmlFor="q" className="visually-hidden">Buscar no Diário Oficial</label>
        <input
          id="q"
          type="search"
          value={draft.q}
          onChange={(e) => setDraft({ ...draft, q: e.target.value })}
          placeholder="Nome, CNPJ, empresa ou assunto"
          autoComplete="off"
        />
        <button type="submit" disabled={loading}>{loading ? "Buscando" : "Buscar"}</button>
      </form>
      <p className="hint">
        Entre aspas ("josé da silva") só a frase exata. OU junta termos (merenda OU alimentação);
        -termo exclui (limpeza -urbana).
      </p>

      <div className="types" role="group" aria-label="Tipo de ato">
        {TYPES.map((t) => (
          <button
            key={t.value || "all"}
            type="button"
            aria-pressed={draft.type === t.value}
            onClick={() => go({ ...draft, q: draft.q.trim(), type: t.value, page: 1 })}
          >
            {t.label}
          </button>
        ))}
      </div>

      {organs.length > 0 && (
        <div className="organ-filter">
          <label htmlFor="organ">Órgão</label>
          <select id="organ" value={draft.organ} onChange={(e) => go({ ...draft, q: draft.q.trim(), organ: e.target.value, page: 1 })}>
            <option value="">Todos os órgãos</option>
            {organs.map((o) => (
              <option key={o.acronym} value={o.acronym}>
                {o.name ? `${o.acronym} · ${o.name}` : o.acronym} ({o.acts.toLocaleString("pt-BR")})
              </option>
            ))}
          </select>
        </div>
      )}

      <details className="more-filters" open={hasAdvancedFilters(state)}>
        <summary>Mais filtros: período e valor</summary>
        <form onSubmit={(e) => { e.preventDefault(); go({ ...draft, q: draft.q.trim(), page: 1 }); }}>
          <div className="filter-grid">
            <label>
              Publicado a partir de
              <input type="date" value={draft.from} onChange={(e) => setDraft({ ...draft, from: e.target.value })} />
            </label>
            <label>
              Publicado até
              <input type="date" value={draft.to} onChange={(e) => setDraft({ ...draft, to: e.target.value })} />
            </label>
            <label>
              Cita valor a partir de (R$)
              <input inputMode="decimal" placeholder="1.000,00" value={draft.min}
                onChange={(e) => setDraft({ ...draft, min: e.target.value })} />
            </label>
            <label>
              Cita valor até (R$)
              <input inputMode="decimal" placeholder="500.000,00" value={draft.max}
                onChange={(e) => setDraft({ ...draft, max: e.target.value })} />
            </label>
          </div>
          <div className="filter-actions">
            <button type="submit" disabled={loading}>Aplicar filtros</button>
            <button type="button" className="secondary"
              onClick={() => go({ ...draft, from: "", to: "", min: "", max: "", page: 1 })}>
              Limpar
            </button>
          </div>
          <p className="fineprint">
            O filtro de valor acha atos que citam ao menos um valor na faixa (global, mensal ou unitário).
          </p>
        </form>
      </details>

      {error && <p className="notice notice-error">{error}</p>}

      {result !== null && (
        <section className="results" aria-live="polite" ref={resultsRef}>
          <p className="count">
            {result.total === 0
              ? "Nenhum ato encontrado. Tente outro termo ou remova filtros."
              : `${result.total.toLocaleString("pt-BR")} ${result.total === 1 ? "ato encontrado" : "atos encontrados"}`}
          </p>
          {result.total > 0 && <ExportLinks state={state} total={result.total} />}
          <ol>
            {result.items.map((h) => <Result key={h.id} hit={h} />)}
          </ol>
          {totalPages > 1 && (
            <nav className="pager" aria-label="Páginas">
              <button type="button" className="secondary" disabled={loading || state.page <= 1}
                onClick={() => goToPage(state.page - 1)}>Anterior</button>
              <span>Página {state.page} de {totalPages.toLocaleString("pt-BR")}</span>
              <button type="button" className="secondary" disabled={loading || state.page >= totalPages}
                onClick={() => goToPage(state.page + 1)}>Próxima</button>
            </nav>
          )}
          {state.q.length >= 3 && <AlertForm query={state.q} />}
          <FeedLink state={state} />
        </section>
      )}

      <footer className="site-footer">
        <a href="/dados">Dados abertos: a base inteira para baixar</a>
        <a href="/mcp">Pergunte pela sua IA (MCP)</a>
      </footer>
    </main>
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

function AlertForm({ query }: { query: string }) {
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState<"idle" | "sending" | "sent" | "error">("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setStatus("sending");
    try {
      await subscribe(email, query);
      setStatus("sent");
    } catch (err) {
      setError((err as Error).message);
      setStatus("error");
    }
  }

  if (status === "sent") {
    return (
      <p className="notice">
        Enviamos um link para {email}. O alerta começa a valer depois que você confirmar.
      </p>
    );
  }

  return (
    <form className="alert" onSubmit={onSubmit}>
      <h2>Avisar quando “{query}” aparecer de novo</h2>
      <p>Você recebe um e-mail no dia em que uma nova edição mencionar este termo. Os filtros não entram no alerta.</p>
      <div className="alert-row">
        <label htmlFor="email" className="visually-hidden">Seu e-mail</label>
        <input id="email" type="email" required value={email}
          onChange={(e) => setEmail(e.target.value)} placeholder="seu@email.com" />
        <button type="submit" disabled={status === "sending"}>Criar alerta</button>
      </div>
      {status === "error" && <p className="notice notice-error">{error}</p>}
    </form>
  );
}

