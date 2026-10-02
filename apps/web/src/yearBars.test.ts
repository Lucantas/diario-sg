import { describe, expect, it } from "vitest";
import { countByYear } from "./yearBars";

describe("countByYear", () => {
  it("conta os atos por ano e preenche com zero os anos sem ato", () => {
    const dates = ["2021-03-01", "2023-05-10", "2023-12-31", "2021-01-02"];

    expect(countByYear(dates)).toEqual([
      { year: 2021, count: 2 },
      { year: 2022, count: 0 },
      { year: 2023, count: 2 },
    ]);
  });

  it("devolve lista vazia sem atos", () => {
    expect(countByYear([])).toEqual([]);
  });
});
