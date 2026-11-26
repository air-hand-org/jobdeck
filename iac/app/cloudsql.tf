resource "google_sql_database_instance" "shared" {
  name             = "jobdeck-postgres"
  region           = var.region
  database_version = "POSTGRES_18"
  settings {
    tier = "db-f1-micro"

    database_flags {
      name = "cloudsql.iam_authentication"
      value = "on"
    }
  }
}

resource "google_sql_user" "provisioner" {
  name = trimsuffix(data.terraform_remote_state.bootstrap.outputs.deployer_service_account, ".gserviceaccount.com")
  instance = google_sql_database_instance.shared.name
  type = "CLOUD_IAM_SERVICE_ACCOUNT"

  database_roles = ["cloudsqlsuperuser"]
}

resource "google_sql_database" "auth" {
  name     = "auth"
  instance = google_sql_database_instance.shared.name
}

resource "google_sql_database" "job" {
  name     = "job"
  instance = google_sql_database_instance.shared.name
}

resource "google_sql_database" "ui" {
  name     = "ui"
  instance = google_sql_database_instance.shared.name
}

resource "google_sql_provision_script" "custom_functions" {
  for_each = toset([
    google_sql_database.auth.name,
    google_sql_database.job.name,
    google_sql_database.ui.name,
  ])

  script = file("${path.module}/sql/custom_role_functions.sql")
  description = "sql to create custom functions"
  instance = google_sql_database_instance.shared.name
  database = each.value

  depends_on = [
    google_sql_user.provisioner
  ]
}

resource "google_sql_provision_script" "custom_roles" {
  for_each = toset([
    google_sql_database.auth.name,
    google_sql_database.job.name,
    google_sql_database.ui.name,
  ])

  script = templatefile(
    "${path.module}/sql/custom_roles.sql.tftpl",
    {
      app = each.value
    }
  )
  description = "sql to create custom roles"
  instance = google_sql_database_instance.shared.name
  database = each.value

  depends_on = [
    google_sql_user.provisioner,
    google_sql_provision_script.custom_functions[each.value],
  ]
}

// FIXME
resource "google_sql_user" "job_runtime_user" {
  name = trimsuffix(google_service_account.runtime["job-api"].email, ".gserviceaccount.com")
  instance = google_sql_database_instance.shared.name
  type = "CLOUD_IAM_SERVICE_ACCOUNT"

  depends_on = [
    google_sql_provision_script.custom_roles["job"]
  ]

  database_roles = [
    "job-runtime"
  ]
}
