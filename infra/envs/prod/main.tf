terraform {
  required_version = ">= 1.6"
  backend "gcs" {
    prefix = "env/prod"
  }
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    neon = {
      source  = "kislerdm/neon"
      version = ">= 0.6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "neon" {
  api_key = var.neon_api_key
}

module "stack" {
  source            = "../../stack"
  project_id        = var.project_id
  region            = var.region
  name_prefix       = "diario-prod"
  deployer_sa_email = var.deployer_sa_email
  resend_api_key    = var.resend_api_key
  email_from        = var.email_from
  public_web_url    = var.public_web_url
}
