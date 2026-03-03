## 0.1.0 (Unreleased)

FEATURES:

* **New Resource**: `landscape_script_v1` — create, update, and archive legacy V1 scripts.
* **New Data Source**: `landscape_script_v1` — read a V1 script by ID.
* **New Resource**: `landscape_script_v2` — create, update, and archive V2 scripts.
* **New Data Source**: `landscape_script_v2` — read a V2 script by ID.
* **New Resource**: `landscape_script_v2_attachment` — attach files to a V2 script.
* **New Data Source**: `landscape_script_v2_attachment` — read a script attachment by ID.
* **New Resource**: `landscape_script_profile` — create, update, and archive script profiles (event, recurring, one-time triggers).
* **New Data Source**: `landscape_script_profile` — read a script profile by ID.

NOTES:

* Built against Landscape OpenAPI `v0.0.9` via `landscape-go-api-client v0.0.9`.
* Supports both email/password and access-key/secret-key authentication.
