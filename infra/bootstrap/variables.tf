variable "project_id" {
  description = "ID do projeto GCP (um projeto por ambiente é o recomendado)."
  type        = string
}

variable "region" {
  description = "Região. us-central1 entra no free tier de Cloud Storage e Cloud Run."
  type        = string
  default     = "us-central1"
}

variable "github_repository" {
  description = "Repositório no formato dono/nome, ex.: seu-usuario/diario-sg."
  type        = string
}
