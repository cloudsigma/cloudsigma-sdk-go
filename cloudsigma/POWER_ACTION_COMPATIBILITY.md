# Power action identifier policy

Start, Stop and Shutdown now accept only nonempty ASCII letters (`A-Z`, `a-z`), digits (`0-9`), hyphen (`-`) and underscore (`_`) as the server identifier. This is an intentional SDK route-segment safety policy, **not public UUID grammar** or a guarantee the server accepts every allowed token.

Previously arbitrary nonempty strings were interpolated into the URL. Path/query characters could reshape the destination or selected action. Such strings are now rejected locally before request construction, with nil result/response and no dispatch. Empty IDs retain `ErrEmptyArgument`. No trimming, decoding, normalization or escaping is attempted. Documented UUIDs and safe legacy tokens remain accepted.

This compatibility tightening applies only to these three methods. Signatures, `{}` request bodies, typed responses, HTTP errors and context handling remain unchanged. HTTP 202 acknowledges acceptance, not lifecycle completion; no polling, retries or implicit lifecycle calls are added. Other SDK methods are not certified by this change. VMware's GET-only public surface is not assigned these POST actions.
