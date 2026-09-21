terraform {
  required_version = ">= 1.6"
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
