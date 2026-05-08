# HiGoOS API Contract

`openapi.yaml` is the contract source of truth for the HiGoOS Go backend and the `web-pc` frontend. Backend handlers, frontend stores, generated TypeScript clients, and contract tests should all be checked against this file before endpoint or field names change.

## Scope

The contract covers the first full backend development pass, including desktop state, event streams, files, storage, security governance, monitoring, settings, remote access, Docker, downloads, media, assistant flows, and Agent workflows. Responses use a common envelope:

- Success: `{ ok: true, requestId, data, meta? }`
- Paginated success: `{ ok: true, requestId, data, pagination, meta? }`
- Error: `{ ok: false, requestId, error }`

Stable shared schemas include `Error`, `Pagination`, `Task`, and `Event`.

## Generate TypeScript Client

Preferred output path for the frontend is `web-pc/src/api/generated`.
This repository currently checks in a lightweight generated bridge at
`web-pc/src/api/generated/client.ts` so imports and contract ownership are
visible before a full generator is added to CI.

Using `openapi-typescript`:

```sh
npx openapi-typescript server-go/api/openapi.yaml -o web-pc/src/api/generated/schema.d.ts
```

Using OpenAPI Generator:

```sh
npx @openapitools/openapi-generator-cli generate \
  -i server-go/api/openapi.yaml \
  -g typescript-fetch \
  -o web-pc/src/api/generated
```

Generated files should be treated as build artifacts. Runtime concerns such as base URL, auth cookies, CSRF headers, request ID propagation, retry policy, and error normalization belong in `web-pc/src/api/runtime.ts`.

## Validate

Lightweight syntax check:

```sh
python3 - <<'PY'
import yaml
with open("server-go/api/openapi.yaml", "r", encoding="utf-8") as f:
    yaml.safe_load(f)
print("openapi.yaml parses")
PY
```

OpenAPI validation when a validator is available:

```sh
npx @redocly/cli lint server-go/api/openapi.yaml
```

or:

```sh
npx swagger-cli validate server-go/api/openapi.yaml
```

Do not make backend handler names or frontend API calls drift from `operationId`, path, parameter, and schema names in this file. If the implementation needs a different shape, update this contract first, regenerate the client, then update handlers and stores.
