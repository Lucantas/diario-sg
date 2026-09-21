variable "name" {
  type = string
}

variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "image" {
  description = "Imagem inicial; depois o pipeline de deploy assume."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "command" {
  description = "Sobrescreve o ENTRYPOINT (ex.: [\"/app/worker\"])."
  type        = list(string)
  default     = null
}

variable "service_account_email" {
  type = string
}

variable "env" {
  type    = map(string)
  default = {}
}

variable "secret_env" {
  description = "Mapa NOME_DA_VARIAVEL => id do secret no Secret Manager."
  type        = map(string)
  default     = {}
}

variable "public" {
  description = "Se true, qualquer pessoa pode invocar (API e site)."
  type        = bool
  default     = false
}

variable "invokers" {
  description = "Mapa chave-estática => membro IAM com permissão de invocar."
  type        = map(string)
  default     = {}
}

variable "min_instances" {
  description = "0 = escala a zero (custo zero parado, com cold start)."
  type        = number
  default     = 0
}

variable "max_instances" {
  type    = number
  default = 2
}

variable "cpu" {
  type    = string
  default = "1"
}

variable "memory" {
  type    = string
  default = "512Mi"
}

variable "concurrency" {
  type    = number
  default = 80
}

variable "timeout_seconds" {
  type    = number
  default = 60
}
