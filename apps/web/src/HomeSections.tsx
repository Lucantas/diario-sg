import { useEffect, useState } from "react";
import { ActHit, LatestEdition, listLatestEditions, listPatterns, searchActs } from "./api";
import { Result } from "./components";
import { actsLabel, editionHref, editionTitle, homeEdition, relativeDay, typeCountLabel } from "./latest";
import { PatternHighlight, highlights } from "./patterns";

const EDITION_CARDS = 4;
const HIGHLIGHTS = 2;

export function LatestEditionSection() {
  const [edition, setEdition] = useState<LatestEdition | null>(null);
  const [acts, setActs] = useState<ActHit[]>([]);

  useEffect(() => {
    let cancelled = false;
    listLatestEditions()
      .then(async (res) => {
        const e = homeEdition(res.items);
        if (cancelled || !e) return;
        setEdition(e);
        const page = await searchActs(new URLSearchParams({ q: "", gazette: e.gazette_id, limit: String(EDITION_CARDS) }));
        if (!cancelled) setActs(page.items);
      })
      .catch(() => { if (!cancelled) setEdition(null); });
    return () => { cancelled = true; };
  }, []);

  if (!edition) return null;
  return (
    <section className="home-section" aria-labelledby="edicao-titulo">
      <div className="home-section-head">
        <h2 id="edicao-titulo">
          {editionTitle(edition)} <span className="home-section-when">· {relativeDay(edition.published_at, new Date())}</span>
        </h2>
        <a href={editionHref(edition.gazette_id)}>{actsLabel(edition.total_acts)} →</a>
      </div>
      <ul className="edition-counts">
        {edition.types.map((t) => (
          <li key={t.type}>
            <a className="pill" href={editionHref(edition.gazette_id, t.type)}>{typeCountLabel(t.type, t.acts, t.value_cents)}</a>
          </li>
        ))}
      </ul>
      {acts.length > 0 && (
        <ol className="act-grid">
          {acts.map((h) => <Result key={h.id} hit={h} />)}
        </ol>
      )}
    </section>
  );
}

export function VerifySection() {
  const [items, setItems] = useState<PatternHighlight[]>([]);

  useEffect(() => {
    let cancelled = false;
    listPatterns()
      .then((res) => { if (!cancelled) setItems(highlights(res.items, HIGHLIGHTS)); })
      .catch(() => { if (!cancelled) setItems([]); });
    return () => { cancelled = true; };
  }, []);

  if (items.length === 0) return null;
  return (
    <section className="home-section" aria-labelledby="verificar-titulo">
      <div className="home-section-head">
        <h2 id="verificar-titulo">Para verificar</h2>
        <a href="/padroes">Todos os padrões →</a>
      </div>
      <div className="verify-grid">
        {items.map((v) => (
          <a key={v.href} className="verify-card" href={v.href}>
            <span className="verify-pattern">{v.pattern}</span>
            <strong>{v.title}</strong>
            <span className="verify-detail">{v.detail}</span>
            <span className="verify-count">{v.count} →</span>
          </a>
        ))}
      </div>
      <p className="fineprint">
        Padrões para verificar, não irregularidades: cada caso só diz que os atos seguem a regra descrita. Confira sempre a
        edição original.
      </p>
    </section>
  );
}
