import { describe, expect, it } from "vitest";
import { donationPix } from "./donation";

describe("donationPix", () => {
  it("não oferece PIX enquanto a chave não foi configurada", () => {
    expect(donationPix("")).toBeNull();
    expect(donationPix("   ")).toBeNull();
  });

  it("monta o copia e cola com o recebedor do projeto", () => {
    const pix = donationPix(" 8f1c2d3e-0000-4000-8000-000000000000 ");

    expect(pix?.key).toBe("8f1c2d3e-0000-4000-8000-000000000000");
    expect(pix?.payload).toContain("0136" + "8f1c2d3e-0000-4000-8000-000000000000");
    expect(pix?.payload).toContain("5924Lucas Dantas de Oliveira");
    expect(pix?.payload).toContain("6011Sao Goncalo");
  });
});
