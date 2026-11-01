# Read-only API probe matrix — 2026-10-31

This receipt records unauthenticated, non-mutating probes against
`https://next.cloudsigma.com/api/2.0/`. No credentials were used. `OPTIONS`
describes deployment routing capability; it does not prove request schemas or
authenticated behavior.

## OPTIONS and unauthenticated GET

| Endpoint | OPTIONS | Unauthenticated GET | Observation |
|---|---:|---:|---|
| `balance/` | 200, `GET` | 401 | Authenticated singleton read remains unqualified. |
| `usage/` | 200, `GET` | 401 | Docs define `poll_time_gt`, `poll_time_lt`, `limit`; native query projection is still needed. |
| `burstusage/` | 200, `GET` | 401 | Read-only, date-filtered surface. |
| `dailyburstusage/` | 200, `GET` | 401 | Read-only, date-filtered surface. |
| `pricing/` | 200, `GET` | 200 JSON | Anonymous response currently exposes `current`, `next`, `meta`, and a large `objects` list. This is deployment/configuration data and should not be assumed safe as a stable public contract without authenticated parity checks. |
| `discount/` | 200, `GET` | 200 JSON | Anonymous response currently returns `{meta,objects}` with `period` and `value`; docs describe the same shape. |
| `ledger/` | 200, `GET` | 401 | Read-only transaction list; schema/query contract still needs qualification. |
| `notification_preferences/` | 200, `GET,PUT,DELETE,POST` | 401 | Mutation statuses/examples are inconsistent in docs; keep lifecycle work blocked. |
| `notification_contacts/` | 200, `GET,PUT,DELETE,POST` | 401 | Mutation and identity contract still needs qualification. |
| `acls/` | 200, `GET,PUT,POST,DELETE` | 401 | ACL ownership and nested permissions need a separate contract. |
| `virtual_routers/` | 200, `GET,PUT,DELETE,POST` | 401 | The current docs also use `virtualrouters` (without underscore) in routing tables; do not choose a route based on OPTIONS alone. |
| `tunnels/` | 200, `GET,POST,DELETE` | 401 | Docs state detail GET, create 201, update 200, delete 204, but the OPTIONS response omits PUT. This is a deployment/documentation inconsistency; no native write should be inferred. |

## Public documentation evidence

- [Billing](https://docs.cloudsigma.com/en/latest/billing.html) documents pricing,
  discount, balance, and transaction-list GET surfaces. Discount examples use a
  paginated `{meta,objects}` response with decimal values represented as strings.
- [Usage](https://docs.cloudsigma.com/en/latest/usage.html) documents a
  paginated GET with `poll_time_gt`, `poll_time_lt`, and `limit` query parameters.
- [Virtual router VPN tunnels](https://docs.cloudsigma.com/en/latest/virtualrouters_vpn_tunnels.html)
  documents GET/POST/PUT/DELETE semantics, while the live OPTIONS response for
  `/tunnels/` omitted PUT.
- The current docs routing table contains both `virtual_routers`-style probe
  behavior and `virtualrouters` legacy paths. This needs authenticated route
  qualification before adding a provider-owned resource.

## Safe next actions

1. Add native JSON data sources for usage, balance, burst usage, and daily burst
   usage only after their query and pagination behavior is covered by synthetic
   tests.
2. Treat pricing and discount as read-only dynamic JSON candidates; preserve
   response size limits and avoid plan-time hidden calls.
3. Keep notification, ACL, virtual-router, tunnel, and ledger mutations out of
   Terraform until authenticated request/response and ownership contracts are
   established.
