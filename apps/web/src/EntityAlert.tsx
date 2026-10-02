import { AlertEntityKind, subscribeEntity } from "./api";
import { AlertForm } from "./AlertForm";
import { alertSubjectLabel, entityFeedUrl } from "./entity";

export function EntityAlert({ kind, value, label }: { kind: AlertEntityKind; value: string; label: string }) {
  const subject = alertSubjectLabel(kind, label);
  return (
    <AlertForm
      title={`Avisar quando ${subject} aparecer de novo`}
      description="Você recebe um e-mail no dia em que uma nova edição citar este número."
      onSubscribe={(email) => subscribeEntity(email, kind, value)}
    >
      <p className="feed">
        Prefere RSS? <a href={window.location.origin + entityFeedUrl(kind, value)}>Assine o feed</a> com os 50 atos
        mais recentes que citam este número, sem precisar de e-mail.
      </p>
    </AlertForm>
  );
}
