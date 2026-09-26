# OCR das edições escaneadas — plano

**Spec:** `docs/superpowers/specs/2026-09-26-ocr-da-camara-design.md`

1. Domínio e portas: `domain.ExtractedText{Text, OCRPages}`,
   `TextExtractor.Extract` e `PageExtractor.ExtractPage` devolvem o
   texto com as páginas de OCR; `domain.Gazette.OCRPages`;
   `ActReadByOCR(pageStart, pageEnd, ocrPages)`. Testes de domínio.
2. Adaptador `pdf`: páginas candidatas (`pdfimages -list`), OCR por
   página, limite `MaxOCRPages`, troca da página no texto cru e no
   `-layout`. Testes com PDF de imagem gerado no teste (pula sem
   Tesseract).
3. Casos de uso `index_gazette`, `reindex_gazettes`, `read_page` levam as
   páginas de OCR até o repositório; migration 022; repositório grava e
   lê `ocr_pages`. Teste de integração.
4. API, MCP e web: `ocr` no ato, `lido_por_ocr` no MCP, aviso na página
   do ato. Dockerfile com o Tesseract.
5. `make reindex` das edições da Câmara sem atos, contagem antes e
   depois, docs (roadmap, README, parser-findings) e "Depois da entrega".
