import { KeyboardEvent, RefObject, useEffect, useState } from "react";
import { Bill, Organ, Suggestion, listBills, suggest } from "./api";
import { SuggestGroup, moveFocus, suggestGroups } from "./discovery";

const REMOTE_MIN = 3;
const DEBOUNCE_MS = 250;
const BILLS_LIMIT = 3;

export function useSuggestions(q: string, organs: Organ[]): SuggestGroup[] {
  const [remote, setRemote] = useState<{ q: string; items: Suggestion[]; bills: Bill[] }>({ q: "", items: [], bills: [] });
  const term = q.trim();

  useEffect(() => {
    if (term.length < REMOTE_MIN) return;
    const ctrl = new AbortController();
    const timer = window.setTimeout(() => {
      Promise.all([
        suggest(term, ctrl.signal).then((r) => r.items).catch(() => []),
        listBills(new URLSearchParams({ q: term, limit: String(BILLS_LIMIT) }), ctrl.signal).then((r) => r.items).catch(() => []),
      ]).then(([items, bills]) => { if (!ctrl.signal.aborted) setRemote({ q: term, items, bills }); });
    }, DEBOUNCE_MS);
    return () => { window.clearTimeout(timer); ctrl.abort(); };
  }, [term]);

  const fresh = remote.q === term;
  return suggestGroups(term, fresh ? remote.items : [], organs, fresh ? remote.bills : []);
}

export function SuggestionsPanel({ id, groups, inputRef }: {
  id: string;
  groups: SuggestGroup[];
  inputRef: RefObject<HTMLInputElement>;
}) {
  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    e.preventDefault();
    const links = [...e.currentTarget.querySelectorAll("a")];
    const next = moveFocus(links.indexOf(document.activeElement as HTMLAnchorElement), e.key === "ArrowDown" ? 1 : -1, links.length);
    (next < 0 ? inputRef.current : links[next])?.focus();
  }

  return (
    <div id={id} className="suggestions" onMouseDown={(e) => e.preventDefault()} onKeyDown={onKeyDown}>
      {groups.map((g) => (
        <div key={g.title} className="suggestions-group">
          <p className="group-title">{g.title}</p>
          <ul aria-label={g.title}>
            {g.items.map((it) => (
              <li key={it.href}>
                <a href={it.href}>
                  <span className="suggestion-label">{it.label}</span>
                  {it.meta && <span className="suggestion-meta">{it.meta}</span>}
                </a>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
