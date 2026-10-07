import { useEffect, useState } from "react";
import { Organ, getCompany, getEntity, suggest } from "./api";
import { Ficha, companyFicha, entityFicha, exactMatch, looksLikeNumber, matchOrgan, organFicha } from "./discovery";

const MIN_QUERY = 3;

export function useFicha(q: string, organs: Organ[]): Ficha | null {
  const term = q.trim();
  const organ = matchOrgan(term, organs);
  const [remote, setRemote] = useState<{ q: string; ficha: Ficha | null }>({ q: "", ficha: null });

  useEffect(() => {
    if (organ || term.length < MIN_QUERY || !looksLikeNumber(term)) return;
    const ctrl = new AbortController();
    fichaFor(term, ctrl.signal)
      .then((ficha) => { if (!ctrl.signal.aborted) setRemote({ q: term, ficha }); })
      .catch(() => { if (!ctrl.signal.aborted) setRemote({ q: term, ficha: null }); });
    return () => ctrl.abort();
  }, [term, organ]);

  if (organ) return organFicha(organ);
  return remote.q === term ? remote.ficha : null;
}

async function fichaFor(term: string, signal: AbortSignal): Promise<Ficha | null> {
  const { items } = await suggest(term, signal);
  const match = exactMatch(term, items);
  if (!match) return null;
  if (match.kind === "cnpj") return companyFicha(match, await getCompany(match.key).catch(() => null));
  const entity = await getEntity(match.kind, match.label.replace(/\//g, "-")).catch(() => null);
  return entityFicha(match, entity);
}

export function FichaCard({ ficha }: { ficha: Ficha }) {
  return (
    <a className="ficha" href={ficha.href}>
      <span className="ficha-kind">{ficha.kind}</span>
      <strong className="ficha-title">{ficha.title}</strong>
      <span className="ficha-facts">{ficha.facts.map((f) => <span key={f}>{f}</span>)}</span>
      <span className="ficha-cta">{ficha.cta}</span>
    </a>
  );
}
