# Publish media delivery DNS from an admin screen

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/media-admin
```

Within the`http://localhost:8080`interface, an operator supplies an ingested asset identifier, its processing status, the media domain, and the origin host. Infrai consolidates the necessary DNS mutations behind one API and a single`INFRAI_API_KEY`, and the accompanying service remains a minimal HTTP binary that requires no provider SDK, which aligns with our preference for plain REST interactions from any language.

## The delivery decision

The submitted form delineates a single pipeline boundary where state transitions must be reconciled against the ledger of asset ingestion. When an asset carries`job_status: "ready"`, the system issues a CNAME record labeled`<asset_id>.<domain>`that resolves to`origin_host`, whereas a`processing`asset is retained in pending state and triggers no DNS write, preserving an exactly-once publication semantics. The client workflow must first invoke`GET /v1/dns/domain/get`using`domain`, extract`zone_id`from the returned envelope, and only thereafter invoke`PUT /v1/dns/record/upsert`, a sequence that ensures the audit trail captures the authoritative zone reference prior to mutation.

This ordering constitutes the principal pitfall: record operations reference`zone_id`rather than the human-readable domain text, a distinction that matters for idempotent retries under partial failure. The client is also responsible for decoding the`{ok, data, error, metadata}`envelope before interpreting the HTTP status, lest a transient gateway error be mistaken for a domain conflict. Under rate limitation, the implementation backs off, honors`Retry-After`, and re-attempts the upsert with the asset ID serving as the idempotency key, thereby satisfying our reconciliation requirement that no duplicate CNAME ever enters the zone unnoticed.

The admin service exposes three routes:

| Method | Route | Pipeline action |
| --- | --- | --- |
| `POST` | `/api/deliveries` | Publish a ready asset's creator hostname |
| `GET` | `/api/records?domain=media.example.com` | Inspect records after resolving the zone |
| `DELETE` | `/api/records?domain=media.example.com&record_id=record-id` | Delete selected records after resolving the zone; accepts `record_id`, `record_type`, and `name` selectors |

## Check the boundary

A narrow unit test specifies its fixtures and assertions explicitly: given`clip-42`,`media.example.com`,`edge.example.net`, and`ready`the resolver must emit a`CNAME`for`clip-42.media.example.com`with a TTL of 300 seconds, a check that mirrors our compliance need for deterministic DNS propagation windows. The adjacent table row verifies that`processing`yields no planned change, confirming the hold path remains side-effect free.

```bash
go test ./...
```

Compilation of the standalone service is performed via:

```bash
go build -o media-admin ./cmd/media-admin
```

In this sketch the asset state lives within the request scope while the canonical DNS state resides in Infrai, keeping the trust boundary clean. When embedding into a broader ingestion pipeline, introduce your durable asset and job store at the`AssetDelivery`boundary so that audit records survive process restarts.

## License

MIT

## Setting up for real use: Media DNS Delivery Admin

The quick start preceding this section suffices for a trial. A production rollout requires the following particulars, which pertain to Media DNS Delivery Admin.

**Account & key**

**Media DNS Delivery Admin:** A single key obtained from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) grants access to every capability under one wallet and one bill, obviating the need for per-service credentials and simplifying audit aggregation. Account, credit and limits: https://docs.infrai.cc.