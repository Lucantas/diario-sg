import { MunicipalCommitment } from "./api";
import { formatIsoDate } from "./registry";

const OBJECT_PREVIEW = 240;

export function commitmentHeading(c: MunicipalCommitment): string {
  return `Empenho ${c.number}/${c.year} · ${formatIsoDate(c.date)} · ${c.entity}`;
}

export function commitmentProcess(c: MunicipalCommitment): string {
  const kind = c.process_kind && c.process_kind !== "Outros/Não aplicável" ? c.process_kind.toLowerCase() : "";
  const process = c.process ? `processo ${c.process}` : "";
  const modality = c.modality && !c.modality.startsWith("Outros") ? c.modality : "";
  return [process, kind, modality].filter(Boolean).join(" · ");
}

export function objectPreview(text: string): string {
  return text.length <= OBJECT_PREVIEW ? text : `${text.slice(0, OBJECT_PREVIEW).trimEnd()}…`;
}
