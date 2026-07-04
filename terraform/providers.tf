provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "kind-config-service"
}

provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = "kind-config-service"
  }
}
