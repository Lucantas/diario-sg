import { PatternHighlightResponse, PatternSearch } from "./api";

export function patternSearchHref(s: PatternSearch) {
  return `/?${new URLSearchParams({ tipo: s.type, de: s.from, ate: s.to, fonte: s.source })}`;
}

export interface PatternHighlight {
  pattern: string;
  title: string;
  detail: string;
  count: string;
  href: string;
}

export function patternHref(id: string) {
  return `/padroes#padrao-${id}`;
}

export function highlights(items: PatternHighlightResponse[]): PatternHighlight[] {
  return items.map((h) => ({
    pattern: h.pattern_title,
    title: h.title,
    detail: h.detail,
    count: h.findings === 1 ? "1 caso na base" : `${h.findings.toLocaleString("pt-BR")} casos na base`,
    href: patternHref(h.pattern_id),
  }));
}
