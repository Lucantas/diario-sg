import { PatternSearch } from "./api";

export function patternSearchHref(s: PatternSearch) {
  return `/?${new URLSearchParams({ tipo: s.type, de: s.from, ate: s.to, fonte: s.source })}`;
}
