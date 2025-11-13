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

data "landscape_v1_script" "my_v1_script" {
  id = 21434
}

data "landscape_v2_script" "my_v2_script" {
  id = 21433
}
