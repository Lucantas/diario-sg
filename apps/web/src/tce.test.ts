import { describe, expect, it } from "vitest";
import { condemnationsLabel, diarioSearchHref, stalledPeriod } from "./tce";

describe("tce", () => {
  it("leva o número do processo para a busca", () => {
    expect(diarioSearchHref('"214.824"')).toBe("/?q=%22214.824%22");
  });

  it("conta as condenações", () => {
    expect(condemnationsLabel(1)).toBe("1 condenação");
    expect(condemnationsLabel(3)).toBe("3 condenações");
  });

  it("descreve o período da obra", () => {
    expect(stalledPeriod("2015-07-01", "2016-08-01")).toBe("iniciada em 01/07/2015, paralisada em 01/08/2016");
    expect(stalledPeriod(null, null)).toBe("datas não informadas");
  });
});
