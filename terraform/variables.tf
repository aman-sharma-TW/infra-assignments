variable "namespace" {
  description = "Kubernetes namespace for all resources"
  type        = string
  default     = "config-service"
}

variable "db_password" {
  description = "PostgreSQL password (passed at apply time, never stored in code)"
  type        = string
  sensitive   = true
}

variable "db_user" {
  description = "PostgreSQL username"
  type        = string
  default     = "configservice"
}

variable "db_name" {
  description = "PostgreSQL database name"
  type        = string
  default     = "configservice"
}
