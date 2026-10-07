import { ActHit } from "./api";
import { editionTitle, longDate, typeCountLabel } from "./latest";
import { SearchState, activeFilters, withoutFilter } from "./searchState";

export interface ResultsHeading {
  count: string;
  scope: string;
}

const PHRASE = /^"([^"]+)"$/;

const MIN_ANY_WORD = 3;

export function editionLabel(first: ActHit | undefined): string {
  return first ? editionTitle(first) : "Uma edição";
}

export function resultsHeading(s: SearchState, total: number, first: ActHit | undefined): ResultsHeading {
  const count = s.type ? typeCountLabel(s.type, total, 0) : `${total.toLocaleString("pt-BR")} ${total === 1 ? "ato" : "atos"}`;
  if (s.edition) {
    const where = first ? `na ${editionTitle(first).toLowerCase()}, de ${longDate(first.published_at)}` : "nesta edição";
    return { count, scope: s.q ? `para “${s.q}” ${where}` : where };
  }
  if (!s.q) return { count, scope: "em todas as edições" };
  const found = `${total.toLocaleString("pt-BR")} ${total === 1 ? "ato encontrado" : "atos encontrados"}`;
  return { count: found, scope: isExactPhrase(s.q) ? `para “${unquoted(s.q)}”, frase exata` : `para “${s.q}”` };
}

export function isExactPhrase(q: string): boolean {
  return PHRASE.test(q.trim());
}

export function unquoted(q: string): string {
  return q.trim().replace(PHRASE, "$1");
}

function plainWords(q: string): string[] | null {
  const text = unquoted(q);
  if (/["]|(^|\s)-|\bOU\b/.test(text)) return null;
  return text.split(/\s+/).filter(Boolean);
}

export function canToggleExact(q: string): boolean {
  const words = plainWords(q);
  return words !== null && words.length >= 2;
}

export function toggleExact(q: string): string {
  return isExactPhrase(q) ? unquoted(q) : `"${q.trim()}"`;
}

export interface Relaxation {
  label: string;
  state: SearchState;
}

export function relaxations(s: SearchState, editionName: string): Relaxation[] {
  const out: Relaxation[] = activeFilters(s, editionName).map((f) => ({
    label: `Sem o filtro ${f.label}`,
    state: withoutFilter(s, f.key),
  }));
  if (isExactPhrase(s.q)) {
    out.unshift({ label: "Sem exigir a frase exata", state: { ...s, q: unquoted(s.q), page: 1 } });
  }
  const words = (plainWords(s.q) ?? []).filter((w) => w.length >= MIN_ANY_WORD);
  if (words.length >= 2) {
    const any = words.join(" OU ");
    out.push({ label: `Com qualquer uma das palavras: ${any}`, state: { ...s, q: any, page: 1 } });
  }
  return out;
}
