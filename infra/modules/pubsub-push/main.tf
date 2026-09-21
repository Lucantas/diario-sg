# Um "pipeline" de fila completo: tópico + assinatura push autenticada
# + política de retry com backoff + dead-letter queue (DLQ) + assinatura
# pull na DLQ para inspeção e reprocessamento manual.

terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

locals {
  # Agente de serviço do Pub/Sub: precisa de permissão para mover mensagens
  # para a DLQ e para gerar tokens OIDC da conta de push.
  pubsub_agent = "serviceAccount:service-${var.project_number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

resource "google_pubsub_topic" "this" {
  project                    = var.project_id
  name                       = var.name
  message_retention_duration = "86400s"
}

resource "google_pubsub_topic" "dlq" {
  project = var.project_id
  name    = "${var.name}-dlq"
}

resource "google_pubsub_subscription" "dlq" {
  project                    = var.project_id
  name                       = "${var.name}-dlq-inspect"
  topic                      = google_pubsub_topic.dlq.id
  ack_deadline_seconds       = 60
  message_retention_duration = "604800s" # 7 dias para investigar

  expiration_policy {
    ttl = ""
  }
}

resource "google_pubsub_subscription" "push" {
  project              = var.project_id
  name                 = "${var.name}-push"
  topic                = google_pubsub_topic.this.id
  ack_deadline_seconds = var.ack_deadline_seconds

  push_config {
    push_endpoint = var.push_endpoint
    oidc_token {
      service_account_email = var.push_service_account_email
      audience              = var.audience
    }
  }

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dlq.id
    max_delivery_attempts = var.max_delivery_attempts
  }

  expiration_policy {
    ttl = ""
  }
}

resource "google_pubsub_topic_iam_member" "dlq_publisher" {
  project = var.project_id
  topic   = google_pubsub_topic.dlq.name
  role    = "roles/pubsub.publisher"
  member  = local.pubsub_agent
}

resource "google_pubsub_subscription_iam_member" "source_subscriber" {
  project      = var.project_id
  subscription = google_pubsub_subscription.push.name
  role         = "roles/pubsub.subscriber"
  member       = local.pubsub_agent
}

resource "google_pubsub_topic_iam_member" "publishers" {
  for_each = var.publishers
  project  = var.project_id
  topic    = google_pubsub_topic.this.name
  role     = "roles/pubsub.publisher"
  member   = each.value
}
