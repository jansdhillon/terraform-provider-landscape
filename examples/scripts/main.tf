terraform {
  required_providers {
    landscape = {
      source = "jansdhillon/landscape"
    }
  }
}

provider "landscape" {
    api_url = "https://landscape.canonical.com"
}

data "landscape_script" "myscript" {
    id = 21145
}
