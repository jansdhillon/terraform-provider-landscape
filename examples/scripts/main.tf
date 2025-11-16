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

resource "landscape_v2_script" "myscript" {
  title    = "0fb72aed-f6d5-8c87-43f5-5abb5613743"
  code     = "#!/bin/bash\nmyscript2"
  username = "jan"
}
