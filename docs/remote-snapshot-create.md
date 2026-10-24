# Remote snapshot creation

Use the dedicated writable contract instead of sending a response model:

```go
request, err := cloudsigma.NewRemoteSnapshotCreateRequest(
    cloudsigma.RemoteSnapshotCreate{
        Drive: &cloudsigma.RemoteSnapshotDriveReference{UUID: driveUUID},
        Name: "backup", // optional
        Location: "destination", // optional; server default when omitted
    },
)
if err != nil { return err }
objects, response, err := client.RemoteSnapshots.CreateDefinitions(ctx, request)
```

A `RemoteSnapshotCreateDefinitionsRequest{Objects: ...}` literal is also supported;
all members must have a drive UUID, validated before any HTTP dispatch. The
constructor copies the slice and each drive reference. It does not confer safe
concurrent mutation of requests. Requests encode a minimal nested drive reference
(UUID and optional resource_uri), never full Drive response fields. Reference
UnmarshalJSON additionally accepts UUID strings; this does not imply the typed
request encoder emits strings. Name and location are writable; drive_meta is
response-only.

POST remotesnapshots returns HTTP 201 with an objects envelope, even for one
object. Batch results retain server order; batches are not atomic. Creating and
transferring are not ready, available means done, unavailable indicates failure.
This SDK call does not poll or infer readiness from a successful POST.

Legacy `RemoteSnapshotCreateRequest` and `RemoteSnapshots.Create` remain supported
with their original JSON behavior, including metadata, tags, and full Drive
serialization. That legacy full-response payload is not a valid minimal create
contract: do not reuse fetched response models as writable inputs. In particular,
the embedded Snapshot marshaler historically omits outer location/drive_meta.
Those bytes are preserved, not silently repaired or advertised as accepted by the
server. Migrate explicitly to the dedicated API to send location correctly.
Existing response models are unchanged; no lossless unknown-field preservation
is claimed.

Both create methods refuse 301/302/303/307/308 before Location parsing or cookie
processing. `errors.Is(err, cloudsigma.ErrRemoteSnapshotCreateRedirect)` indicates
that the original POST outcome is unknown. Do not blindly retry; reconcile first.
No response identity is acknowledged, caller redirect callbacks are not invoked,
and shared HTTP configuration is not changed. Ordinary success cookies, HTTP
errors, cancellation and timeouts retain the existing client behavior. This is
synthetic offline qualification, not live server acceptance or a release.
