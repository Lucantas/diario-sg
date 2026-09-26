# Padrões do fornecedor (Entrega 3, parte 3)

Data: 2026-09-26. Os quatro padrões novos da Entrega 3, que cruzam as
contratações do Diário com o cadastro da Receita (parte 1) e as sanções
da CGU (parte 2). Entram em `/padroes` e `GET /v1/patterns` ao lado dos
atuais, com a mesma linguagem: "padrão para verificar", regra em texto e
os atos que acionaram.

## Fatos que orientam o desenho

Medidos na base local em 26/09/2026, com as 1.298 contratações dos
painéis (atos da mesma empresa ligados por processo ou contrato):

- Empresa aberta menos de 180 dias antes da primeira contratação: 4.
- Capital social menor que 10% do valor da contratação: 17 sem filtro,
  com ruído de cooperativas e associações (capital sem sentido de
  garantia) e de contratações de centenas de reais; 9 só com sociedades e
  empresários, em contratações de pelo menos R$ 100 mil.
- Sócio em comum: 34 pares de sócio e grupo, mas vários grupos repetem as
  mesmas empresas com sócios diferentes; juntando, 19 grupos.
- Mesmo endereço (logradouro, número, complemento, bairro, cidade e CEP):
  5 grupos.
- Contratação publicada no período de uma sanção que alcança São Gonçalo:
  0. Das 406 sanções carregadas, 34 alcançam o município (declaração de
  inidoneidade, abrangência "Todas as Esferas" ou órgão sancionador de São
  Gonçalo); a base só tem as sanções que estão no CEIS e no CNEP desde a
  primeira carga, e as contratações são antigas.

## Decisões

1. **Contratação** é a mesma dos painéis (`SupplierContracts`, a partir de
   `panelContracts`): uma por grupo de atos da mesma empresa, com a data do
   primeiro ato, o maior valor de contrato ou, sem ele, o de ata.
2. **Empresa aberta pouco antes da primeira contratação**
   (`empresa_nova_contratada`): a primeira contratação da empresa no Diário
   foi publicada até 179 dias depois da abertura da matriz. Só matriz: a
   data de abertura da filial não diz a idade da empresa. Contratação
   antes da abertura fica de fora (costuma ser CNPJ digitado errado).
3. **Capital social menor que 10% do contratado**
   (`capital_menor_que_contrato`): a maior contratação da empresa, de pelo
   menos R$ 100 mil, passa de dez vezes o capital social. Dez por cento é
   o máximo que a lei deixa exigir de capital mínimo (art. 31, § 3º, da Lei
   8.666; art. 69, § 4º, da Lei 14.133). Só naturezas com capital de
   sociedade (sociedades, EIRELI e empresário individual).
4. **Fornecedores com sócio em comum** (`socio_em_comum`): duas ou mais
   empresas contratadas (CNPJ básico diferente) com o mesmo sócio, pelo
   nome e pelo documento como a Receita publica (CPF mascarado). Grupos
   com as mesmas empresas viram um só. O achado **não nomeia a pessoa
   física** (ADR 0006): diz quantos sócios em comum e manda para a página
   de cada empresa, onde o sócio aparece; sócio pessoa jurídica é nomeado.
5. **Fornecedores no mesmo endereço** (`endereco_em_comum`): duas ou mais
   empresas contratadas com o mesmo endereço completo, sem acento e
   pontuação. Endereço sem número fica de fora.
6. **Fornecedor sancionado contratado** (`sancionado_contratado`):
   contratação publicada entre o início e o fim de uma sanção da empresa
   (qualquer estabelecimento) que alcança São Gonçalo: declaração de
   inidoneidade, abrangência "Todas as Esferas em todos os Poderes" ou
   órgão sancionador de São Gonçalo. Impedimento e suspensão de outro ente
   não entram (art. 156, § 4º, da Lei 14.133 restringe ao ente).
7. **Leitura na consulta**, como os padrões atuais: `ListPatterns` recebe
   uma segunda fonte (`SupplierPatternSource`: atos dos painéis, perfis da
   Receita e sanções). Sem migration.
8. **Página `/padroes`**: sem mudança de componente; cada achado traz os
   atos (o de maior valor de cada contratação) e o texto com os nomes e os
   links das empresas no detalhe.

## Fora do escopo

- Padrões com a Câmara (as contratações são da Prefeitura).
- Histórico do cadastro: capital, endereço e sócios são do mês mais
  recente da Receita, não da data do contrato.

## Testes

- Domínio: cada finder com casos que acionam e parecidos que não acionam
  (filial, contratação antes da abertura, cooperativa, contratação abaixo
  do piso, mesma empresa em dois estabelecimentos, sanção de outro ente,
  contratação fora do período).
- Integração: `/v1/patterns` traz os cinco padrões novos com um caso cada,
  e o de sócio em comum não traz o nome da pessoa física.
- Base local: contagens acima, conferência manual de um caso de cada.

**Pronto quando:** `/padroes` mostra os cinco padrões com os casos da base
local, cada um com teste de caso que aciona e de caso parecido que não.
