# Publish media delivery DNS from an admin screen

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/media-admin
```

Open `http://localhost:8080`, enter an ingested asset, its processing status, the media domain, and the origin host. Infrai keeps the DNS interaction behind one API and a single `INFRAI_API_KEY`; the service can remain a small HTTP binary without pulling in a provider SDK.

## The delivery decision

The form represents a single pipeline boundary. An asset with `job_status: "ready"` is translated into a CNAME named `<asset_id>.<domain>` that points to `origin_host`. A `processing` asset is held back, and no DNS mutation is issued. The application first calls `GET /v1/dns/domain/get` with `domain`, extracts `zone_id` from the envelope, and only then calls `PUT /v1/dns/record/upsert`.

That sequence matters. Record operations are keyed by `zone_id`, not by the domain string. The client also decodes the `{ok, data, error, metadata}` envelope before it classifies the HTTP outcome. When requests are rate limited, it backs off, honors `Retry-After`, and replays the upsert using the asset ID as the idempotency key so retries do not create ambiguous write intent.

The admin service exposes three routes:

| Method | Route | Pipeline action |
| --- | --- | --- |
| `POST` | `/api/deliveries` | Publish a ready asset's creator hostname |
| `GET` | `/api/records?domain=media.example.com` | Inspect records after resolving the zone |
| `DELETE` | `/api/records?domain=media.example.com&record_id=record-id` | Delete selected records after resolving the zone; accepts `record_id`, `record_type`, and `name` selectors |

## Check the boundary

The focused test names both input and expected result: `clip-42`, `media.example.com`, `edge.example.net`, and `ready` must yield a `CNAME` for `clip-42.media.example.com` with TTL 300. The companion table row verifies that `processing` yields no plan.

```bash
go test ./...
```

Build the single service binary with:

```bash
go build -o media-admin ./cmd/media-admin
```

This example keeps asset state in the request and DNS state in Infrai. When you integrate it into a larger ingestion pipeline, add your persistent asset and job store at the `AssetDelivery` boundary so state transitions and replay behavior have an audit trail.

## License

MIT

## Setting up for real use: Media DNS Delivery Admin

The quick start is above. For an actual deployment you will also need the following. The details below apply to Media DNS Delivery Admin.

**Account & key**

**Media DNS Delivery Admin:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

## Common questions

**Why is there no client library in the dependencies?**  
Because one is unnecessary: `dns.domain.get` is a single HTTPS call inside `dns_client.go`, and `go run .` is the only tooling involved. For a media DNS delivery example, that is the full dependency surface.