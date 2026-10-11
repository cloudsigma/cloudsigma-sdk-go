# Server.Jobs response-only projection

`Server.Jobs` holds opaque JSON populated by SDK service responses: List, Get,
Create, Update, Clone and CloneWithAvoid. No nested Job schema, polling or
completion semantics are inferred. Published readonly/required metadata does not
prove every response contains jobs.

`json:"-"` excludes Jobs from standard encoding/json marshaling of Server and
embedded Create/Update requests, even when caller-provided raw bytes are invalid
JSON. Direct json.Unmarshal into public Server ignores jobs, leaving an existing
Jobs value untouched. SDK private response decoding is intentionally different.
This exclusion does not cover arbitrary reflection serializers or structural
logging; callers must treat raw response data appropriately.

Fresh independent decodes own independent Jobs bytes, separate from input bytes.
Ordinary Server assignment and slice/value conversions copy the RawMessage slice
header, not its backing data. Copy bytes explicitly if independent mutable values
are needed. On a reused private decoder, absent jobs or a whole null object clears
Jobs; explicit null is retained as `null`, and an empty array as `[]`. Other fields
retain existing merge behavior. SDK methods allocate fresh results. This clearing
policy is deliberate; no whole-object error atomicity is promised by nested
private decoders.

Adding this exported field changes struct shape. External unkeyed literals,
reflection indices, unsafe layout assumptions and embedding/promoted selectors
may be affected. Server was already noncomparable due to existing slices/maps.
Stable request JSON is NOT universal Go source/layout/logging compatibility.
Final maintainer compatibility acceptance and future version classification are
separate decisions; no release version or mandatory bump is chosen here.

An alternative opt-in response wrapper (`*Server` plus Jobs) with new methods
would avoid changing Server's exported shape but introduce parallel endpoint APIs
and shared internal factoring. Changing existing return types to that wrapper
would itself break callers. This candidate provisionally retains the response-only
field for consistent access across existing endpoints, subject to review.

Other unknown server fields and nested field coverage remain incomplete. This
single-field increment does not establish full public API coverage.
