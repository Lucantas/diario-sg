import { describe, expect, it } from "vitest";
import { warningText } from "./warnings";

const pages = { page_start: 4, page_end: 15 };

describe("warningText", () => {
  it("explica cada aviso de extração", () => {
    expect(warningText("sem_numero", pages)).toBe(
      "O título deste ato não tem número. A extração pode ter juntado ou cortado atos; confira no PDF.",
    );
    expect(warningText("so_titulo", pages)).toBe(
      "Só o título deste ato foi extraído. O texto dele está no PDF.",
    );
  });

  it("diz quantas páginas o ato ocupa", () => {
    expect(warningText("muitas_paginas", pages)).toBe(
      "Este ato ocupa 12 páginas. Ele pode ter engolido atos vizinhos; confira no PDF.",
    );
  });

  it("ignora avisos que o site ainda não conhece", () => {
    expect(warningText("novo_aviso", pages)).toBe("");
    expect(warningText("toString", pages)).toBe("");
  });
});
