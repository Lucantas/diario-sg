import { FormEvent, KeyboardEvent, ReactNode, useEffect, useRef, useState } from "react";
import { Organ, Source } from "./api";
import { ActiveFilter, FilterKey, SearchState, THEME_LABEL, TYPE_OPTIONS, withSource } from "./searchState";
import { SOURCE_LABEL } from "./types";
import { SearchIcon } from "./Brand";
import { suggestionsStatus } from "./discovery";
import { SuggestionsPanel, useSuggestions } from "./Suggestions";

export function SearchForm({ draft, setDraft, onSubmit, loading, organs }: {
  draft: SearchState;
  setDraft: (s: SearchState) => void;
  onSubmit: (e: FormEvent) => void;
  loading: boolean;
  organs: Organ[];
}) {
  const [open, setOpen] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const groups = useSuggestions(draft.q, organs);
  const showing = open && groups.length > 0;

  function onKeyDown(e: KeyboardEvent<HTMLFormElement>) {
    if (e.key === "Escape" && showing) {
      setOpen(false);
      inputRef.current?.focus();
    }
  }

  function onInputKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key !== "ArrowDown" || !showing) return;
    e.preventDefault();
    document.querySelector<HTMLAnchorElement>("#sugestoes a")?.focus();
  }

  return (
    <form className="search-hero" onSubmit={onSubmit} role="search" onKeyDown={onKeyDown}
      onBlur={(e) => { if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setOpen(false); }}>
      <label htmlFor="q" className="visually-hidden">Buscar nos Diários Oficiais</label>
      <div className="search-hero-field">
        <div className="search-hero-box">
          <SearchIcon size={22} />
          <input
            id="q"
            ref={inputRef}
            type="search"
            value={draft.q}
            onChange={(e) => { setDraft({ ...draft, q: e.target.value }); setOpen(true); }}
            onFocus={() => setOpen(true)}
            onKeyDown={onInputKeyDown}
            placeholder="Nome, CNPJ, empresa ou assunto"
            autoComplete="off"
            aria-describedby="sugestoes-status"
          />
          <button type="submit" className="primary" disabled={loading}>{loading ? "Buscando" : "Buscar"}</button>
        </div>
        <p id="sugestoes-status" className="visually-hidden" aria-live="polite">{showing ? suggestionsStatus(groups) : ""}</p>
        {showing && <SuggestionsPanel id="sugestoes" groups={groups} inputRef={inputRef} />}
      </div>
    </form>
  );
}

interface FieldsProps {
  value: SearchState;
  onChange: (s: SearchState) => void;
}

type FieldVariant = "field" | "pill";

function ScopeField({ label, variant, children }: { label: [string, string]; variant: FieldVariant; children: ReactNode }) {
  const [full, short] = label;
  return (
    <label className={variant === "pill" ? "pill-select" : "field"}>
      <span className={variant === "pill" ? undefined : "field-label"}>{variant === "pill" ? short : full}</span>
      {children}
    </label>
  );
}

export function ScopeSelects({ value, onChange, organs, variant = "field" }: FieldsProps & { organs: Organ[]; variant?: FieldVariant }) {
  const set = (patch: Partial<SearchState>) => onChange({ ...value, ...patch, page: 1 });
  return (
    <>
      <ScopeField label={["Tipo de ato", "Tipo"]} variant={variant}>
        <select value={value.type} onChange={(e) => set({ type: e.target.value as SearchState["type"] })}>
          {TYPE_OPTIONS.map((t) => <option key={t.value || "all"} value={t.value}>{t.label}</option>)}
        </select>
      </ScopeField>
      <ScopeField label={["Diário", "Diário"]} variant={variant}>
        <select value={value.source} onChange={(e) => onChange(withSource(value, e.target.value))}>
          <option value="">Prefeitura e Câmara</option>
          {(Object.keys(SOURCE_LABEL) as Source[]).map((src) => (
            <option key={src} value={src}>{SOURCE_LABEL[src]}</option>
          ))}
        </select>
      </ScopeField>
      {organs.length > 0 && value.source !== "diario_camara" && (
        <ScopeField label={["Órgão", "Órgão"]} variant={variant}>
          <select value={value.organ} onChange={(e) => set({ organ: e.target.value })}>
            <option value="">Todos os órgãos</option>
            {organs.map((o) => (
              <option key={o.acronym} value={o.acronym}>
                {o.name ? `${o.acronym} · ${o.name}` : o.acronym} ({o.acts.toLocaleString("pt-BR")} atos)
              </option>
            ))}
          </select>
        </ScopeField>
      )}
      <ScopeField label={["Tema", "Tema"]} variant={variant}>
        <select value={value.theme} onChange={(e) => set({ theme: e.target.value })}>
          <option value="">Todos os temas</option>
          {Object.entries(THEME_LABEL).map(([slug, label]) => <option key={slug} value={slug}>{label}</option>)}
        </select>
      </ScopeField>
    </>
  );
}

export function FilterPills({ value, onChange, organs, rangeCount, onOpenDialog, onClear }: FieldsProps & {
  organs: Organ[];
  rangeCount: number;
  onOpenDialog: () => void;
  onClear: (() => void) | null;
}) {
  return (
    <div className="filter-pills" role="group" aria-label="Filtros">
      <ScopeSelects value={value} onChange={onChange} organs={organs} variant="pill" />
      <button type="button" className={rangeCount ? "pill pill-active" : "pill"} aria-haspopup="dialog" onClick={onOpenDialog}>
        {rangeCount ? `Período e valor (${rangeCount})` : "Período e valor"}
      </button>
      {onClear && <button type="button" className="quiet pill-clear" onClick={onClear}>Limpar filtros</button>}
    </div>
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

export function FilterBar({ filters, onOpen, onRemove, className = "" }: {
  filters: ActiveFilter[];
  onOpen?: () => void;
  onRemove: (key: FilterKey) => void;
  className?: string;
}) {
  return (
    <div className={`filter-bar ${className}`.trim()}>
      {onOpen && (
        <button type="button" className="pill filters-button" aria-haspopup="dialog" onClick={onOpen}>
          <span className="filters-icon" aria-hidden="true"><span /><span /><span /></span>
          {filters.length ? `Filtros (${filters.length})` : "Filtros"}
        </button>
      )}
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
          <span className="sheet-handle" aria-hidden="true" />
          <h2 id="filters-title">Filtros</h2>
          <button type="button" className="icon-button" aria-label="Fechar filtros" onClick={onClose}>×</button>
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
