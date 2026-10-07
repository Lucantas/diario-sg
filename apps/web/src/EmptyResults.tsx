import { useEffect, useState } from "react";
import { searchActs } from "./api";
import { Relaxation, ResultsHeading } from "./results";
import { SearchState, apiParams, queryFromState } from "./searchState";

interface Counted extends Relaxation {
  total: number;
}

export function EmptyResults({ heading, relaxed, onGo }: {
  heading: ResultsHeading;
  relaxed: Relaxation[];
  onGo: (s: SearchState) => void;
}) {
  const [counted, setCounted] = useState<Counted[]>([]);
  const key = relaxed.map((r) => queryFromState(r.state)).join("|");

  useEffect(() => {
    const ctrl = new AbortController();
    setCounted([]);
    Promise.all(relaxed.map(async (r) => {
      const params = apiParams(r.state, 1);
      if (!params) return { ...r, total: 0 };
      const res = await searchActs(params, ctrl.signal).catch(() => ({ total: 0 }));
      return { ...r, total: res.total };
    })).then((all) => { if (!ctrl.signal.aborted) setCounted(all.filter((r) => r.total > 0)); });
    return () => ctrl.abort();
  }, [key]);

  return (
    <div className="card empty-state">
      <h2>0 atos encontrados <span className="results-heading-query">{heading.scope}</span></h2>
      {counted.length > 0 && (
        <ul className="relaxations">
          {counted.map((r) => (
            <li key={r.label}>
              <a href={"/" + queryFromState(r.state)} onClick={(e) => { e.preventDefault(); onGo(r.state); }}>
                <span>{r.label}</span>
                <strong>{r.total.toLocaleString("pt-BR")} {r.total === 1 ? "ato" : "atos"} →</strong>
              </a>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
