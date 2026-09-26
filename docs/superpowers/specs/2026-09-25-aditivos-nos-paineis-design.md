# Aditivos e prorrogações nos painéis

Data: 2026-09-25. Fecha a pendência "Valor de aditivos e prorrogações" dos
painéis (`docs/superpowers/specs/2026-09-24-paineis-design.md`).

## Fatos que orientam o desenho

- O painel conta cada contratação uma vez, pelo maior valor de contrato, e
  deixa aditivos de fora. Um contrato de serviço contínuo prorrogado por
  cinco anos aparece com o valor de um ano só.
- O valor que um aditivo declara nem sempre é o que ele acrescenta:
  - na prorrogação de prazo de serviço contínuo, "VALOR TOTAL DO CONTRATO:
    R$ 10.439.280,00" é o valor do novo período (contrato 016/2022 da FMS,
    prorrogado em 2023, 2024 e 2026 com o mesmo valor);
  - na prorrogação "sem ônus" ou "sem alteração de valor" (obra atrasada), o
    valor citado é o do contrato original e não acrescenta nada;
  - no aditivo de acréscimo, o valor em reais costuma ser o novo total; o
    que ele acrescenta é o "acréscimo de R$ …" dito no texto ou o
    percentual declarado (`acts.declared_increase_bp`) sobre o valor
    contratado;
  - supressão, reajuste sem valor de acréscimo, rerratificação, corrigenda,
    revogação e readequação não têm valor a somar.
- O mesmo aditivo é publicado mais de uma vez (republicação dias depois).

## Decisões

1. Nova coluna "Aditivos e prorrogações" (`amended_cents`) por fornecedor,
   nos totais, por ano e por secretaria. Não entra no contratado nem nas
   atas.
2. O ato é aditivo ou prorrogação quando o tipo é `aditivo` ou quando o
   começo do texto diz termo aditivo, aditamento, prorrogação do contrato,
   do prazo ou da ata, termo de prorrogação, "objeto … prorroga" ou "fica
   prorrogado". Rerratificação, corrigenda, revogação, readequação,
   reajuste sem acréscimo e supressão ficam sem valor, como hoje.
3. Valor do aditivo, na ordem:
   - "acréscimo de R$ X", "valor do acréscimo: R$ X" ou "valor acrescido de
     R$ X": X;
   - prorrogação sem "sem ônus", "sem alteração de valor", "sem acréscimo"
     ou "sem reflexo financeiro": o valor principal lido do ato;
   - percentual declarado: percentual × maior valor contratado da mesma
     contratação;
   - nenhum dos três: sem valor.
4. O aditivo entra na contratação pelo número do processo ou do contrato,
   como os outros atos. Aditivo sem o contrato na base entra como
   contratação só com aditivos (não conta em "contratações").
5. Cada aditivo conta no ano em que foi publicado: o filtro de ano soma o
   contratado das contratações que começaram no ano e os aditivos
   publicados no ano.
6. Aditivos da mesma contratação com o mesmo valor em até 90 dias contam
   uma vez.

## Fora do escopo

- Somar aditivos ao contratado (as bases são diferentes e o valor de um
  aditivo é menos certo).
- Aditivo de ata de registro de preços (acréscimo de quantitativo): entra
  como aditivo, sem separar.

## Testes

- Domínio: valor do aditivo para cada regra, com trechos reais (A4PM
  016/2022, "sem ônus", acréscimo em reais, percentual); republicação conta
  uma vez; aditivo conta no ano em que saiu; contratação só com aditivos
  não conta em "contratações".
- Integração: `/v1/panels/suppliers` traz `amended_cents`.
- Web: coluna e totais, e o texto que explica a coluna.
