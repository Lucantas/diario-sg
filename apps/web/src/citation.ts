import { ActHit, Source } from "./api";
import { SOURCE_NAME } from "./types";

const MONTHS = ["jan.", "fev.", "mar.", "abr.", "maio", "jun.", "jul.", "ago.", "set.", "out.", "nov.", "dez."];

type Citable = Pick<ActHit, "gazette_id" | "title" | "edition_number" | "published_at" | "is_extra"
  | "source_url" | "page_start" | "page_end" | "pdf_sha256" | "source">;

function citationAuthor(source: Source) {
  return source === "diario_camara" ? "SÃO GONÇALO (RJ). Câmara Municipal." : "SÃO GONÇALO (RJ).";
}

export function pageFragment(hit: Pick<Citable, "page_start">) {
  return hit.page_start ? `#page=${hit.page_start}` : "";
}

export function pageLabel(hit: Pick<Citable, "page_start" | "page_end">) {
  if (!hit.page_start) return "";
  if (!hit.page_end || hit.page_end === hit.page_start) return `p. ${hit.page_start}`;
  return `p. ${hit.page_start}-${hit.page_end}`;
}

export function archivedPdfUrl(origin: string, hit: Pick<Citable, "gazette_id" | "page_start">) {
  return `${origin}/api/v1/gazettes/${hit.gazette_id}/pdf${pageFragment(hit)}`;
}

function abntDate(d: Date) {
  return `${d.getDate()} ${MONTHS[d.getMonth()]} ${d.getFullYear()}`;
}

function sentence(text: string) {
  return `${text.replace(/[\s.:;,]+$/, "")}.`;
}

export function formatCitation(hit: Citable, origin: string, accessed: Date) {
  const published = new Date(`${hit.published_at}T12:00:00`);
  const edition = `ed. ${hit.edition_number || "s/n"}${hit.is_extra ? " (extra)" : ""}`;
  const where = [edition, abntDate(published), pageLabel(hit)].filter(Boolean).join(", ");
  return [
    `${citationAuthor(hit.source)} ${SOURCE_NAME[hit.source] ?? SOURCE_NAME.diario_prefeitura}, ${where}.`,
    sentence(hit.title),
    `Disponível em: <${hit.source_url}${pageFragment(hit)}>.`,
    `Cópia arquivada em: <${archivedPdfUrl(origin, hit)}>.`,
    `SHA-256 do PDF: ${hit.pdf_sha256}.`,
    `Acesso em: ${abntDate(accessed)}.`,
  ].join(" ");
}
