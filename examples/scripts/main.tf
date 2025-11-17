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
  title        = "dead script walkig"
  code         = file("v1.sh")
  username     = "jan"
  time_limit   = 500
  access_group = "global"
}

resource "landscape_script" "active" {
  title        = "dead script walkig2"
  code         = file("v2.sh")
  username     = "jan"
  time_limit   = 500
  access_group = "global"
  status       = "ACTIVE"
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
  value = landscape_script.v1
}

output "v2_script" {
  value = landscape_script.active
}

output "data_v1" {
  value = data.landscape_script.v1
}

output "data_v2" {
  value = data.landscape_script.active
}
