output "namespace" {
  value = kubernetes_namespace.app.metadata[0].name
}

output "app_service" {
  value = "config-service.${var.namespace}.svc.cluster.local"
}

output "port_forward_command" {
  value = "kubectl port-forward -n ${var.namespace} svc/config-service 8080:8080"
}
