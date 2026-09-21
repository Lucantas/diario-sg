variable "name" {
  type = string
}

variable "project_id" {
  type = string
}

variable "project_number" {
  type = string
}

variable "push_endpoint" {
  description = "URL HTTPS que recebe as mensagens (ex.: worker/events/x)."
  type        = string
}

variable "audience" {
  description = "Audience do token OIDC: a URL base do serviço Cloud Run."
  type        = string
}

variable "push_service_account_email" {
  description = "Conta cuja identidade assina o token OIDC do push."
  type        = string
}

variable "publishers" {
  description = "Mapa chave-estática => membro IAM que pode publicar no tópico."
  type        = map(string)
  default     = {}
}

variable "ack_deadline_seconds" {
  description = "Tempo para o consumidor responder (máx. 600)."
  type        = number
  default     = 300
}

variable "max_delivery_attempts" {
  type    = number
  default = 5
}
