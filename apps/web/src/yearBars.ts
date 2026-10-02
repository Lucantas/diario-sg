export interface YearCount {
  year: number;
  count: number;
}

export function countByYear(isoDates: string[]): YearCount[] {
  if (isoDates.length === 0) return [];
  const counts = new Map<number, number>();
  for (const date of isoDates) {
    const year = Number(date.slice(0, 4));
    counts.set(year, (counts.get(year) ?? 0) + 1);
  }
  const years = [...counts.keys()];
  const first = Math.min(...years);
  const last = Math.max(...years);
  return Array.from({ length: last - first + 1 }, (_, i) => ({ year: first + i, count: counts.get(first + i) ?? 0 }));
}
