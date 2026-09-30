# Licenças ambientais, alertas com filtro e vínculos de sócios

Data: 29/09/2026. Origem: o teste com a licença da Mical Invest (LMP nº
008/2026, processo 3.813/2026, Diário de 12/06/2026), empresa de uma nora
do prefeito. A busca achou o ato, mas o tipo era `outro`, o alerta não
aceitava filtro, a tela de busca não tinha tema e nenhum padrão olhava
licença nem vínculo de sócio com agente público.

## O que entra

1. **Tipo `licenca_ambiental`.** O parser classifica como licença
   ambiental:
   - título que começa com concessão, renovação, cancelamento, suspensão
     ou cassação de licença, com um marcador ambiental no texto ("licença
     municipal prévia/de instalação/de operação", "licença ambiental",
     "meio ambiente");
   - ato que cairia em `outro` e cujo texto abre com "torna público que
     recebeu/requereu/obteve … licença" (aviso cujo título virou o nome
     da empresa).
   Licença de servidor ("Licença Prêmio", "Licença sem vencimentos") não
   tem o marcador e fica como está. Rótulo: "Licença ambiental".
2. **Alerta com filtros.** A assinatura por termo ganha filtros opcionais
   de tipo, órgão, tema e diário; o termo passa a ser opcional quando há
   ao menos um filtro. Cada edição nova é comparada pela mesma busca do
   site (`ActFilter` com a edição). O assunto do e-mail diz os filtros. A
   assinatura por entidade não muda.
3. **Tema na tela de busca.** Seletor "Tema" (hoje só meio ambiente) na
   busca do site, levado para a URL, a exportação, o RSS e o alerta.
4. **Padrão "Empresa nova recebe licença ambiental".** Concessão de
   licença ambiental a empresa (matriz) aberta até 2 anos antes da
   publicação. O limite é maior que o dos contratos (180 dias) porque um
   empreendimento é licenciado meses depois de a empresa ser aberta para
   ele; a Mical foi aberta 19 meses antes da licença.
5. **Padrão "Sócio com o nome de agente público".** Sócio pessoa física
   de empresa que contratou ou recebeu licença, com nome de três palavras
   ou mais, igual ao de um agente político da folha ou a um nome que
   aparece em ato de nomeação ou exoneração. Ligação por nome, sempre
   `fraca` e dita como "possível" (ADR 0006): homônimos são comuns. O
   cruzamento com os atos fica numa view materializada, atualizada pelo
   job `receita` e pela reindexação, porque custa de 10 a 25 s.

## Fora

- Parentesco: nenhuma base aberta diz quem é cônjuge ou filho de quem, e
  comparar sobrenomes em massa seria montar perfil de parentes (ADR 0006).
- Doações de campanha: depende do TSE, bloqueado.
- Porte do empreendimento (unidades, área) lido do texto da licença.
- Alerta por e-mail para os padrões.

## Testes

- Parser: aviso de licença ambiental, licença de servidor, aviso com o
  nome da empresa no título.
- Domínio: assunto do alerta com filtros; validação (termo ou filtro);
  os dois padrões com casos de fronteira (dias, homônimo curto, sócio
  pessoa jurídica).
- Integração: alerta com filtro dispara só para o ato certo da edição;
  padrões novos a partir do banco.
- Web: estado da busca com tema na URL, na exportação e no RSS.
