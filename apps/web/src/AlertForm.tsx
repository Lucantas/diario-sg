import { FormEvent, ReactNode, useState } from "react";

interface AlertFormProps {
  title: string;
  description: string;
  onSubscribe: (email: string) => Promise<unknown>;
  children?: ReactNode;
}

export function AlertForm({ title, description, onSubscribe, children }: AlertFormProps) {
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState<"idle" | "sending" | "sent" | "error">("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setStatus("sending");
    try {
      await onSubscribe(email);
      setStatus("sent");
    } catch (err) {
      setError((err as Error).message);
      setStatus("error");
    }
  }

  return (
    <section id="alerta" className="alert" aria-label="Alerta por e-mail">
      {status === "sent" ? (
        <div className="notice" role="status">
          <strong className="notice-title">Você vai receber um e-mail</strong>
          Enviamos um link para {email}. O alerta começa a valer depois que você confirmar.
        </div>
      ) : (
        <form onSubmit={onSubmit}>
          <h2>{title}</h2>
          <p>{description}</p>
          <div className="alert-row">
            <label htmlFor="email" className="visually-hidden">Seu e-mail</label>
            <input id="email" type="email" required value={email}
              onChange={(e) => setEmail(e.target.value)} placeholder="seu@email.com" />
            <button type="submit" disabled={status === "sending"}>Criar alerta</button>
          </div>
          {status === "error" && <p className="notice notice-error" role="alert">{error}</p>}
        </form>
      )}
      {children}
    </section>
  );
}
