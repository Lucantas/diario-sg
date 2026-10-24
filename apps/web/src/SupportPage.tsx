import { useState } from "react";
import donations from "./doacoes.json";
import { DonationPix, PIX_RECEIVER_NAME, donationPix } from "./donation";
import { goalProgress } from "./goal";
import { PixQrCode } from "./PixQrCode";

const QUESTIONS: [string, string][] = [
  ["Dá para doar todo mês pelo PIX?", "Dá. A maioria dos bancos deixa agendar um PIX recorrente: use a chave acima, escolha o valor e a frequência mensal."],
  ["A doação é dedutível do imposto de renda?", "Não. O Diário SG não é uma ONG, é um projeto de uma pessoa física."],
  ["Doar me dá algum acesso a mais?", "Não. Quem doa vê o mesmo site e usa a mesma API que todo mundo."],
  ["Isso é da Prefeitura?", "Não. O projeto não tem vínculo com a Prefeitura, com a Câmara nem com nenhum órgão público, e não substitui a edição oficial."],
];

export function SupportPage() {
  const pix = donationPix();
  return (
    <main className="page">
      <p className="crumb"><a href="/">← Voltar para a busca</a></p>
      <header className="masthead">
        <p className="eyebrow">Apoie o projeto</p>
        <h1>Ajude a manter o Diário SG</h1>
      </header>

      <div className="support">
        <div className="support-text support-intro">
          <p className="support-lede">
            Quase todo dia a Prefeitura e a Câmara de São Gonçalo publicam quem foi nomeado, que empresa foi contratada e por quanto.
            Está tudo no Diário Oficial, em PDFs de dezenas de páginas que quase ninguém abre. O Diário SG abre por você.
          </p>
          <p>
            Ele é feito e mantido por uma pessoa, no tempo livre, e hoje roda em servidores gratuitos. Se você acha que alguém precisa
            ficar de olho no que sai ali, dá para ajudar a pagar a conta.
          </p>
        </div>

        <article className="support-text support-body">

          <h2>O que já existe</h2>
          <p>
            São 5.286 edições desde 2010 e 162.438 atos separados um a um, com busca em português e alerta por e-mail quando aparece o
            nome, o CNPJ ou o processo que você pediu.
          </p>
          <p>
            Um extrato de contrato não fica solto. Ele leva à página da empresa, onde aparecem os sócios registrados na Receita, as
            punições da CGU e quanto a Prefeitura empenhou e pagou a ela segundo o TCE-RJ. Quinze <a href="/padroes">regras escritas</a> apontam
            casos que merecem uma segunda olhada, como dispensas de licitação fracionadas e aditivos acima do limite da lei. Cada caso
            mostra a regra e os atos que a acionaram, e nenhum deles é uma acusação.
          </p>
          <p>
            A <a href="/dados">base inteira</a> sai toda semana para download, e o <a href="https://github.com/Lucantas/diario-sg">código é aberto</a>.
          </p>

          <h2>Por que precisa de dinheiro</h2>
          <p>Hoje o projeto custa zero real por mês. Isso tem um preço, só que é pago de outro jeito.</p>
          <p>
            A API desliga depois de 15 minutos sem uso, então quem chega primeiro espera ela acordar. O banco gratuito tem 1 GB, e cada
            fonte nova de dados come um pedaço dele. Sair desses limites custa pouco, mas custa todo mês.
          </p>
          <p>
            O que mais pesa é o tempo. A Prefeitura muda o formato do PDF e o parser precisa aprender de novo. Em outubro de 2026 o site
            de proposições da Câmara mudou de endereço e a coleta parou até eu consertar. Cruzar uma fonte nova, como os empenhos do
            TCE-RJ ou as emendas federais, leva semanas. Esse trabalho não aparece na tela, e é ele que mantém a busca confiável.
          </p>

          <h2>Para onde vai o que entrar</h2>
          <ol className="support-list">
            <li>Servidor que não dorme e banco com espaço para crescer.</li>
            <li>Manutenção: consertar a coleta quando um site muda, revisar o que o parser leu errado.</li>
            <li>Fontes novas. A próxima da fila é o andamento das leis na Câmara.</li>
          </ol>

          <h2>O que não muda</h2>
          <p>
            Tudo continua aberto e gratuito, para quem doa e para quem não doa. Não existe área de apoiador, não tem anúncio, e doação
            não compra destaque nem a retirada de nada.
          </p>
          <p>
            Não aceito dinheiro da Prefeitura, da Câmara, de partidos, de mandatos, de candidatos nem de empresas que têm contrato com o
            município. Se uma doação dessas chegar, ela é devolvida.
          </p>
          <p>
            Uma doação mensal pequena vale mais do que uma grande que acontece uma vez, porque a conta também é mensal. Se não dá para
            doar, mandar o site para quem acompanha a política da cidade também ajuda.
          </p>

          <h2>Perguntas</h2>
          <dl className="support-faq">
            {QUESTIONS.map(([q, a]) => (
              <div key={q}>
                <dt>{q}</dt>
                <dd>{a}</dd>
              </div>
            ))}
          </dl>
        </article>

        <aside className="support-box" aria-labelledby="doar-titulo">
          <h2 id="doar-titulo">Doe pelo PIX</h2>
          <GoalMeter />
          {pix ? <PixDetails pix={pix} /> : <p className="notice">A chave PIX entra aqui em breve.</p>}
        </aside>
      </div>
    </main>
  );
}

function GoalMeter() {
  const progress = goalProgress(donations);
  return (
    <div className="goal">
      <p className="support-goal">
        <strong>Meta: {progress.goal} por mês.</strong> É o que tira o site do plano gratuito: servidor que não dorme e banco sem o
        limite de 1 GB.
      </p>
      <p className="goal-numbers" id="meta-numeros">
        <strong>{progress.received}</strong> de {progress.goal} em {progress.month}
      </p>
      <progress className="goal-bar" max={100} value={progress.percent} aria-labelledby="meta-numeros">{progress.percent}%</progress>
      <p className="goal-meta">
        {progress.reached ? "Meta do mês batida. " : `${progress.percent}% da meta. `}
        Atualizado em {progress.updatedOn}.
      </p>
    </div>
  );
}

function PixDetails({ pix }: { pix: DonationPix }) {
  return (
    <>
      <PixQrCode payload={pix.payload} />
      <p className="support-receiver">Recebedor: {PIX_RECEIVER_NAME}</p>
      <CopyField label="Chave PIX" value={pix.key} button="Copiar chave" />
      <CopyField label="PIX copia e cola" value={pix.payload} button="Copiar código" />
      <p className="fineprint">Escolha o valor no app do banco. Para doar todo mês, agende um PIX recorrente.</p>
    </>
  );
}

function CopyField({ label, value, button }: { label: string; value: string; button: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="copy-field">
      <p className="copy-field-label">{label}</p>
      <code className="copy-field-value">{value}</code>
      <button type="button" onClick={copy}>{copied ? "Copiado" : button}</button>
    </div>
  );
}
