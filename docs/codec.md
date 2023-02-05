# Codec

Each message is prefixed with a 1 byte (`uint8`) type:
* `DIGEST-REQUEST`: `1`
* `DIGEST-RESPONSE`: `2`
* `DELTA`: `3`

Since only UDP is supported no framing information is needed.

Variable size strings (such as the peer address and state keys and values) are
prefixed with their `uint8` size. This limits the size of these fields to 256
bytes though that should be enough.

### `DIGEST-REQUEST`
Contains a list of entries appended together, each containing:
* Peer address: Encoded string
* Peer version: `uint64`

### `DIGEST-RESPONSE`
This is the same format as `DIGEST-REQUEST` except has a type of
`DIGEST-RESPONSE` to indicate the receiver should not respond with its own
digest.

### `DELTA`
Contains a list of entries appended together, each containing:
* Peer address: Encoded string,
* Key: Encoded string,
* Value: Encoded string,
* Version: `uint64`
