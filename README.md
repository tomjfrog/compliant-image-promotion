# Claims Processor — JFrog Xray + AppTrust Demo

A deliberately simple **hello-world** application used to showcase two JFrog
capabilities end-to-end:

1. **Dependency scanning with JFrog Xray** — both the Go back-end (`go.mod`) and
   the JavaScript front-end (`web/package.json`) carry real dependencies that get
   audited, and the published Docker image gets layer-scanned.
2. **Compliant promotion with JFrog AppTrust** — the build is bundled into an
   immutable Application Version that is promoted through a governed lifecycle:
   **`DEV → NP → PROD`**, with signed evidence and a manual PROD gate.

```
┌────────────┐    ┌─────────────┐    ┌──────────────┐
│  Vite JS   │ ── │  Go back-end │ ── │ Docker image │
│  front-end │    │  (chi router)│    │ (distroless) │
└────────────┘    └─────────────┘    └──────────────┘
        │                                    │
        └──────────── built into ────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │  Xray: jf audit  +  jf docker scan  │
        │        +  jf rt build-scan          │
        └─────────────────┬──────────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │  AppTrust Application Version        │
        │  DEV ──▶ NP ──▶ PROD (gated)         │
        └─────────────────────────────────────┘
```

## Architecture

| Component | Tech | Purpose |
|---|---|---|
| Back-end | Go 1.23 + [`go-chi/chi`](https://github.com/go-chi/chi) | Serves `GET /api/hello`, `GET /api/healthz`, and the static front-end |
| Front-end | Vite + [`axios`](https://github.com/axios/axios) | Single page that calls `/api/hello` and renders the JSON |
| Image | Multi-stage `Dockerfile` → `distroless/static` | Tiny, non-root runtime |

The Go binary serves the built front-end from `STATIC_DIR` (default
`./web/dist`), so the whole app runs from one container on port `8080`.

## Run it locally

Back-end + built front-end (single server):

```bash
cd web && npm install && npm run build && cd ..
go run .                       # http://localhost:8080
```

Front-end with hot reload (proxies /api to the Go server on :8080):

```bash
go run .                       # terminal 1
cd web && npm run dev          # terminal 2 → http://localhost:5173
```

Docker:

```bash
docker build --build-arg APP_VERSION=0.0.1 -t claimsprocessor:local .
docker run --rm -p 8080:8080 claimsprocessor:local
```

## CI/CD workflows

| Workflow | Trigger | What it does |
|---|---|---|
| `.github/workflows/bootstrap-jfrog.yml` | Manual | Creates the Artifactory repositories and registers the AppTrust application. Run **once**. |
| `.github/workflows/ci.yml` | Push to `main` / manual | Audit → build → push → Xray scan → provenance → AppTrust version → promote `DEV → NP → PROD`. |

### Pipeline stages (`ci.yml`)

1. **Xray SCA audit** — `jf audit` scans `go.mod` + `web/package.json`; results
   uploaded to the GitHub *Security → Code scanning* tab as SARIF.
2. **Build, push & scan** — `docker build`, `jf docker push` (records build-info),
   `jf docker scan` (image layers), `jf rt build-scan` (resolved build graph).
3. **Provenance** — `actions/attest-build-provenance` signs SLSA provenance,
   auto-ingested into **JFrog Evidence** by the `setup-jfrog-cli` post step.
4. **AppTrust version + promote** — `jf apptrust version-create` bundles the
   build, then `jf apptrust version-promote` moves it through `DEV → NP`.
5. **PROD gate** — runs in the GitHub `production` environment (add required
   reviewers for a manual approval) before promoting to `PROD`.

## Required GitHub configuration

Set these under **Settings → Secrets and variables → Actions**.

### Variables

| Name | Example | Notes |
|---|---|---|
| `JF_URL` | `https://tomjfrog.jfrog.io` | JFrog Platform base URL |
| `JF_DOCKER_REGISTRY` | `tomjfrog.jfrog.io` | Artifactory Docker registry host |
| `XRAY_WATCH_NAME` | `uhgcomp-watch` | *(optional)* applies an Xray Watch to gate the audit |
| `ENABLE_EVIDENCE` | `true` | *(optional)* turn on signed test/approval evidence |
| `EVIDENCE_KEY_ALIAS` | `uhgcomp-key` | *(optional)* alias of the public key in JFrog Key Management |

### Secrets

| Name | Notes |
|---|---|
| `OIDC_PROVIDER_NAME` | OIDC provider integration configured in JFrog |
| `OIDC_AUDIENCE` | OIDC audience string |
| `EVIDENCE_PRIVATE_KEY` | *(optional)* PEM private key for `jf evd create` |

### Fixed demo identifiers (hard-coded in the workflows)

| Setting | Value |
|---|---|
| JFrog Project | `uhgcomp` |
| AppTrust Application key | `claims-processor-application` |
| Docker image | `claimsprocessor` |
| Build name | `claims-processor-build` |
| Lifecycle stages | `DEV → NP → PROD` |

### Repository topology (stage-mapped)

Each package type (`docker`, `go`, `npm`) has per-stage local repos mapped to a
lifecycle environment, a shared remote proxy, and a virtual that aggregates them.
The image is pushed to the `uhgcomp-docker` virtual (default deployment =
`uhgcomp-docker-dev-local`); AppTrust promotion then moves the version through
the NP and PROD locals.

| Repository | Type | Environment |
|---|---|---|
| `uhgcomp-<type>-remote` | remote (proxy) | `DEV` |
| `uhgcomp-<type>-dev-local` | local | `DEV` |
| `uhgcomp-<type>-np-local` | local | `uhgcomp-NP` |
| `uhgcomp-<type>-prod-local` | local | `PROD` |
| `uhgcomp-<type>` | virtual | `DEV` |

> The `NP` stage is a **custom project environment**, so it is referenced as
> `uhgcomp-NP` (custom project environments are key-prefixed). `DEV` and `PROD`
> are platform-global environments. Repositories were provisioned in the
> `uhgcomp` project via the JFrog Experimental MCP server; `bootstrap-jfrog.yml`
> reproduces the same topology as code.

## Demoing the "compliant" part

- **Show a finding:** `npm install` already reports vulnerabilities in the
  front-end tree — `jf audit` surfaces these in Xray and on the PR/Security tab.
- **Show gating:** flip `--fail=false` to `--fail=true` in `ci.yml`, or attach an
  Xray **Watch** via `XRAY_WATCH_NAME`, to block the build on policy violations.
- **Show governance:** configure **entry/exit gates** on the `DEV`, `NP`, and
  `PROD` stages in AppTrust so promotions only succeed when policy passes; add
  required reviewers on the GitHub `production` environment for a human gate.
