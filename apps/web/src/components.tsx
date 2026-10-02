import { useRef, useState } from "react";
import { ActHit } from "./api";
import { archivedPdfUrl, formatCitation, pageFragment, pageLabel } from "./citation";
import { entityPath } from "./entity";
import { ReportForm } from "./ReportForm";
import { PHASE_LABEL, TYPE_LABEL, formatCents, formatCnpj } from "./types";
import { warningText } from "./warnings";

type Panel = "cite" | "report" | null;

const GAZETTE_NAME = { diario_prefeitura: "Diário da Prefeitura", diario_camara: "Diário da Câmara" } as const;

export function formatLongDate(isoDate: string) {
  return new Date(isoDate + "T12:00:00").toLocaleDateString("pt-BR", { day: "numeric", month: "long", year: "numeric" });
}

export function Result({ hit }: { hit: ActHit }) {
  const [panel, setPanel] = useState<Panel>(null);
  const toggle = (p: Exclude<Panel, null>) => setPanel(panel === p ? null : p);
  const pages = pageLabel(hit);
  const fromCamara = hit.source === "diario_camara";
  return (
    <li className="result">
      <p className="meta">
        <span className="tag">{TYPE_LABEL[hit.type]}</span>
        {hit.phase && PHASE_LABEL[hit.phase] !== TYPE_LABEL[hit.type] && <span className="tag">{PHASE_LABEL[hit.phase]}</span>}
        <span className={fromCamara ? "gazette gazette-camara" : "gazette"}>{GAZETTE_NAME[hit.source]}</span>
        <a href={hit.source_url + pageFragment(hit)} target="_blank" rel="noopener"
          title={fromCamara ? "Abrir o PDF no site da Câmara" : "Abrir o PDF no site da prefeitura"}>
          Edição {hit.edition_number || "s/n"}{hit.is_extra && " (extra)"}, {formatLongDate(hit.published_at)}{pages && `, ${pages}`}
        </a>
        <a href={archivedPdfUrl("", hit)} target="_blank" rel="noopener" title="Abrir a cópia do PDF guardada pelo Diário SG">
          cópia arquivada
        </a>
        {hit.organ && (
          <span className="organ" title={hit.organ_name || undefined}>
            {hit.organ}{hit.organ_name && ` · ${hit.organ_name}`}
          </span>
        )}
      </p>
      <h2>{hit.title}</h2>
      <Warnings hit={hit} />
      {hit.snippet && <p className="snippet"><Highlighted text={hit.snippet} /></p>}
      {hit.values_cents.length > 0 && <Values cents={hit.values_cents} />}
      {hit.cnpjs.length > 0 && (
        <p className="cnpjs">
          <span>Empresas citadas</span>
          {hit.cnpjs.map((c) => (
            <a key={c} className="cnpj" href={`/empresa/${c}`} title="Ver todos os atos desta empresa">{formatCnpj(c)}</a>
          ))}
        </p>
      )}
      {hit.mentions.length > 0 && (
        <p className="mentions">
          <span>Processos e contratos citados</span>
          {hit.mentions.map((m) => (
            <a key={`${m.kind}-${m.slug}`} className="mention" href={entityPath(m.kind, m.slug)}>{m.label}</a>
          ))}
        </p>
      )}
      <p className="actions">
        <button type="button" className="link-button" aria-expanded={panel === "cite"} onClick={() => toggle("cite")}>
          Citar este ato
        </button>
        <button type="button" className="link-button" aria-expanded={panel === "report"} onClick={() => toggle("report")}>
          Reportar erro
        </button>
      </p>
      {panel === "cite" && <Citation hit={hit} />}
      {panel === "report" && <ReportForm hit={hit} />}
    </li>
  );
}

function Warnings({ hit }: { hit: ActHit }) {
  const texts = hit.warnings.map((w) => warningText(w, hit)).filter(Boolean);
  if (texts.length === 0) return null;
  return (
    <ul className="warnings" aria-label="Avisos de extração">
      {texts.map((t) => <li key={t}>{t}</li>)}
    </ul>
  );
}

const SHOWN_VALUES = 3;

function Values({ cents }: { cents: number[] }) {
  const hidden = cents.length - SHOWN_VALUES;
  return (
    <p className="values">
      Valores citados:{" "}
      <span className="values-list">
        {cents.slice(0, SHOWN_VALUES).map(formatCents).join(" · ")}
        {hidden > 0 && ` · +${hidden}`}
      </span>
    </p>
  );
}

function Citation({ hit }: { hit: ActHit }) {
  const text = formatCitation(hit, window.location.origin, new Date());
  const ref = useRef<HTMLParagraphElement>(null);
  const [copied, setCopied] = useState(false);

  function selectText() {
    const node = ref.current;
    const selection = window.getSelection();
    if (!node || !selection) return;
    const range = document.createRange();
    range.selectNodeContents(node);
    selection.removeAllRanges();
    selection.addRange(range);
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      selectText();
    }
  }

  return (
    <div className="cite">
      <p ref={ref}>{text}</p>
      <button type="button" onClick={copy}>{copied ? "Copiado" : "Copiar citação"}</button>
    </div>
  );
}

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
