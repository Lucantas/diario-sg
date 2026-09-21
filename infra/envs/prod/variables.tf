variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

variable "deployer_sa_email" {
  type = string
}

variable "neon_api_key" {
  type      = string
  sensitive = true
}

variable "resend_api_key" {
  type      = string
  default   = ""
  sensitive = true
}

variable "email_from" {
  type    = string
  default = ""
}

variable "public_web_url" {
  type    = string
  default = ""
}
