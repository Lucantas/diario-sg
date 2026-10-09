import { describe, expect, it } from "vitest";
import { goalProgress } from "./goal";

const october = { month: "2026-10", goal_cents: 12000, received_cents: 8550, updated_on: "2026-10-09" };

describe("goalProgress", () => {
  it("descreve o quanto entrou no mês em relação à meta", () => {
    const progress = goalProgress(october);

    expect(progress).toEqual({
      percent: 71,
      received: "R$ 85,50",
      goal: "R$ 120",
      month: "outubro de 2026",
      updatedOn: "9 de outubro",
      reached: false,
    });
  });

  it("começa o mês vazio", () => {
    const progress = goalProgress({ ...october, received_cents: 0 });

    expect(progress.percent).toBe(0);
    expect(progress.received).toBe("R$ 0");
  });

  it("enche a barra e marca a meta batida quando passa dela", () => {
    const progress = goalProgress({ ...october, received_cents: 15000 });

    expect(progress.percent).toBe(100);
    expect(progress.received).toBe("R$ 150");
    expect(progress.reached).toBe(true);
  });
});
