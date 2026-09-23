import { useRef, useState } from "react";
import { ActHit } from "./api";
import { archivedPdfUrl, formatCitation, pageFragment, pageLabel } from "./citation";
import { TYPE_LABEL, formatCents, formatCnpj } from "./types";

export function Result({ hit }: { hit: ActHit }) {
  const [citing, setCiting] = useState(false);
  const date = new Date(hit.published_at + "T12:00:00").toLocaleDateString("pt-BR");
  const pages = pageLabel(hit);
  return (
    <li className="result">
      <p className="meta">
        <span className={`tag tag-${hit.type}`}>{TYPE_LABEL[hit.type]}</span>
        <a href={hit.source_url + pageFragment(hit)} target="_blank" rel="noopener" title="Abrir o PDF no site da prefeitura">
          Edição {hit.edition_number || "s/n"}{hit.is_extra && " (extra)"}, {date}{pages && `, ${pages}`}
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
      <p className="snippet"><Highlighted text={hit.snippet} /></p>
      {hit.values_cents.length > 0 && <Values cents={hit.values_cents} />}
      {hit.cnpjs.length > 0 && (
        <p className="cnpjs">
          Empresas citadas:{" "}
          {hit.cnpjs.map((c) => (
            <a key={c} className="cnpj" href={`/empresa/${c}`} title="Ver todos os atos desta empresa">{formatCnpj(c)}</a>
          ))}
        </p>
      )}
      <button type="button" className="link-button" aria-expanded={citing} onClick={() => setCiting(!citing)}>
        Citar este ato
      </button>
      {citing && <Citation hit={hit} />}
    </li>
  );
}

const SHOWN_VALUES = 3;

function Values({ cents }: { cents: number[] }) {
  const hidden = cents.length - SHOWN_VALUES;
  return (
    <p className="values">
      Valores citados: {cents.slice(0, SHOWN_VALUES).map(formatCents).join(" · ")}
      {hidden > 0 && ` · +${hidden}`}
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
