package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type Message struct {
	To, Subject, HTML string
}

type Sender interface {
	Send(ctx context.Context, m Message) error
}

type Notifier struct {
	sender Sender
	webURL string
}

var _ ports.Notifier = (*Notifier)(nil)

func NewNotifier(s Sender, publicWebURL string) *Notifier {
	return &Notifier{sender: s, webURL: strings.TrimRight(publicWebURL, "/")}
}

func (n *Notifier) link(path, token string) string {
	return n.webURL + path + "?token=" + url.QueryEscape(token)
}

var confirmTmpl = template.Must(template.New("c").Parse(`
<p>Recebemos um pedido para avisar este e-mail quando o Diário Oficial da Prefeitura ou da Câmara de São Gonçalo publicar algo sobre:</p>
<p><strong>{{.Query}}</strong></p>
<p><a href="{{.Link}}">Confirmar alerta</a></p>
<p>Se não foi você, ignore esta mensagem: nada será enviado sem confirmação.</p>`))

func (n *Notifier) SendConfirmation(ctx context.Context, s domain.Subscription) error {
	var b bytes.Buffer
	if err := confirmTmpl.Execute(&b, map[string]string{"Query": s.Query, "Link": n.link("/confirmar", s.ConfirmToken)}); err != nil {
		return err
	}
	return n.sender.Send(ctx, Message{To: s.Email, Subject: "Confirme seu alerta do Diário Oficial", HTML: b.String()})
}

var matchesTmpl = template.Must(template.New("m").Funcs(template.FuncMap{
	"plain": func(s string) string { return strings.NewReplacer("⟦", "", "⟧", "").Replace(s) },
}).Parse(`
<p>A edição {{.Edition}} do {{.SourceName}} de {{.Date}} tem {{len .Hits}} resultado(s) para <strong>{{.Query}}</strong>:</p>
{{range .Hits}}<p><strong>{{.Title}}</strong><br>{{plain .Snippet}}</p>{{end}}
<p><a href="{{.Source}}">Abrir a edição original</a></p>
<p style="font-size:12px"><a href="{{.Unsub}}">Cancelar este alerta</a></p>`))

func (n *Notifier) SendMatches(ctx context.Context, s domain.Subscription, g domain.Gazette, hits []domain.ActHit) error {
	var b bytes.Buffer
	err := matchesTmpl.Execute(&b, map[string]any{
		"Edition": g.EditionNumber, "SourceName": domain.SourceName(g.Source), "Date": g.PublishedAt.Format("02/01/2006"), "Hits": hits,
		"Query": s.Query, "Source": g.SourceURL, "Unsub": n.link("/cancelar", s.UnsubscribeToken),
	})
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("“%s” no Diário da %s de %s", s.Query, domain.SourceLabel(g.Source), g.PublishedAt.Format("02/01"))
	return n.sender.Send(ctx, Message{To: s.Email, Subject: subject, HTML: b.String()})
}

type LogSender struct{ Log *slog.Logger }

func (l LogSender) Send(_ context.Context, m Message) error {
	l.Log.Info("e-mail (modo log)", "to", m.To, "subject", m.Subject, "html", m.HTML)
	return nil
}

type ResendSender struct {
	APIKey string
	From   string
	HTTP   *http.Client
}

func NewResendSender(apiKey, from string) *ResendSender {
	return &ResendSender{APIKey: apiKey, From: from, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (r *ResendSender) Send(ctx context.Context, m Message) error {
	payload, _ := json.Marshal(map[string]any{"from": r.From, "to": []string{m.To}, "subject": m.Subject, "html": m.HTML})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("resend: status %d: %s", resp.StatusCode, msg)
	}
	return nil
}
