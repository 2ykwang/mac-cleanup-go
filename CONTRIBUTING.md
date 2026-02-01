# Contributing

Thanks for your interest in contributing! Small fixes and bug reports are very welcome.

## Issues
- Use issues for bug reports and feature requests.
- A short description is enough.

## Pull requests
- Briefly describe what changed and why.
- For large changes, please open an issue first to discuss.

## Targets
- Targets define cleanup categories and paths used by the app.
- TL;DR: edit `internal/config/targets.yaml` -> `make targets-validate` -> PR.
- Location: `internal/config/targets.yaml`
- Required: `id`, `name`, `group`, `safety` (safe|moderate|risky), `method` (trash|permanent|builtin|manual)
- `manual` requires `guide`
- `builtin` requires code changes in `internal/target`

### Template
```yaml
- id: my-app-cache
  name: My App Cache
  group: app
  safety: safe
  method: trash
  note: short reason
  paths:
    - "~/Library/.../*"
```

## Development
- fmt: `make fmt`
- test: `make test`
- coverage (patch): `make patch-cover-worktree`
- run: `make run`
