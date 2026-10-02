import { FormEvent } from "react";
import { ActType, Organ, Source } from "./api";
import { SearchState, THEME_LABEL, withSource } from "./searchState";
import { SOURCE_LABEL } from "./types";

const TYPES: { value: ActType | ""; label: string }[] = [
  { value: "", label: "Tudo" },
  { value: "nomeacao", label: "Nomeações" },
  { value: "exoneracao", label: "Exonerações" },
  { value: "contrato", label: "Contratos" },
  { value: "aditivo", label: "Aditivos" },
  { value: "licitacao", label: "Licitações" },
  { value: "dispensa", label: "Sem licitação" },
  { value: "decreto", label: "Decretos" },
  { value: "lei", label: "Leis" },
  { value: "resolucao", label: "Resoluções" },
  { value: "prestacao_contas", label: "Prestações de contas" },
  { value: "licenca_ambiental", label: "Licenças ambientais" },
  { value: "despacho", label: "Despachos" },
  { value: "edital", label: "Editais" },
];

export interface FiltersProps {
  draft: SearchState;
  setDraft: (s: SearchState) => void;
  go: (s: SearchState) => void;
  organs: Organ[];
  loading: boolean;
  advancedOpen: boolean;
}

function trimmed(s: SearchState): SearchState {
  return { ...s, q: s.q.trim() };
}

export function SearchForm({ draft, setDraft, onSubmit, loading, size }: {
  draft: SearchState;
  setDraft: (s: SearchState) => void;
  onSubmit: (e: FormEvent) => void;
  loading: boolean;
  size: "md" | "lg";
}) {
  return (
    <form className={`search search-${size}`} onSubmit={onSubmit} role="search">
      <label htmlFor="q" className="field-label">Buscar nos Diários Oficiais</label>
      <div className="search-row">
        <input
          id="q"
          type="search"
          value={draft.q}
          onChange={(e) => setDraft({ ...draft, q: e.target.value })}
          placeholder="Nome, CNPJ, empresa ou assunto"
          autoComplete="off"
        />
        <button type="submit" className="primary" disabled={loading}>{loading ? "Buscando" : "Buscar"}</button>
      </div>
    </form>
  );
}

export function TypeChips({ draft, go }: Pick<FiltersProps, "draft" | "go">) {
  return (
    <div className="types" role="group" aria-label="Tipo de ato">
      {TYPES.map((t) => (
        <button
          key={t.value || "all"}
          type="button"
          aria-pressed={draft.type === t.value}
          onClick={() => go({ ...trimmed(draft), type: t.value, page: 1 })}
        >
          {t.label}
        </button>
      ))}
    </div>
  );
}

export function TypeList({ draft, go }: Pick<FiltersProps, "draft" | "go">) {
  return (
    <fieldset className="type-list">
      <legend className="field-label">Tipo de ato</legend>
      {TYPES.map((t) => (
        <button
          key={t.value || "all"}
          type="button"
          aria-pressed={draft.type === t.value}
          onClick={() => go({ ...trimmed(draft), type: t.value, page: 1 })}
        >
          {t.label}
        </button>
      ))}
    </fieldset>
  );
}

export function ScopeSelects({ draft, go, organs }: Pick<FiltersProps, "draft" | "go" | "organs">) {
  return (
    <>
      <label className="field">
        <span className="field-label">Diário</span>
        <select value={draft.source} onChange={(e) => go(withSource(trimmed(draft), e.target.value))}>
          <option value="">Prefeitura e Câmara</option>
          {(Object.keys(SOURCE_LABEL) as Source[]).map((src) => (
            <option key={src} value={src}>{SOURCE_LABEL[src]}</option>
          ))}
        </select>
      </label>
      {organs.length > 0 && draft.source !== "diario_camara" && (
        <label className="field">
          <span className="field-label">Órgão</span>
          <select value={draft.organ} onChange={(e) => go({ ...trimmed(draft), organ: e.target.value, page: 1 })}>
            <option value="">Todos os órgãos</option>
            {organs.map((o) => (
              <option key={o.acronym} value={o.acronym}>
                {o.name ? `${o.acronym} · ${o.name}` : o.acronym} ({o.acts.toLocaleString("pt-BR")} atos)
              </option>
            ))}
          </select>
        </label>
      )}
      <label className="field">
        <span className="field-label">Tema</span>
        <select value={draft.theme} onChange={(e) => go({ ...trimmed(draft), theme: e.target.value, page: 1 })}>
          <option value="">Todos os temas</option>
          {Object.entries(THEME_LABEL).map(([slug, label]) => <option key={slug} value={slug}>{label}</option>)}
        </select>
      </label>
    </>
  );
}

export function MoreFilters({ draft, setDraft, go, loading, advancedOpen }: Omit<FiltersProps, "organs">) {
  return (
    <details className="more-filters" open={advancedOpen}>
      <summary>Mais filtros: período e valor</summary>
      <form onSubmit={(e) => { e.preventDefault(); go({ ...trimmed(draft), page: 1 }); }}>
        <div className="filter-grid">
          <label className="field">
            <span className="field-label">Publicado a partir de</span>
            <input type="date" value={draft.from} onChange={(e) => setDraft({ ...draft, from: e.target.value })} />
          </label>
          <label className="field">
            <span className="field-label">Publicado até</span>
            <input type="date" value={draft.to} onChange={(e) => setDraft({ ...draft, to: e.target.value })} />
          </label>
          <label className="field">
            <span className="field-label">Cita valor a partir de (R$)</span>
            <input className="numeric" inputMode="decimal" placeholder="1.000,00" value={draft.min}
              onChange={(e) => setDraft({ ...draft, min: e.target.value })} />
          </label>
          <label className="field">
            <span className="field-label">Cita valor até (R$)</span>
            <input className="numeric" inputMode="decimal" placeholder="500.000,00" value={draft.max}
              onChange={(e) => setDraft({ ...draft, max: e.target.value })} />
          </label>
        </div>
        <div className="filter-actions">
          <button type="submit" disabled={loading}>Aplicar filtros</button>
          <button type="button" className="quiet"
            onClick={() => go({ ...draft, from: "", to: "", min: "", max: "", page: 1 })}>
            Limpar
          </button>
        </div>
        <p className="fineprint">
          O filtro de valor acha atos que citam ao menos um valor na faixa (global, mensal ou unitário).
        </p>
      </form>
    </details>
  );
}
