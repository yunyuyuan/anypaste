// Package uploadproto defines the small resumable-upload wire protocol shared by
// the server (internal/handler) and the CLI (cmd/cli). It is a minimal subset of
// the tus protocol: the client uploads a file as a sequence of raw-body chunks,
// each appended at a byte offset, so any single request stays small enough to
// pass through proxies with tight body-size/timeout limits (e.g. Cloudflare's
// free tier) and a dropped transfer can resume instead of restarting.
//
// Flow, keyed by the existing paste id (no separate upload session):
//
//	HEAD /file/upload/{id}                      -> HeaderUploadOffset: bytes already stored
//	POST /file/upload/{id}  (raw chunk body)    headers: Upload-Offset, Upload-Length, Upload-Filename
//	                                            -> HeaderUploadOffset: new offset; finalizes when it reaches Upload-Length
//
// A POST whose Upload-Offset disagrees with what the server has gets 409 with the
// authoritative HeaderUploadOffset, so the client re-syncs and continues.
//
// The web client mirrors these constants in web/src/components/CreateModal.tsx —
// keep the two in sync.
package uploadproto

const (
	// HeaderUploadOffset is the byte offset a chunk starts at (request) and the
	// total bytes the server has stored (HEAD/POST response).
	HeaderUploadOffset = "Upload-Offset"
	// HeaderUploadLength is the total size of the complete file, in bytes.
	HeaderUploadLength = "Upload-Length"
	// HeaderUploadFilename is the original file name, used only to derive the
	// stored file's extension at finalize. Percent-encoded by the client.
	HeaderUploadFilename = "Upload-Filename"
)

// ChunkSize is the default chunk the clients send. 5 MiB is comfortably under
// Cloudflare's 100 MB free-tier body cap and uploads in well under the proxy
// timeout even on a slow uplink.
const ChunkSize = 5 << 20

// MaxFileSize is the largest complete upload accepted, matching the previous
// single-request limit.
const MaxFileSize = 1 << 30 // 1 GiB
