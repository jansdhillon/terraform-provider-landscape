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
  title        = "my v2 script"
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

  depends_on = [landscape_script_v2.active]
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
