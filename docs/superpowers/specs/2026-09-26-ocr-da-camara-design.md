# OCR das edições escaneadas

## Problema

78 edições do Diário da Câmara (251 páginas, 8% das edições) são imagem:
o `pdftotext` só lê o cabeçalho que o site sobrepõe, e a edição fica sem
atos e fora da busca. O Diário da Prefeitura não tem edição assim na base
local.

O que tem nelas, conferido por amostra: decretos legislativos de
suplementação orçamentária, movimentação financeira do fundo especial,
relatórios de gestão fiscal (tabelas). Com o Tesseract em português a
300 dpi, o texto corrido sai quase limpo; as tabelas saem com números
trocados.

## Desenho

- **Quando.** Página com menos de 400 letras no `pdftotext` **e** com uma
  imagem de pelo menos 400 × 400 pixels (`pdfimages -list`) vai para OCR.
  Página de texto curto sem imagem grande (fim de edição, página em
  branco) não vai.
- **Como.** `pdftoppm -r 300 -gray` da página, `tesseract -l por --psm 1`
  (com detecção de orientação, porque há tabela deitada). O texto do OCR
  substitui o da página, inclusive no modo com colunas da Câmara.
- **Limite.** O worker (Cloud Run, 300 s) lê no máximo 25 páginas por OCR
  por edição, em ordem; o resto fica sem texto, como hoje. O `reindex`
  não tem limite.
- **Proveniência.** `gazettes.ocr_pages integer[]` (migration 022) guarda
  as páginas lidas por OCR. O ato cujas páginas caem nelas sai com
  `ocr: true` na API (`/v1/acts/{id}`) e `lido_por_ocr` no MCP
  (`ler_ato`, `pagina_original`); a página do ato avisa que o texto veio
  de imagem e pode ter erro, com o link da página original.
- **Erro.** Sem o `tesseract` ou sem o português instalado, a extração
  falha com a mensagem do Tesseract (não segue calada sem o texto).
- **Imagem.** O Dockerfile instala `tesseract-ocr`,
  `tesseract-ocr-data-por` e `tesseract-ocr-data-osd` (Alpine 3.20,
  Tesseract 5.3.4, conferido).

## Fora

- OCR de tabela com estrutura (colunas e valores): o texto serve para a
  busca, não para somar.
- Corrigir palavra do OCR por dicionário.
