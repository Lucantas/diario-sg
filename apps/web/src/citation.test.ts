import { describe, expect, it } from "vitest";
import { ActHit } from "./api";
import { archivedPdfUrl, formatCitation, pageFragment, pageLabel } from "./citation";

const hit: ActHit = {
  id: "a1", gazette_id: "g1", type: "portaria", title: "PORTARIA Nº 10/2026", organ: "SEMAD",
  organ_name: "", snippet: "", edition_number: "1771", published_at: "2026-09-18", is_extra: false,
  source_url: "https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf", cnpjs: [], values_cents: [],
  page_start: 3, page_end: 4, pdf_sha256: "ab".repeat(32),
};
const noPage = { ...hit, page_start: null, page_end: null };

describe("pageLabel", () => {
  it("mostra uma página ou o intervalo", () => {
    expect(pageLabel({ ...hit, page_end: 3 })).toBe("p. 3");
    expect(pageLabel(hit)).toBe("p. 3-4");
  });

  it("fica vazio sem página", () => {
    expect(pageLabel(noPage)).toBe("");
  });
});

describe("links", () => {
  it("apontam a página do ato quando ela é conhecida", () => {
    expect(pageFragment(hit)).toBe("#page=3");
    expect(archivedPdfUrl("https://diario.exemplo", hit)).toBe("https://diario.exemplo/api/v1/gazettes/g1/pdf#page=3");
  });

  it("abrem o PDF no início sem página", () => {
    expect(pageFragment(noPage)).toBe("");
    expect(archivedPdfUrl("https://diario.exemplo", noPage)).toBe("https://diario.exemplo/api/v1/gazettes/g1/pdf");
  });
});

describe("formatCitation", () => {
  const accessed = new Date(2026, 8, 22);

  it("cita edição, data, página, links e hash", () => {
    expect(formatCitation(hit, "https://diario.exemplo", accessed)).toBe(
      "SÃO GONÇALO (RJ). Diário Oficial do Município de São Gonçalo, ed. 1771, 18 set. 2026, p. 3-4. " +
      "PORTARIA Nº 10/2026. Disponível em: <https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf#page=3>. " +
      "Cópia arquivada em: <https://diario.exemplo/api/v1/gazettes/g1/pdf#page=3>. " +
      `SHA-256 do PDF: ${"ab".repeat(32)}. Acesso em: 22 set. 2026.`,
    );
  });

  it("marca edição extra, sem número e sem página", () => {
    const text = formatCitation({ ...noPage, edition_number: "", is_extra: true }, "https://diario.exemplo", accessed);

    expect(text).toContain("ed. s/n (extra), 18 set. 2026. PORTARIA");
    expect(text).toContain("<https://do.pmsg.rj.gov.br/diario/2026_09_18.pdf>");
    expect(text).not.toContain("#page");
  });

  it("não duplica o ponto de título que já termina em ponto", () => {
    const text = formatCitation({ ...hit, title: "Nomeia:" }, "https://diario.exemplo", accessed);

    expect(text).toContain("p. 3-4. Nomeia. Disponível");
  });
});
