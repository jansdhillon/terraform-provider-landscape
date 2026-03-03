# terraform-provider-landscape

Terraform provider for [Landscape](https://ubuntu.com/landscape). Manages Landscape scripts and script profiles as Terraform resources.

## Requirements

- Terraform >= 1.6
- Go >= 1.23 (to build from source)

## Usage

```terraform
terraform {
  required_providers {
    landscape = {
      source  = "jansdhillon/landscape"
      version = "~> 0.1"
    }
  }
}

provider "landscape" {
  base_url   = var.landscape_base_url
  access_key = var.landscape_access_key
  secret_key = var.landscape_secret_key
}
```

Email/password auth is also supported:

```terraform
provider "landscape" {
  base_url = var.landscape_base_url
  email    = var.landscape_email
  password = var.landscape_password
  account  = var.landscape_account  # optional
}
```

All provider arguments can also be set via environment variables: `LANDSCAPE_BASE_URL`, `LANDSCAPE_ACCESS_KEY`, `LANDSCAPE_SECRET_KEY`, `LANDSCAPE_EMAIL`, `LANDSCAPE_PASSWORD`, `LANDSCAPE_ACCOUNT`.

## Resources and data sources

| Type        | Name                             | Description                                            |
| ----------- | -------------------------------- | ------------------------------------------------------ |
| resource    | `landscape_script_v1`            | Legacy V1 script                                       |
| resource    | `landscape_script_v2`            | V2 script with interpreter line                        |
| resource    | `landscape_script_v2_attachment` | File attachment for a V2 script                        |
| resource    | `landscape_script_profile`       | Script profile (event, recurring, or one-time trigger) |
| data source | `landscape_script_v1`            | Read a V1 script by ID                                 |
| data source | `landscape_script_v2`            | Read a V2 script by ID                                 |
| data source | `landscape_script_v2_attachment` | Read a script attachment by ID                         |
| data source | `landscape_script_profile`       | Read a script profile by ID                            |

See [docs/](docs/) or the [Terraform Registry](https://registry.terraform.io/providers/jansdhillon/landscape) for full attribute reference.

## Example

```terraform
resource "landscape_script_v2" "deploy" {
  title      = "deploy-app"
  code       = file("scripts/deploy.sh")
  username   = "ubuntu"
  time_limit = 300
}

resource "landscape_script_profile" "on_enroll" {
  title      = "post-enrollment deploy"
  script_id  = landscape_script_v2.deploy.id
  username   = "ubuntu"
  time_limit = 300
  trigger = {
    type       = "event"
    event_type = "post_enrollment"
  }
}

resource "landscape_script_profile" "nightly" {
  title      = "nightly deploy"
  script_id  = landscape_script_v2.deploy.id
  username   = "ubuntu"
  time_limit = 300
  tags       = ["prod"]
  trigger = {
    type        = "recurring"
    interval    = "0 2 * * *"
    start_after = "2026-04-01T00:00:00Z"
  }
}
```

## Building from source

```shell
git clone https://github.com/jansdhillon/terraform-provider-landscape
cd terraform-provider-landscape
go install
```

To regenerate docs:

```shell
make generate
```

To run unit tests:

```shell
go test ./...
```

Acceptance tests run against a real Landscape instance and require credentials in the environment:

```shell
make testacc
```

## Development

The provider is generated from the [landscape-openapi-spec](https://github.com/jansdhillon/landscape-openapi-spec) via [landscape-go-api-client](https://github.com/jansdhillon/landscape-go-api-client). When the spec is updated, the client is automatically regenerated and Dependabot opens a bump PR against this repo.
