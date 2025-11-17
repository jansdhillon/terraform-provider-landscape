terraform {
  required_providers {
    landscape = {
      source = "jansdhillon/landscape"
    }
  }
}

provider "landscape" {
  base_url = "https://landscape.canonical.com"
}

resource "landscape_script" "v1" {
  title        = "dead script walking"
  code         = file("v1.sh")
  username     = "jan"
  time_limit   = 500
  access_group = "global"
}

resource "landscape_script" "active" {
  title        = "IS THAT HIM WITH THE SOMBREREO ON"
  code         = file("v2.sh")
  username     = "jan"
  time_limit   = 500
  access_group = "global"
  status       = "ACTIVE"
}

resource "landscape_script_attachment" "att" {
  script_id = landscape_script.active.id
  filename  = "hello.txt"
  content   = <<-EOT
  my att
  EOT

  depends_on = [landscape_script.active]
}



data "landscape_script" "v1" {
  id = 21665

  depends_on = [landscape_script.v1]
}

data "landscape_script" "active" {
  id = 21671

  depends_on = [landscape_script.active]
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
  value     = data.landscape_script.v1
  sensitive = true
}

output "data_v2" {
  value     = data.landscape_script.active
  sensitive = true
}

output "att" {
  value     = landscape_script_attachment.att
  sensitive = true
}
