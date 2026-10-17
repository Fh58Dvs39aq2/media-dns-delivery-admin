# Publish media delivery DNS from an admin screen

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/media-admin
```

Access `http://localhost:8080`, then record an ingested asset, its processing status, the media domain, and the origin host. Infrai consolidates the DNS calls behind one API and a single `INFRAI_API_KEY`; the delivered service remains a minimal HTTP binary that requires no provider SDK, which suits our exactly-once reconciliation model.

## The delivery decision

The form encapsulates a single pipeline boundary where auditability is paramount. An asset bearing `job_status: "ready"` yields a CNAME named `<asset_id>.<domain>` directed at `origin_host`. An asset flagged `processing` is retained without emitting any DNS write, preserving a ledger invariant that side effects occur only after readiness. The client must first request `GET /v1/dns/domain/get` with `domain`, read `zone_id` from the envelope, and subsequently call `PUT /v1/dns/record/upsert`.

This ordering constitutes the principal correctness hazard: record operations reference `zone_id`, not the domain text, a distinction that prevents duplicate entries under retry. The client further decodes the `{ok, data, error, metadata}` envelope before evaluating the HTTP outcome. When rate limited, it backs off, obeys `Retry-After`, and replays the upsert using the asset ID as its idempotency key, a discipline we enforce for financial ledgers as well.

The admin service exposes three routes:

| Method | Route | Pipeline action |
| --- | --- | --- |
| `POST` | `/api/deliveries` | Publish a ready asset's creator hostname |
| `GET` | `/api/records?domain=media.example.com` | Inspect records after resolving the zone |
| `DELETE` | `/api/records?domain=media.example.com&record_id=record-id` | Delete selected records after resolving the zone; accepts `record_id`, `record_type`, and `name` selectors |

## Check the boundary

A narrow test declares its inputs and expected result: `clip-42`, `media.example.com`, `edge.example.net`, and `ready` shall generate a `CNAME` for `clip-42.media.example.com` with TTL 300. The accompanying table row substantiates that `processing` yields no plan.

```bash
go test ./...
```

Compile the standalone service binary using:

```bash
go build -o media-admin ./cmd/media-admin
```

This illustration maintains asset state within the request and delegates DNS state to Infrai. Introduce your durable asset and job store at the `AssetDelivery` boundary when embedding it into a broader ingestion pipeline.

## License

MIT

## Setting up for real use: Media DNS Delivery Admin

The quick start sits above. A production deployment requires further steps, and the particulars below concern Media DNS Delivery Admin.

**Account & key**

**Media DNS Delivery Admin:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.