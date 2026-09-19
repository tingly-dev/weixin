package wecom

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tingly-dev/weixin/types"
)

// TestUploadMedia_UsesRealServerIssuedIDs is the regression test for the
// fabricated upload_id/media_id bug: before the ack-body fix, uploadInit and
// uploadFinish never read the server's actual response and instead reused a
// locally-generated string as both the upload_id sent in every chunk/finish
// call and the media_id returned to the caller. A real WeCom server issues
// its own IDs, so a client that ignores them sends chunks against an
// upload_id the server never allocated.
func TestUploadMedia_UsesRealServerIssuedIDs(t *testing.T) {
	const serverUploadID = "server-upload-abc"
	const serverMediaID = "server-media-xyz"

	var chunkUploadIDsMu sync.Mutex
	var chunkUploadIDs []string
	var finishUploadID atomic.Value

	wsURL, cleanup := newTestWsServer(t, func(t *testing.T, conn *websocket.Conn, connNum int) {
		acceptAuth(t, conn)
		for {
			frame := readFrame(t, conn)
			if frame == nil {
				return
			}
			switch frame.Cmd {
			case CmdUploadMediaInit:
				writeAck(t, conn, frame.Headers.ReqID, 0, "", map[string]interface{}{
					"upload_id": serverUploadID,
				})
			case CmdUploadMediaChunk:
				var chunk UploadChunkBody
				if err := parseFrameBody(frame.Body, &chunk); err != nil {
					t.Errorf("parse chunk body: %v", err)
				}
				chunkUploadIDsMu.Lock()
				chunkUploadIDs = append(chunkUploadIDs, chunk.UploadID)
				chunkUploadIDsMu.Unlock()
				writeAck(t, conn, frame.Headers.ReqID, 0, "", nil)
			case CmdUploadMediaFinish:
				var finish UploadFinishBody
				if err := parseFrameBody(frame.Body, &finish); err != nil {
					t.Errorf("parse finish body: %v", err)
				}
				finishUploadID.Store(finish.UploadID)
				writeAck(t, conn, frame.Headers.ReqID, 0, "", map[string]interface{}{
					"type":       "file",
					"media_id":   serverMediaID,
					"created_at": "1700000000",
				})
				// Upload is done; stop reading. Looping on would race the
				// test's own completion against the client.Disconnect() in
				// t.Cleanup, since a read error from that close would then
				// call t.Errorf from a goroutine after the test has already
				// returned.
				return
			default:
				t.Errorf("unexpected frame cmd from client: %q", frame.Cmd)
			}
		}
	})
	defer cleanup()

	client := newConnectedTestClient(t, wsURL)
	bot := &WecomBot{client: client}

	tmp, err := os.CreateTemp(t.TempDir(), "upload-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := tmp.WriteString("hello wecom upload"); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	tmp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := bot.UploadMedia(ctx, &types.MediaUploadRequest{
		FilePath: tmp.Name(),
		FileName: "upload.txt",
	})
	if err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}

	if result.EncryptQuery != serverMediaID {
		t.Fatalf("returned media id = %q, want the server-issued %q (not a fabricated local id)", result.EncryptQuery, serverMediaID)
	}

	chunkUploadIDsMu.Lock()
	defer chunkUploadIDsMu.Unlock()
	if len(chunkUploadIDs) == 0 {
		t.Fatal("no chunk frames observed")
	}
	for _, id := range chunkUploadIDs {
		if id != serverUploadID {
			t.Fatalf("chunk sent upload_id=%q, want the server-issued %q", id, serverUploadID)
		}
	}
	if got := finishUploadID.Load(); got != serverUploadID {
		t.Fatalf("finish sent upload_id=%v, want the server-issued %q", got, serverUploadID)
	}
}
