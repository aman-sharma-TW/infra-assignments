resource "kubernetes_namespace" "app" {
  metadata {
    name = var.namespace
  }
}

resource "kubernetes_secret" "postgres_credentials" {
  metadata {
    name      = "postgres-credentials"
    namespace = kubernetes_namespace.app.metadata[0].name
  }

  data = {
    password            = var.db_password
    postgres-password   = var.db_password
  }
}

resource "helm_release" "postgresql" {
  name       = "postgresql"
  namespace  = kubernetes_namespace.app.metadata[0].name
  repository = "oci://registry-1.docker.io/bitnamicharts"
  chart      = "postgresql"
  version    = "18.7.10"

  set {
    name  = "auth.username"
    value = var.db_user
  }

  set {
    name  = "auth.database"
    value = var.db_name
  }

  set {
    name  = "auth.existingSecret"
    value = kubernetes_secret.postgres_credentials.metadata[0].name
  }

  set {
    name  = "auth.secretKeys.adminPasswordKey"
    value = "postgres-password"
  }

  set {
    name  = "auth.secretKeys.userPasswordKey"
    value = "password"
  }

  set {
    name  = "primary.persistence.size"
    value = "1Gi"
  }

  set {
    name  = "primary.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "primary.resources.requests.memory"
    value = "128Mi"
  }

  depends_on = [kubernetes_secret.postgres_credentials]
}

resource "helm_release" "config_service" {
  name      = "config-service"
  namespace = kubernetes_namespace.app.metadata[0].name
  chart     = "${path.module}/../helm/config-service"

  set {
    name  = "env.DB_HOST"
    value = "postgresql.${var.namespace}.svc.cluster.local"
  }

  set {
    name  = "env.DB_USER"
    value = var.db_user
  }

  set {
    name  = "env.DB_NAME"
    value = var.db_name
  }

  depends_on = [helm_release.postgresql]
}
