import { describe, expect, it } from "vitest";
import { condemnationsLabel, coverageLabel, diarioSearchHref, rreoPeriodLabel, stalledPeriod } from "./tce";

describe("tce", () => {
  it("leva o número do processo para a busca", () => {
    expect(diarioSearchHref('"214.824" TCE')).toBe("/?q=%22214.824%22+TCE");
  });

  it("conta as condenações", () => {
    expect(condemnationsLabel(1)).toBe("1 condenação");
    expect(condemnationsLabel(3)).toBe("3 condenações");
  });

  it("descreve o período da obra", () => {
    expect(stalledPeriod("2015-07-01", "2016-08-01")).toBe("iniciada em 01/07/2015, paralisada em 01/08/2016");
    expect(stalledPeriod(null, null)).toBe("datas não informadas");
  });

  it("diz até onde vai o RREO e quanto o TCE cobre", () => {
    expect(rreoPeriodLabel(6)).toBe("ano fechado");
    expect(rreoPeriodLabel(3)).toBe("até o 3º bimestre");
    expect(coverageLabel(8712)).toBe("87,1%");
    expect(coverageLabel(10000)).toBe("100%");
  });
});
