import { PNCPContract } from "./api";
import { formatIsoDate } from "./registry";

export function pncpTerm(c: PNCPContract): string {
  const start = formatIsoDate(c.starts_at);
  const end = formatIsoDate(c.ends_at);
  if (start && end) return `${start} a ${end}`;
  if (start) return `desde ${start}`;
  return end ? `até ${end}` : "vigência não informada";
}

export function pncpHeading(c: PNCPContract): string {
  const number = c.number ? `${c.kind} ${c.number}` : c.kind;
  return c.process ? `${number}, processo ${c.process}` : number;
}
