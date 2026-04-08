# Rackspace Spot MCP Server — Improvement Task List

- **Generated:** 2026-02-25
- **API version:** v1
- **Source:** Spot_Public_API.yaml
- **Method:** Full API spec vs MCP tool surface comparison

---

## Overview

| ID | Priority | Type | Tool | Summary |
|----|----------|------|------|---------|
| T-01 | 🔴 High | Change | `spot_nodepools_update` | Add autoscaling params |
| T-02 | 🔴 High | Add | `pricing_get_history` | New: price history per server class |
| T-03 | 🔴 High | Add | `pricing_get_percentiles` | New: price percentile distribution |
| T-04 | 🟠 Medium | Add | `cloudspaces_update` | New: edit existing cloudspace |
| T-05 | 🟠 Medium | Change | `ondemand_nodepools_update` | Add autoscaling params (parity with spot) |
| T-06 | 🟠 Medium | Add | `pricing_get_comparable` | New: hyperscaler price comparison |
| T-07 | 🟡 Low | Add | `auth_get_token` | New: OAuth token generation |
| T-08 | 🟡 Low | Change | `cloudspaces_get_config` | Expose full kubeconfig generation options |
| B-01 | 🔴 Bug | Fix | `spot_nodepools_update` | $ prefix in bidprice causes HTTP 422 |
| B-02 | 🔴 Bug | Fix | `spot_nodepools_create` | BM class empty minBidPrice causes ParseFloat crash |

---

## Feature Tasks

### T-01 · `spot_nodepools_update` · Add autoscaling parameters
- **Priority:** 🔴 High
- **Type:** Change (extend existing tool)
- **API endpoint:** `PATCH /apis/ngpc.rxt.io/v1/namespaces/{namespace}/spotnodepools/{name}`
- **API fields to add:**
  ```
  spec.autoscaling.enabled    boolean  — enable/disable autoscaling
  spec.autoscaling.minNodes   integer  — floor when idle
  spec.autoscaling.maxNodes   integer  — ceiling under load
  ```
- **New tool params:** `autoscaling_enabled`, `autoscaling_min_nodes`, `autoscaling_max_nodes`
- **Notes:** Confirmed present in API schema under `SpotNodePoolSpec`. Currently the tool silently ignores these fields. Attempted to use today — only workaround is direct kubectl patch.

---

### T-02 · `pricing_get_history` · New tool: price history per server class
- **Priority:** 🔴 High
- **Type:** Add (new tool)
- **API endpoint:** `GET /price_history.json?server_class={name}`
- **API host:** `https://ngpc-prod-public-data.s3.us-east-2.amazonaws.com` (unauthenticated)
- **Response schema:**
  ```
  {
    auction: string,          — server class name
    history: [
      { run_at: int64,        — unix timestamp of auction
        hammer_price: float } — final clearing price in USD
    ]
  }
  ```
- **New tool params:** `serverclass` (string)
- **Notes:** No auth token required. Most valuable missing tool — enables volatility analysis, P95 bid recommendations, and pattern detection (e.g. price spikes at certain hours). Currently `pricing_get` returns only a single current snapshot with no historical context.

---

### T-03 · `pricing_get_percentiles` · New tool: price percentile distribution
- **Priority:** 🔴 High
- **Type:** Add (new tool)
- **API endpoint:** `GET /percentiles.json`
- **API host:** `https://ngpc-prod-public-data.s3.us-east-2.amazonaws.com` (unauthenticated)
- **Response schema:**
  ```
  {
    regions: {
      {region}: {
        serverclasses: {
          {class}: { percentiles: {...} }
        }
      }
    }
  }
  ```
- **New tool params:** `region` (optional filter), `serverclass` (optional filter)
- **Notes:** No auth token required. Precomputed statistical distributions — agent can use P50/P95 values to set bids without manually processing raw history. Complements T-02.

---

### T-04 · `cloudspaces_update` · New tool: edit existing cloudspace
- **Priority:** 🟠 Medium
- **Type:** Add (new tool)
- **API endpoint:** `PATCH /apis/ngpc.rxt.io/v1/namespaces/{namespace}/cloudspaces/{name}`
- **API fields:**
  ```
  spec.kubernetesVersion      string   — trigger k8s version upgrade
  spec.HAControlPlane         boolean  — enable/disable HA control plane
  spec.preEmptionWebhookURL   string   — update preemption webhook
  spec.cni                    string   — change CNI plugin
  ```
- **New tool params:** `name`, `kubernetes_version`, `ha_control_plane`, `preemption_webhook_url`, `cni`
- **Notes:** Currently there is no MCP tool to mutate an existing cloudspace — only create and delete. Kubernetes version upgrades are completely blocked without direct API access.

---

### T-05 · `ondemand_nodepools_update` · Add autoscaling parameters
- **Priority:** 🟠 Medium
- **Type:** Change (extend existing tool)
- **API endpoint:** `PATCH /apis/ngpc.rxt.io/v1/namespaces/{namespace}/ondemandnodepools/{name}`
- **API fields to add:**
  ```
  spec.autoscaling.enabled    boolean
  spec.autoscaling.minNodes   integer
  spec.autoscaling.maxNodes   integer
  ```
- **New tool params:** `autoscaling_enabled`, `autoscaling_min_nodes`, `autoscaling_max_nodes`
- **Notes:** Same gap as T-01 but for on-demand node pools. Required for feature parity between spot and on-demand pool management.

---

### T-06 · `pricing_get_comparable` · New tool: hyperscaler price comparison
- **Priority:** 🟠 Medium
- **Type:** Add (new tool)
- **API endpoint:** `GET /comparable_prices.json`
- **API host:** `https://ngpc-prod-public-data.s3.us-east-2.amazonaws.com` (unauthenticated)
- **Response schema:**
  ```
  {
    regions: {
      {region}: {
        {serverclass}: {
          hyperscaler_average_price: float
        }
      }
    }
  }
  ```
- **New tool params:** `region` (optional filter), `serverclass` (optional filter)
- **Notes:** No auth token required. Benchmarks Rackspace spot prices against AWS/GCP/Azure equivalents. Useful context when an agent is advising on cost optimization or platform selection.

---

### T-07 · `auth_get_token` · New tool: OAuth token generation
- **Priority:** 🟡 Low
- **Type:** Add (new tool)
- **API endpoint:** `POST /oauth/token` (unauthenticated)
- **Request body:** `{ client_id, client_secret, grant_type }`
- **Notes:** MCP server likely handles auth internally. Low value for interactive use but useful if the MCP is used in automation pipelines where token lifecycle management matters.

---

### T-08 · `cloudspaces_get_config` · Expose full kubeconfig generation options
- **Priority:** 🟡 Low
- **Type:** Change (extend existing tool)
- **API endpoint:** `POST /apis/auth.ngpc.rxt.io/v1/generate-kubeconfig` (unauthenticated)
- **Notes:** Tool exists and works. Review whether the API supports additional generation options (expiry, user scoping, cluster targeting) that are not currently exposed as tool parameters.

---

## Bugs

### B-01 · `spot_nodepools_update` · $ prefix in bidprice causes HTTP 422
- **Severity:** 🔴 High
- **Affected tools:** `spot_nodepools_update`, likely `spot_nodepools_create`
- **Error observed:**
  ```
  HTTP 422: bidPrice can only be a positive number up to three decimal places
  ```
- **Root cause:** Tool sends bid price with leading `$` (e.g. `$0.005`) but API expects a bare float (e.g. `0.005`).
- **Fix:** Strip leading `$` from `bidprice` parameter before constructing API request body.

---

### B-02 · `spot_nodepools_create` · BM server class empty minBidPrice causes ParseFloat crash
- **Severity:** 🔴 High
- **Affected tools:** `spot_nodepools_create`, `spot_nodepools_update`
- **Affected server classes:** `gp.bm2.*-lon`, `gp.bm2.*-iad`, `io.bm2-*` (all Bare Metal)
- **Error observed:**
  ```
  BidPrice validation failed: failed to parse serverclass min bid price:
  strconv.ParseFloat: parsing "": invalid syntax
  ```
- **Root cause:** Bare Metal classes have an empty `minBidPrice` field in the API response (returned as `"$"` with no value). The admission webhook crashes when trying to parse it before comparing against the user's bid.
- **Fix (MCP side):** Before creating a pool, call `serverclasses_get` and validate that `minBidPrice` is non-empty. Return a descriptive error like `"Server class gp.bm2.large-lon is unavailable for spot bidding (deprecated)"` instead of forwarding to the API.
- **Fix (Rackspace side):** Remove deprecated BM classes from the `serverclasses_list` response or set their `availability` field to `unavailable`.
- **Context:** Rackspace has confirmed BM classes are being deprecated and have been removed from the UI. API listing is stale.

---

## Notes for agents using this document

- All three pricing endpoints (T-02, T-03, T-06) are **unauthenticated** and served from S3. They can be called with a plain HTTP GET and require no bearer token.
- Bid price format accepted by the API is a **bare float** (e.g. `0.030`), not a dollar-prefixed string. Validate before sending.
- Bid price must be a **multiple of 0.005** and within the min/max range for the server class. Always call `pricing_get` first to get the current market price and validate against `serverclasses_get` for the min bid floor.
- The `wonCount` field on a node pool is the authoritative indicator of actual provisioned nodes vs `desired`. A `status: Partial` with `wonCount < desired` means the bid is at or below the clearing price — increase the bid.
- Autoscaling (`enabled: true`) requires both `minNodes` and `maxNodes` to be set. Setting `enabled: false` without clearing those fields is safe.
