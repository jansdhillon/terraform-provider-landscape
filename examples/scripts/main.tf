terraform {
  required_providers {
    landscape = {
      source = "jansdhillon/landscape"
    }
  }
}

provider "landscape" {}

resource "landscape_script_v1" "v1" {
  title    = "buy for a dollar"
  code     = file("v1.sh")
  username = "ubuntu"
}

resource "landscape_script_v2" "active" {
  title        = uuid()
  code         = file("v2.sh")
  username     = "jan"
  time_limit   = 500
  access_group = "global"
}

resource "landscape_script_v2_attachment" "my_attachment" {
  script_id = landscape_script_v2.active.id
  filename  = "attachment.txt"
  content   = <<-EOT
  my attachment for a v2 script
  EOT
}

resource "landscape_script_profile" "on_enroll" {
  title      = "post-enrollment setup ${uuid()}"
  script_id  = landscape_script_v2.active.id
  username   = "root"
  time_limit = 300
  trigger = {
    type       = "event"
    event_type = "post_enrollment"
  }
}

resource "landscape_script_profile" "hourly" {
  title      = "hourly check ${uuid()}"
  script_id  = landscape_script_v2.active.id
  username   = "ubuntu"
  time_limit = 60
  tags       = ["web", "prod"]
  trigger = {
    type        = "recurring"
    interval    = "0 * * * *"
    start_after = "2026-04-01T00:00:00Z"
  }
}

data "landscape_script_profile" "read_profile" {
  id = landscape_script_profile.on_enroll.id
}

data "landscape_script_v1" "data_v1" {
  id = landscape_script_v1.v1.id
}

data "landscape_script_v2" "data_v2" {
  id = landscape_script_v2.active.id
}

data "landscape_script_v2_attachment" "data_attachment" {
  id        = landscape_script_v2_attachment.my_attachment.id
  script_id = data.landscape_script_v2.data_v2.id
}

output "v1_script" {
  value = landscape_script_v1.v1
}

output "v2_script" {
  value = landscape_script_v2.active
}

output "data_v1" {
  value = data.landscape_script_v1.data_v1
}

output "data_v2" {
  value = data.landscape_script_v2.data_v2
}

output "v2_attachment" {
  value = landscape_script_v2_attachment.my_attachment
}

output "data_v2_attachment" {
  value = data.landscape_script_v2_attachment.data_attachment
}

resource "landscape_gpg_key" "mirror_key" {
  name     = "test-mirror-key"
  material = file("gpg.key")
}

resource "landscape_distribution" "ubuntu" {
  name         = "ubuntu"
  access_group = "global"
}

resource "landscape_series" "noble" {
  name          = "noble"
  distribution  = landscape_distribution.ubuntu.name
  pockets       = ["release", "updates", "security", "proposed", "backports"]
  components    = ["main", "universe", "multiverse", "restricted"]
  architectures = ["amd64"]
  gpg_key       = landscape_gpg_key.mirror_key.name
  mirror_uri    = "http://archive.ubuntu.com/ubuntu"
}

resource "landscape_repository_profile" "noble_mirror" {
  title         = "apply-ubuntu-noble-mirror"
  pockets       = ["release", "updates", "security", "proposed", "backports"]
  series        = landscape_series.noble.name
  distribution  = landscape_distribution.ubuntu.name
  all_computers = false
  tags          = ["noble"]
}

output "gpg_key" {
  value     = landscape_gpg_key.mirror_key
  sensitive = true
}

output "distribution" {
  value = landscape_distribution.ubuntu
}

output "series" {
  value = landscape_series.noble
}

output "repository_profile" {
  value = landscape_repository_profile.noble_mirror
}
