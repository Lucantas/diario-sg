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

- **Quando.** Página com menos de 200 letras no `pdftotext` **e** com
  imagens cobrindo pelo menos 25% da área (tamanho e resolução do
  `pdfimages -list` contra o tamanho da página no `pdfinfo`) vai para
  OCR. Nas 246 páginas escaneadas da Câmara, o texto nativo tem no máximo
  123 letras (o cabeçalho sobreposto) e a imagem cobre de 27% a 100%. Os
  anexos de foto da Prefeitura (termo de apreensão de animal) têm cerca de
  320 letras e a foto cobre 10% a 13%: ficam de fora pelas duas regras.
- **Como.** `pdftoppm -r 300 -gray` da página, `tesseract -l por --psm 1`
  (com detecção de orientação, porque há tabela deitada). O texto do OCR
  substitui o da página, inclusive no modo com colunas da Câmara.
- **Limite.** O worker (Cloud Run, 300 s, 1 vCPU) para de ler por OCR
  depois de 90 s na edição; as páginas que sobram ficam sem texto, como
  hoje, até o próximo `reindex`, que não tem limite. O Tesseract roda com
  `OMP_THREAD_LIMIT=1`: com várias threads num núcleo só ele fica 4 vezes
  mais lento.
- **Proveniência.** O indexador marca `acts.read_by_ocr` (migration 022)
  no ato cujas páginas foram lidas por OCR, e o ato ganha o aviso
  `lido_por_ocr`, como os outros avisos de extração (busca, página do ato,
  exportação e MCP). A ferramenta `pagina_original` do MCP diz
  `lida_por_ocr` na página.
- **Erro.** Sem o `tesseract` ou sem o português instalado, a extração
  falha com a mensagem do Tesseract (não segue calada sem o texto).
- **Imagem.** O Dockerfile instala `tesseract-ocr`,
  `tesseract-ocr-data-por` e `tesseract-ocr-data-osd` (Alpine 3.20,
  Tesseract 5.3.4, conferido).

## Fora

- OCR de tabela com estrutura (colunas e valores): o texto serve para a
  busca, não para somar.
- Corrigir palavra do OCR por dicionário.

## Depois da entrega

- A proveniência ficou no ato (`acts.read_by_ocr`), não na edição
  (`gazettes.ocr_pages`, como o plano previa): o aviso é por ato, e a
  página do ato já diz quais páginas ler na edição original.
- A primeira regra (imagem de pelo menos 400×400 px) mandava para o OCR as
  páginas de fotos da Prefeitura; a regra de letras e cobertura acima a
  substituiu.
- `reindex` local de 2010 a 2026, 5.286 edições em 6 h 57 min, nenhuma
  falha: 854 atos lidos por OCR em 739 edições, 227 da Câmara (163
  edições, de 10/2020 a 09/2026) e 627 da Prefeitura (576 edições, de
  09/2011 a 07/2026). A Câmara passou de 4.853 para 5.018 atos e a
  Prefeitura de 157.837 para 157.937; nenhum tipo nem órgão perdeu ato.
  Sobram sem atos só as 5 edições da Câmara com PDF em branco.
- Na Câmara, o OCR leu o cabeçalho com ruído ("D.0O.E"), e 98 dos 227 atos
  ficaram em `outro`.
