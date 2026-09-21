# Copie estes valores para as variáveis do Environment no GitHub
# (Settings > Environments > dev|prod > Variables).
output "TF_STATE_BUCKET" {
  value = google_storage_bucket.tfstate.name
}

output "WIF_PROVIDER" {
  value = google_iam_workload_identity_pool_provider.github.name
}

output "TERRAFORM_SA" {
  value = google_service_account.gh_terraform.email
}

output "DEPLOYER_SA" {
  value = google_service_account.gh_deployer.email
}
