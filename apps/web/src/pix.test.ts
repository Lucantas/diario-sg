import { describe, expect, it } from "vitest";
import { pixPayload } from "./pix";

describe("pixPayload", () => {
  it("monta o copia e cola estático do exemplo do manual do BR Code", () => {
    const payload = pixPayload({ key: "123e4567-e12b-12d1-a456-426655440000", name: "Fulano de Tal", city: "BRASILIA" });

    expect(payload).toBe(
      "00020126580014br.gov.bcb.pix0136123e4567-e12b-12d1-a456-4266554400005204000053039865802BR5913Fulano de Tal6008BRASILIA62070503***63041D3D",
    );
  });

  it("tira acentos e corta nome e cidade nos limites do padrão", () => {
    const payload = pixPayload({ key: "chave", name: "Lucas Dantas de Oliveira Souza Lima", city: "São Gonçalo do Rio Abaixo" });

    expect(payload).toContain("5925Lucas Dantas de Oliveira ");
    expect(payload).toContain("6015Sao Goncalo do ");
  });
});
