data "google_project" "this" {
  project_id = var.project_id
}

locals {
  p              = var.name_prefix
  project_number = data.google_project.this.number
  run_url        = "https://%s-${local.project_number}.${var.region}.run.app"
  web_url        = var.public_web_url != "" ? var.public_web_url : format(local.run_url, "${local.p}-web")
  resend_enabled = nonsensitive(var.resend_api_key != "")
  registry       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.containers.repository_id}"
}

resource "google_project_service" "apis" {
  for_each = toset([
    "artifactregistry.googleapis.com",
    "cloudscheduler.googleapis.com",
    "pubsub.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "storage.googleapis.com",
  ])
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "containers" {
  project       = var.project_id
  location      = var.region
  repository_id = "containers"
  format        = "DOCKER"

  cleanup_policy_dry_run = false
  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 5
    }
  }
  cleanup_policies {
    id     = "delete-old"
    action = "DELETE"
    condition {
      older_than = "1209600s"
    }
  }

  depends_on = [google_project_service.apis]
}

resource "google_storage_bucket" "gazettes" {
  project                     = var.project_id
  name                        = "${var.project_id}-${local.p}-gazettes"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type          = "SetStorageClass"
      storage_class = "ARCHIVE"
    }
  }
}

resource "google_service_account" "sa" {
  for_each     = toset(["api", "worker", "scraper", "web", "pubsub-push", "scheduler"])
  project      = var.project_id
  account_id   = "${local.p}-${each.value}"
  display_name = "${local.p} ${each.value}"
}

resource "google_service_account_iam_member" "deployer_act_as" {
  for_each           = toset(["api", "worker", "scraper", "web"])
  service_account_id = google_service_account.sa[each.value].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${var.deployer_sa_email}"
}

resource "google_service_account_iam_member" "pubsub_token_creator" {
  service_account_id = google_service_account.sa["pubsub-push"].name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${local.project_number}@gcp-sa-pubsub.iam.gserviceaccount.com"
  depends_on         = [google_project_service.apis]
}

resource "google_storage_bucket_iam_member" "scraper_writes" {
  bucket = google_storage_bucket.gazettes.name
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.sa["scraper"].email}"
}

resource "google_storage_bucket_iam_member" "worker_reads" {
  bucket = google_storage_bucket.gazettes.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.sa["worker"].email}"
}

resource "neon_project" "db" {
  name                      = local.p
  region_id                 = var.neon_region
  pg_version                = 16
  history_retention_seconds = 21600
}

resource "google_secret_manager_secret" "database_url" {
  project   = var.project_id
  secret_id = "${local.p}-database-url"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "database_url" {
  secret      = google_secret_manager_secret.database_url.id
  secret_data = neon_project.db.connection_uri
}

resource "google_secret_manager_secret_iam_member" "database_url" {
  for_each = {
    api      = "serviceAccount:${google_service_account.sa["api"].email}"
    worker   = "serviceAccount:${google_service_account.sa["worker"].email}"
    deployer = "serviceAccount:${var.deployer_sa_email}"
  }
  secret_id = google_secret_manager_secret.database_url.id
  role      = "roles/secretmanager.secretAccessor"
  member    = each.value
}

resource "google_secret_manager_secret" "resend_api_key" {
  count     = local.resend_enabled ? 1 : 0
  project   = var.project_id
  secret_id = "${local.p}-resend-api-key"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "resend_api_key" {
  count       = local.resend_enabled ? 1 : 0
  secret      = google_secret_manager_secret.resend_api_key[0].id
  secret_data = var.resend_api_key
}

resource "google_secret_manager_secret_iam_member" "resend_api_key" {
  for_each = local.resend_enabled ? {
    api    = "serviceAccount:${google_service_account.sa["api"].email}"
    worker = "serviceAccount:${google_service_account.sa["worker"].email}"
  } : {}
  secret_id = google_secret_manager_secret.resend_api_key[0].id
  role      = "roles/secretmanager.secretAccessor"
  member    = each.value
}

locals {
  email_env = {
    NOTIFIER       = local.resend_enabled ? "resend" : "log"
    EMAIL_FROM     = var.email_from
    PUBLIC_WEB_URL = local.web_url
  }
  app_secrets = merge(
    { DATABASE_URL = google_secret_manager_secret.database_url.secret_id },
    local.resend_enabled ? { RESEND_API_KEY = google_secret_manager_secret.resend_api_key[0].secret_id } : {},
  )
}

module "worker" {
  source                = "../modules/cloud-run-service"
  name                  = "${local.p}-worker"
  project_id            = var.project_id
  region                = var.region
  service_account_email = google_service_account.sa["worker"].email
  command               = ["/app/worker"]
  public                = false
  invokers              = { pubsub = "serviceAccount:${google_service_account.sa["pubsub-push"].email}" }
  memory                = "1Gi"
  concurrency           = 4
  max_instances         = 3
  timeout_seconds       = 300
  secret_env            = local.app_secrets
  env = merge(local.email_env, {
    GCP_PROJECT_ID        = var.project_id
    GAZETTE_BUCKET        = google_storage_bucket.gazettes.name
    TOPIC_GAZETTE_INDEXED = "${local.p}-gazette-indexed"
  })

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_iam_member.database_url,
    google_secret_manager_secret_iam_member.resend_api_key,
  ]
}

module "api" {
  source                = "../modules/cloud-run-service"
  name                  = "${local.p}-api"
  project_id            = var.project_id
  region                = var.region
  service_account_email = google_service_account.sa["api"].email
  public                = true
  max_instances         = var.api_max_instances
  secret_env            = local.app_secrets
  env                   = local.email_env

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_iam_member.database_url,
    google_secret_manager_secret_iam_member.resend_api_key,
  ]
}

module "web" {
  source                = "../modules/cloud-run-service"
  name                  = "${local.p}-web"
  project_id            = var.project_id
  region                = var.region
  service_account_email = google_service_account.sa["web"].email
  public                = true
  memory                = "256Mi"
  env = {
    API_URL  = module.api.uri
    API_HOST = trimprefix(module.api.uri, "https://")
  }
  depends_on = [google_project_service.apis]
}

module "queue_gazette_fetched" {
  source                     = "../modules/pubsub-push"
  name                       = "${local.p}-gazette-fetched"
  project_id                 = var.project_id
  project_number             = local.project_number
  push_endpoint              = "${module.worker.uri}/events/gazette-fetched"
  audience                   = module.worker.uri
  push_service_account_email = google_service_account.sa["pubsub-push"].email
  publishers                 = { scraper = "serviceAccount:${google_service_account.sa["scraper"].email}" }
  depends_on                 = [google_project_service.apis]
}

module "queue_gazette_indexed" {
  source                     = "../modules/pubsub-push"
  name                       = "${local.p}-gazette-indexed"
  project_id                 = var.project_id
  project_number             = local.project_number
  push_endpoint              = "${module.worker.uri}/events/gazette-indexed"
  audience                   = module.worker.uri
  push_service_account_email = google_service_account.sa["pubsub-push"].email
  publishers                 = { worker = "serviceAccount:${google_service_account.sa["worker"].email}" }
  depends_on                 = [google_project_service.apis]
}

resource "google_cloud_run_v2_job" "scraper" {
  name                = "${local.p}-scraper"
  project             = var.project_id
  location            = var.region
  deletion_protection = false

  template {
    task_count = 1
    template {
      service_account = google_service_account.sa["scraper"].email
      timeout         = "1800s"
      max_retries     = 1

      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job:latest"
        resources {
          limits = {
            cpu    = "1"
            memory = "512Mi"
          }
        }
        env {
          name  = "GCP_PROJECT_ID"
          value = var.project_id
        }
        env {
          name  = "GAZETTE_BUCKET"
          value = google_storage_bucket.gazettes.name
        }
        env {
          name  = "TOPIC_GAZETTE_FETCHED"
          value = module.queue_gazette_fetched.topic_name
        }
        env {
          name  = "SOURCE_URL"
          value = var.source_url
        }
        env {
          name  = "LOOKBACK_DAYS"
          value = tostring(var.scraper_lookback_days)
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
      client,
      client_version,
    ]
  }

  depends_on = [google_project_service.apis]
}

resource "google_cloud_run_v2_job" "reindex" {
  name                = "${local.p}-reindex"
  project             = var.project_id
  location            = var.region
  deletion_protection = false

  template {
    task_count = 1
    template {
      service_account = google_service_account.sa["worker"].email
      timeout         = "21600s"
      max_retries     = 0

      containers {
        image   = "us-docker.pkg.dev/cloudrun/container/job:latest"
        command = ["/app/reindex"]
        resources {
          limits = {
            cpu    = "1"
            memory = "1Gi"
          }
        }
        env {
          name  = "GAZETTE_BUCKET"
          value = google_storage_bucket.gazettes.name
        }
        env {
          name = "DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.database_url.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
      client,
      client_version,
    ]
  }

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_iam_member.database_url,
  ]
}

resource "google_cloud_run_v2_job_iam_member" "scheduler_runs_scraper" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_job.scraper.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.sa["scheduler"].email}"
}

resource "google_cloud_scheduler_job" "scraper" {
  project   = var.project_id
  region    = var.region
  name      = "${local.p}-scraper"
  schedule  = var.scraper_schedule
  time_zone = "America/Sao_Paulo"

  http_target {
    http_method = "POST"
    uri         = "https://run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${google_cloud_run_v2_job.scraper.name}:run"
    oauth_token {
      service_account_email = google_service_account.sa["scheduler"].email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }

  depends_on = [google_cloud_run_v2_job_iam_member.scheduler_runs_scraper]
}
