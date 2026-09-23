variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

variable "name_prefix" {
  description = "Prefixo de todos os recursos, ex.: diario-dev."
  type        = string
}

variable "deployer_sa_email" {
  description = "Conta do workflow de deploy (output DEPLOYER_SA do bootstrap)."
  type        = string
}

variable "neon_region" {
  description = "Região do Neon. Veja as opções em https://neon.tech/docs/introduction/regions"
  type        = string
  default     = "aws-us-east-2"
}

variable "resend_api_key" {
  description = "Chave do Resend. Vazia = e-mails apenas registrados em log."
  type        = string
  default     = ""
  sensitive   = true
}

variable "email_from" {
  description = "Remetente, ex.: Alertas <alertas@seudominio.com>."
  type        = string
  default     = ""
}

variable "public_web_url" {
  description = "URL pública do site (domínio próprio). Vazia = URL do Cloud Run."
  type        = string
  default     = ""
}

variable "source_url" {
  type    = string
  default = "https://do.pmsg.rj.gov.br/"
}

variable "scraper_schedule" {
  description = "Cron da coleta (fuso America/Sao_Paulo)."
  type        = string
  default     = "0 8,13,19 * * 1-6"
}

variable "camara_scraper_schedule" {
  description = "Cron da coleta do Diário da Câmara (fuso America/Sao_Paulo)."
  type        = string
  default     = "30 21 * * 1-6"
}

variable "scraper_lookback_days" {
  type    = number
  default = 3
}

variable "api_max_instances" {
  type    = number
  default = 3
}

variable "dump_schedule" {
  description = "Cron do dump semanal da base (fuso America/Sao_Paulo)."
  type        = string
  default     = "0 4 * * 0"
}
