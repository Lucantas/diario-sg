const MONTHS = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];

export function formatYearMonth(ym: string): string {
  const [y, m] = ym.split("-");
  return `${MONTHS[Number(m) - 1]}/${y}`;
}

export function coverageLabel(c: { from: string; to: string }): string {
  return `${formatYearMonth(c.from)} a ${formatYearMonth(c.to)}`;
}

