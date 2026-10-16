# Opt-in ACL update definitions

`client.ACLs.UpdateDefinition(ctx, id, definition)` accepts an `ACLDefinition`
raw-message map, sends a direct object via `PUT /acls/{id}/`, and preserves the
full response object (including populated grantees) rather than the legacy ACL
projection. The documented response is HTTP 200. Legacy models and methods are
unchanged. No response links are followed.

This update-specific policy deliberately differs from CreateDefinition: it
forwards `owner`, `resource_uri` and `uuid`, including null, so the complete
[public Editing example](https://docs.cloudsigma.com/en/latest/acls.html#editing)
is representable without silently stripping fields. The same public schema
marks these fields readonly. Forwarding does **not** establish writability,
server acceptance, ignored echoes, identity checks, or rejection semantics.
That public readonly-input-handling facet remains unresolved. Unknown fields
and empty objects are representable, not guaranteed server-valid.

Nil definitions, invalid JSON values and unsafe route identifiers are rejected
before dispatch. Identifier validation is the SDK conservative segment policy,
not an assertion of public UUID grammar. Raw values retain numeric tokens,
null/empty/absent distinctions and nested content; JSON encoding can normalize
whitespace and escaping. Duplicate keys follow encoding/json semantics.

Callers are responsible for the complete intended metadata replacement under
the [common metadata contract](https://docs.cloudsigma.com/en/latest/meta.html).
There is no implicit read/merge, atomic update guarantee or invented ACL
permission semantics. Do not mutate input maps/bytes concurrently with the call.
Returned raw values are caller-owned. Null/nonobject successful responses are
refused; permissive object decoding is not schema validation. Response.Meta is
not populated.

Offline tests include unchanged complete public request/response JSON bodies,
including readonly echoes, and synthetic boundary cases. Mock success proves
client representation and transport behavior only, not live ACL mutation or
provider acceptance.
