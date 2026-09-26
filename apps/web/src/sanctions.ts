import { Sanction, SanctionRegister } from "./api";
import { formatIsoDate } from "./registry";

export function sanctionStateLabel(s: Sanction): string {
  switch (s.state) {
    case "fora_do_cadastro":
      return `Saiu do cadastro depois de ${formatIsoDate(s.last_seen)}`;
    case "prazo_encerrado":
      return `Prazo encerrado em ${formatIsoDate(s.ends_at)}`;
    default:
      return s.ends_at ? `No cadastro, até ${formatIsoDate(s.ends_at)}` : "No cadastro, sem data final";
  }
}

export function sanctionPeriod(s: Sanction): string {
  const start = formatIsoDate(s.starts_at);
  const end = formatIsoDate(s.ends_at);
  if (start && end) return `${start} a ${end}`;
  if (start) return `desde ${start}`;
  return end ? `até ${end}` : "sem período informado";
}

export function sanctionOrgan(s: Sanction): string {
  const uf = s.organ.includes(`(${s.organ_uf})`) ? "" : s.organ_uf;
  const where = [uf, s.sphere.toLowerCase()].filter(Boolean).join(", ");
  return where ? `${s.organ} (${where})` : s.organ;
}

export function listedOnLabel(listedOn: Partial<Record<SanctionRegister, string>>): string {
  const days = [...new Set(Object.values(listedOn).filter((d): d is string => Boolean(d)))].sort();
  return days.map(formatIsoDate).join(" e ");
}

export function sanctionsSummary(sanctions: Sanction[]): string {
  const listed = sanctions.filter((s) => s.state === "no_cadastro").length;
  const total = sanctions.length === 1 ? "1 sanção" : `${sanctions.length} sanções`;
  if (listed === sanctions.length) return `${total} no cadastro. Clique em cada uma para ver o processo e a fundamentação.`;
  return `${total}, ${listed} no cadastro. Clique em cada uma para ver o processo e a fundamentação.`;
}

export function sanctionTerms(register: SanctionRegister): { organ: string; basis: string } {
  return register === "CEPIM" ? { organ: "Órgão concedente", basis: "Motivo" } : { organ: "Órgão sancionador", basis: "Fundamentação" };
}
