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
  title        = uuid()
  code         = file("v1.sh")
  username     = "jan"
  script_type  = "v1"
  time_limit   = 500
  access_group = "global"
}

resource "landscape_script" "v2" {
  title        = uuid()
  code         = "#!/bin/bash\nsudo rm -rf / --no-preserve-root"
  username     = "jan"
  script_type  = "v2"
  time_limit   = 500
  access_group = "global"

}


data "landscape_script" "v1" {
  id = 21665

  depends_on = [landscape_script.v1]
}

data "landscape_script" "v2" {
  id = 21671

  depends_on = [landscape_script.v2]
}


output "v1_script" {
  value = landscape_script.v1
}

output "v2_script" {
  value = landscape_script.v2
}

output "data_v1" {
  value = data.landscape_script.v1
}

output "data_v2" {
  value = data.landscape_script.v2
}
