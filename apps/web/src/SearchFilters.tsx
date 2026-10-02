import { FormEvent, ReactNode, useEffect, useRef } from "react";
import { Organ, Source } from "./api";
import { ActiveFilter, FilterKey, SearchState, THEME_LABEL, TYPE_OPTIONS, withSource } from "./searchState";
import { SOURCE_LABEL } from "./types";

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

interface FieldsProps {
  value: SearchState;
  onChange: (s: SearchState) => void;
}

export function ScopeSelects({ value, onChange, organs }: FieldsProps & { organs: Organ[] }) {
  const set = (patch: Partial<SearchState>) => onChange({ ...value, ...patch, page: 1 });
  return (
    <>
      <label className="field">
        <span className="field-label">Tipo de ato</span>
        <select value={value.type} onChange={(e) => set({ type: e.target.value as SearchState["type"] })}>
          {TYPE_OPTIONS.map((t) => <option key={t.value || "all"} value={t.value}>{t.label}</option>)}
        </select>
      </label>
      <label className="field">
        <span className="field-label">Diário</span>
        <select value={value.source} onChange={(e) => onChange(withSource(value, e.target.value))}>
          <option value="">Prefeitura e Câmara</option>
          {(Object.keys(SOURCE_LABEL) as Source[]).map((src) => (
            <option key={src} value={src}>{SOURCE_LABEL[src]}</option>
          ))}
        </select>
      </label>
      {organs.length > 0 && value.source !== "diario_camara" && (
        <label className="field">
          <span className="field-label">Órgão</span>
          <select value={value.organ} onChange={(e) => set({ organ: e.target.value })}>
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
        <select value={value.theme} onChange={(e) => set({ theme: e.target.value })}>
          <option value="">Todos os temas</option>
          {Object.entries(THEME_LABEL).map(([slug, label]) => <option key={slug} value={slug}>{label}</option>)}
        </select>
      </label>
    </>
  );
}

const VALUE_NOTE = "O filtro de valor acha atos que citam ao menos um valor na faixa (global, mensal ou unitário).";

function DateInput({ label, field, value, onChange }: FieldsProps & { label: string; field: "from" | "to" }) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      <input type="date" value={value[field]} onChange={(e) => onChange({ ...value, [field]: e.target.value })} />
    </label>
  );
}

function ValueInput({ label, field, placeholder, value, onChange }: FieldsProps & {
  label: string;
  field: "min" | "max";
  placeholder: string;
}) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      <input className="numeric" inputMode="decimal" placeholder={placeholder} value={value[field]}
        onChange={(e) => onChange({ ...value, [field]: e.target.value })} />
    </label>
  );
}

export function MoreFilters({ value, onChange, onApply, loading, open }: FieldsProps & {
  onApply: () => void;
  loading: boolean;
  open: boolean;
}) {
  return (
    <details className="more-filters" open={open}>
      <summary>Período e valor</summary>
      <form onSubmit={(e) => { e.preventDefault(); onApply(); }}>
        <DateInput label="Publicado a partir de" field="from" value={value} onChange={onChange} />
        <DateInput label="Publicado até" field="to" value={value} onChange={onChange} />
        <ValueInput label="Cita valor a partir de (R$)" field="min" placeholder="1.000,00" value={value} onChange={onChange} />
        <ValueInput label="Cita valor até (R$)" field="max" placeholder="500.000,00" value={value} onChange={onChange} />
        <button type="submit" disabled={loading}>Aplicar filtros</button>
        <p className="fineprint">{VALUE_NOTE}</p>
      </form>
    </details>
  );
}

export function FilterBar({ filters, onOpen, onRemove, className = "" }: {
  filters: ActiveFilter[];
  onOpen: () => void;
  onRemove: (key: FilterKey) => void;
  className?: string;
}) {
  return (
    <div className={`filter-bar ${className}`.trim()}>
      <button type="button" className="filters-button" aria-haspopup="dialog" onClick={onOpen}>
        <span className="filters-icon" aria-hidden="true"><span /><span /><span /></span>
        {filters.length ? `Filtros (${filters.length})` : "Filtros"}
      </button>
      {filters.map((f) => (
        <button key={f.key} type="button" className="filter-chip" aria-label={`Remover filtro: ${f.label}`}
          onClick={() => onRemove(f.key)}>
          <span>{f.label}</span>
          <span aria-hidden="true" className="filter-chip-x">×</span>
        </button>
      ))}
    </div>
  );
}

export function FiltersDialog({ open, value, onChange, organs, onClose, onClear, onApply, applyLabel, canClear, error }:
  FieldsProps & {
    open: boolean;
    organs: Organ[];
    onClose: () => void;
    onClear: () => void;
    onApply: () => void;
    applyLabel: string;
    canClear: boolean;
    error: ReactNode;
  }) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      className="filters-dialog"
      aria-labelledby="filters-title"
      onClose={onClose}
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <form className="filters-dialog-body" onSubmit={(e) => { e.preventDefault(); onApply(); }}>
        <header className="filters-dialog-head">
          <h2 id="filters-title">Filtros</h2>
          <button type="button" className="link-button" aria-label="Fechar filtros" onClick={onClose}>Fechar</button>
        </header>
        <div className="filters-dialog-fields">
          <ScopeSelects value={value} onChange={onChange} organs={organs} />
          <fieldset className="pair">
            <legend className="field-label">Período de publicação</legend>
            <DateInput label="A partir de" field="from" value={value} onChange={onChange} />
            <DateInput label="Até" field="to" value={value} onChange={onChange} />
          </fieldset>
          <fieldset className="pair">
            <legend className="field-label">Cita valor (R$)</legend>
            <ValueInput label="A partir de" field="min" placeholder="1.000,00" value={value} onChange={onChange} />
            <ValueInput label="Até" field="max" placeholder="500.000,00" value={value} onChange={onChange} />
            <p className="fineprint">{VALUE_NOTE}</p>
          </fieldset>
          {error}
        </div>
        <footer className="filters-dialog-foot">
          <button type="button" className="link-button" disabled={!canClear} onClick={onClear}>Limpar</button>
          <button type="submit" className="primary">{applyLabel}</button>
        </footer>
      </form>
    </dialog>
  );
}
