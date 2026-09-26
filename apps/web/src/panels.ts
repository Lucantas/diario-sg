export interface PanelParams {
  year: number | null;
  organ: string;
}

const MIN_YEAR = 2000;
const MAX_YEAR = 2100;

export function parsePanelParams(search: string): PanelParams {
  const q = new URLSearchParams(search);
  const year = Number(q.get("ano"));
  return {
    year: Number.isInteger(year) && year >= MIN_YEAR && year <= MAX_YEAR ? year : null,
    organ: (q.get("orgao") ?? "").trim().toUpperCase(),
  };
}

function panelQuery(p: PanelParams, yearKey: string, organKey: string) {
  const q = new URLSearchParams();
  if (p.year !== null) q.set(yearKey, String(p.year));
  if (p.organ) q.set(organKey, p.organ);
  const s = q.toString();
  return s ? `?${s}` : "";
}

export function panelHref(p: PanelParams) {
  return `/paineis${panelQuery(p, "ano", "orgao")}`;
}

export function panelApiPath(p: PanelParams) {
  return `/v1/panels/suppliers${panelQuery(p, "year", "organ")}`;
}

const compactBRL = new Intl.NumberFormat("pt-BR", {
  style: "currency", currency: "BRL", notation: "compact", maximumFractionDigits: 1,
});

const exactBRL = new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" });

const COMPACT_FROM_CENTS = 100_000;

export function formatCompactCents(cents: number) {
  return (cents < COMPACT_FROM_CENTS ? exactBRL : compactBRL).format(cents / 100);
}

export function contractsLabel(contracts: number) {
  if (contracts === 0) return "Só aditivos e prorrogações";
  return contracts === 1 ? "1 contratação" : `${contracts} contratações`;
}
