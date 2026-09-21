import { ActHit } from "./api";
import { TYPE_LABEL, formatCnpj } from "./types";

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
            <a key={c} className="cnpj" href={`/empresa/${c}`} title="Ver todos os atos desta empresa">{formatCnpj(c)}</a>
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

