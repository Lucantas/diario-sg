output "web_url" {
  value = module.web.uri
}

output "api_url" {
  value = module.api.uri
}

output "worker_url" {
  value = module.worker.uri
}

output "registry" {
  description = "Prefixo das imagens para o pipeline de deploy."
  value       = local.registry
}

output "gazettes_bucket" {
  value = google_storage_bucket.gazettes.name
}

output "dead_letter_subscriptions" {
  value = [
    module.queue_gazette_fetched.dlq_subscription,
    module.queue_gazette_indexed.dlq_subscription,
  ]
}

output "reindex_job" {
  value = google_cloud_run_v2_job.reindex.name
}
