import { describe, expect, it } from "vitest";
import { ActHit } from "./api";
import { entityPath, groupByOrgan, parseEntityPath } from "./entity";

const hit = (id: string, organ: string, published_at: string) => ({ id, organ, published_at }) as ActHit;

describe("entityPath e parseEntityPath", () => {
  it("vão e voltam pelo slug", () => {
    expect(entityPath("contrato", "30-FMS-2011")).toBe("/contrato/30-FMS-2011");
    expect(parseEntityPath("/contrato/30-FMS-2011")).toEqual({ kind: "contrato", slug: "30-FMS-2011" });
    expect(parseEntityPath("/processo/14.672-2021")).toEqual({ kind: "processo", slug: "14.672-2021" });
    expect(parseEntityPath("/empresa/123")).toBeNull();
  });
});

describe("groupByOrgan", () => {
  const organs = [{ organ: "SEMAD", organ_name: "", acts: 2 }, { organ: "SEMTRAN", organ_name: "", acts: 1 }, { organ: "", organ_name: "", acts: 1 }];
  const acts = [hit("c", "SEMAD", "2024-03-27"), hit("d", "", "2024-05-01"), hit("b", "SEMTRAN", "2024-02-02"), hit("a", "SEMAD", "2023-02-27")];

  it("agrupa na ordem dos órgãos, sem órgão por último, do mais antigo ao mais recente", () => {
    expect(groupByOrgan(acts, organs, "").map((g) => [g.organ, g.acts.map((a) => a.id)])).toEqual([
      ["SEMAD", ["a", "c"]], ["SEMTRAN", ["b"]], ["", ["d"]],
    ]);
  });

  it("filtra pelo órgão escolhido", () => {
    expect(groupByOrgan(acts, organs, "SEMTRAN").map((g) => g.organ)).toEqual(["SEMTRAN"]);
  });
});
