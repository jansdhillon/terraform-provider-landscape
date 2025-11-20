terraform {
  required_providers {
    landscape = {
      source = "jansdhillon/landscape"
    }
  }
}

provider "landscape" {}

resource "landscape_script" "v1" {
  title        = "dead script walking3"
  code         = file("v1.sh")
  username     = "jan"
  time_limit   = 500
  access_group = "global"
}

resource "landscape_script" "active" {
  title        = "IS THAT HIM WITH THE SOMBREREO ON2"
  code         = file("v2.sh")
  username     = "jan"
  time_limit   = 500
  access_group = "global"
  status       = "ACTIVE"
}

resource "landscape_script_attachment" "my_attachment" {
  script_id = landscape_script.active.id
  filename  = "hello3.txt"
  content   = <<-EOT
  my attachment
  EOT
}

resource "landscape_script_attachment" "my_attachment2" {
  script_id = landscape_script.v1.id
  filename  = "hello.txt"
  content   = <<-EOT
  my attachment
  EOT
}


data "landscape_script" "data_v1" {
  id = landscape_script.v1.id
}

data "landscape_script" "data_v2" {
  id = landscape_script.active.id
}

data "landscape_script_attachment" "data_attachment" {
  id        = landscape_script_attachment.my_attachment
  script_id = data.landscape_script.data_v2
}


output "v1_script" {
  value     = landscape_script.v1
  sensitive = true
}

output "v2_script" {
  value     = landscape_script.active
  sensitive = true
}

output "data_v1" {
  value     = data.landscape_script.data_v1
  sensitive = true
}

output "data_v2" {
  value     = data.landscape_script.data_v2
  sensitive = true
}

output "att" {
  value     = landscape_script_attachment.my_attachment
  sensitive = true
}
