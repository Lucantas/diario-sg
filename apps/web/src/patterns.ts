import { Pattern, PatternSearch } from "./api";

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

export function highlights(patterns: Pattern[], limit: number): PatternHighlight[] {
  return patterns
    .filter((p) => p.findings.length > 0)
    .slice(0, limit)
    .map((p) => ({
      pattern: p.title,
      title: p.findings[0].title,
      detail: p.findings[0].detail,
      count: p.findings.length === 1 ? "1 caso na base" : `${p.findings.length.toLocaleString("pt-BR")} casos na base`,
      href: patternHref(p.id),
    }));
}
