import { FormEvent, useState } from "react";
import {
  ActHit, ActType, confirmSubscription, searchActs, subscribe, unsubscribe,
} from "./api";
import { CompanyPage } from "./CompanyPage";
import { TYPE_LABEL } from "./types";

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

export function App() {
  const path = window.location.pathname;
  const token = new URLSearchParams(window.location.search).get("token") ?? "";
  if (path === "/confirmar") return <TokenPage kind="confirm" token={token} />;
  if (path === "/cancelar") return <TokenPage kind="cancel" token={token} />;
  const company = path.match(/^\/empresa\/([\d./-]+)$/);
  if (company) return <CompanyPage cnpj={decodeURIComponent(company[1])} />;
  return <SearchPage />;
}

function SearchPage() {
  const [query, setQuery] = useState("");
  const [type, setType] = useState<ActType | "">("");
  const [hits, setHits] = useState<ActHit[] | null>(null);
  const [total, setTotal] = useState(0);
  const [searched, setSearched] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function run(q: string, t: ActType | "") {
    setLoading(true);
    setError("");
    try {
      const res = await searchActs(q, t);
      setHits(res.items);
      setTotal(res.total);
      setSearched(q);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    run(query.trim(), type);
  }

  function onType(t: ActType | "") {
    setType(t);
    if (hits !== null) run(query.trim(), t);
  }

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
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Nome, CNPJ, empresa ou assunto"
          autoComplete="off"
        />
        <button type="submit" disabled={loading}>{loading ? "Buscando" : "Buscar"}</button>
      </form>

      <div className="types" role="group" aria-label="Tipo de ato">
        {TYPES.map((t) => (
          <button
            key={t.value || "all"}
            type="button"
            aria-pressed={type === t.value}
            onClick={() => onType(t.value)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {error && <p className="notice notice-error">{error}</p>}

      {hits !== null && (
        <section className="results" aria-live="polite">
          <p className="count">
            {total === 0
              ? "Nenhum ato encontrado. Tente outro termo ou remova o filtro de tipo."
              : `${total} ${total === 1 ? "ato encontrado" : "atos encontrados"}`}
          </p>
          <ol>
            {hits.map((h) => <Result key={h.id} hit={h} />)}
          </ol>
          {searched.length >= 3 && <AlertForm query={searched} />}
        </section>
      )}
    </main>
  );
}

export function Result({ hit }: { hit: ActHit }) {
  const date = new Date(hit.published_at + "T12:00:00").toLocaleDateString("pt-BR");
  return (
    <li className="result">
      <p className="meta">
        <span className={`tag tag-${hit.type}`}>{TYPE_LABEL[hit.type]}</span>
        <a href={hit.source_url} target="_blank" rel="noopener" title="Abrir o PDF da edição original">
          Edição {hit.edition_number || "s/n"}, {date}
        </a>
      </p>
      <h2>{hit.title}</h2>
      <p className="snippet"><Highlighted text={hit.snippet} /></p>
      {hit.cnpjs.length > 0 && (
        <p className="cnpjs">
          Empresas citadas:{" "}
          {hit.cnpjs.map((c) => (
            <a key={c} className="cnpj" href={`/empresa/${c}`} title="Ver todos os atos desta empresa">{c}</a>
          ))}
        </p>
      )}
    </li>
  );
}

// A API marca os termos encontrados com ⟦ ⟧; renderizamos sem usar HTML cru.
export function Highlighted({ text }: { text: string }) {
  const parts = text.split(/(⟦[^⟧]*⟧)/g);
  return (
    <>
      {parts.map((p, i) =>
        p.startsWith("⟦") ? <mark key={i}>{p.slice(1, -1)}</mark> : <span key={i}>{p}</span>,
      )}
    </>
  );
}

function AlertForm({ query }: { query: string }) {
  const [email, setEmail] = useState("");
  const [state, setState] = useState<"idle" | "sending" | "sent" | "error">("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setState("sending");
    try {
      await subscribe(email, query);
      setState("sent");
    } catch (err) {
      setError((err as Error).message);
      setState("error");
    }
  }

  if (state === "sent") {
    return (
      <p className="notice">
        Enviamos um link para {email}. O alerta começa a valer depois que você confirmar.
      </p>
    );
  }

  return (
    <form className="alert" onSubmit={onSubmit}>
      <h2>Avisar quando “{query}” aparecer de novo</h2>
      <p>Você recebe um e-mail no dia em que uma nova edição mencionar este termo.</p>
      <div className="alert-row">
        <label htmlFor="email" className="visually-hidden">Seu e-mail</label>
        <input id="email" type="email" required value={email}
          onChange={(e) => setEmail(e.target.value)} placeholder="seu@email.com" />
        <button type="submit" disabled={state === "sending"}>Criar alerta</button>
      </div>
      {state === "error" && <p className="notice notice-error">{error}</p>}
    </form>
  );
}

function TokenPage({ kind, token }: { kind: "confirm" | "cancel"; token: string }) {
  const [state, setState] = useState<"idle" | "done" | "error">("idle");
  const [message, setMessage] = useState("");
  const confirm = kind === "confirm";

  async function onClick() {
    try {
      if (confirm) {
        const s = await confirmSubscription(token);
        setMessage(`Alerta confirmado. Você será avisado quando “${s.query}” aparecer.`);
      } else {
        await unsubscribe(token);
        setMessage("Alerta cancelado. Você não receberá mais e-mails sobre ele.");
      }
      setState("done");
    } catch (err) {
      setMessage((err as Error).message);
      setState("error");
    }
  }

  return (
    <main className="page narrow">
      <h1>{confirm ? "Confirmar alerta" : "Cancelar alerta"}</h1>
      {!token && <p className="notice notice-error">Link incompleto. Abra o link do e-mail novamente.</p>}
      {token && state === "idle" && (
        <button className="primary" onClick={onClick}>
          {confirm ? "Confirmar alerta" : "Cancelar alerta"}
        </button>
      )}
      {state !== "idle" && (
        <p className={`notice ${state === "error" ? "notice-error" : ""}`}>{message}</p>
      )}
      <p><a href="/">Voltar para a busca</a></p>
    </main>
  );
}
