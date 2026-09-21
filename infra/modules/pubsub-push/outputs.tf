output "topic_name" {
  value = google_pubsub_topic.this.name
}

output "dlq_subscription" {
  value = google_pubsub_subscription.dlq.name
}
