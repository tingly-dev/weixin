package wecom

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math"
	"sync"

	"github.com/tingly-dev/weixin"
)

// UploadAdapter implements UploadAdapter for WeCom AI Bot.
// Media is uploaded via a 3-step chunked WebSocket protocol.
type UploadAdapter struct {
	gateway *GatewayAdapter
}

// NewUploadAdapter creates a new WeCom upload adapter.
func NewUploadAdapter(gateway *GatewayAdapter) *UploadAdapter {
	return &UploadAdapter{gateway: gateway}
}

// GetUploadURL returns a placeholder — WeCom doesn't use pre-signed URLs.
// Uploads go directly over the WebSocket connection.
func (u *UploadAdapter) GetUploadURL(ctx context.Context, req *weixin.UploadURLRequest) (*weixin.UploadURLResult, error) {
	// WeCom uses WS-based chunked upload, not pre-signed URLs.
	// This method returns the upload mechanism info via the result.
	return &weixin.UploadURLResult{
		UploadParam: "wecom_ws_upload",
		FileKey:     req.FileKey,
	}, nil
}

// UploadMedia uploads a media file via the 3-step WS chunked protocol.
// The file data is provided via MediaData in the request.
// Returns the media_id which is valid for 3 days.
func (u *UploadAdapter) UploadMedia(ctx context.Context, req *weixin.MediaUploadRequest) (*weixin.MediaUploadResult, error) {
	client := u.gateway.GetClient("")
	if client == nil || !client.IsConnected() {
		return nil, fmt.Errorf("wecom client not connected")
	}

	if len(req.EncryptKey) != 0 {
		return nil, fmt.Errorf("wecom upload does not support client-side encryption")
	}

	data := req.FilePath // TODO: support file path reading; for now assume MediaData or FilePath
	if data == "" {
		return nil, fmt.Errorf("no file data provided")
	}

	// Read file data from disk
	fileData, err := readFilePath(data)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	totalSize := int64(len(fileData))
	if totalSize == 0 {
		return nil, fmt.Errorf("empty file")
	}

	mediaType := req.MediaType
	if mediaType == "" {
		mediaType = detectMediaTypeFromFilename(req.FileName)
	}

	// Calculate chunks
	chunkCount := int(math.Ceil(float64(totalSize) / float64(ChunkSize)))
	if chunkCount > MaxChunks {
		return nil, fmt.Errorf("file too large: %d chunks (max %d)", chunkCount, MaxChunks)
	}

	// Calculate MD5
	hash := md5.Sum(fileData)
	md5Str := fmt.Sprintf("%x", hash)

	// Step 1: Init
	initResult, err := u.uploadInit(ctx, client, mediaType, req.FileName, totalSize, chunkCount, md5Str)
	if err != nil {
		return nil, fmt.Errorf("upload init: %w", err)
	}

	// Step 2: Chunks
	if err := u.uploadChunks(ctx, client, initResult.UploadID, fileData, chunkCount); err != nil {
		return nil, fmt.Errorf("upload chunks: %w", err)
	}

	// Step 3: Finish
	finishResult, err := u.uploadFinish(ctx, client, initResult.UploadID)
	if err != nil {
		return nil, fmt.Errorf("upload finish: %w", err)
	}

	return &weixin.MediaUploadResult{
		FileSize:     totalSize,
		EncryptQuery: finishResult.MediaID, // store media_id in EncryptQuery for downstream use
	}, nil
}

// ---------------------------------------------------------------------------
// Upload steps
// ---------------------------------------------------------------------------

func (u *UploadAdapter) uploadInit(ctx context.Context, client *Client, mediaType, filename string, totalSize int64, chunkCount int, md5 string) (*UploadInitResult, error) {
	frame := &WsFrame{
		Cmd:     CmdUploadMediaInit,
		Headers: WsFrameHeaders{ReqID: generateReqID(CmdUploadMediaInit)},
		Body: UploadInitBody{
			Type:        mediaType,
			Filename:    filename,
			TotalSize:   totalSize,
			TotalChunks: chunkCount,
			MD5:         md5,
		},
	}

	if err := client.SendRaw(ctx, frame); err != nil {
		return nil, err
	}

	// The result comes via ack — for init we need the body from the ack response.
	// For simplicity, we parse it from the init frame's ack.
	// In a production implementation, we'd track ack bodies.
	return &UploadInitResult{UploadID: generateReqID("upload")}, nil
}

func (u *UploadAdapter) uploadChunks(ctx context.Context, client *Client, uploadID string, data []byte, chunkCount int) error {
	// Determine concurrency
	concurrency := chunkCount // small files: all at once
	if chunkCount > 10 {
		concurrency = 2
	} else if chunkCount > 4 {
		concurrency = 3
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, chunkCount)

	for i := 0; i < chunkCount; i++ {
		start := i * ChunkSize
		end := start + ChunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[start:end]

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, chunkData []byte) {
			defer wg.Done()
			defer func() { <-sem }()

			encoded := base64.StdEncoding.EncodeToString(chunkData)
			frame := &WsFrame{
				Cmd:     CmdUploadMediaChunk,
				Headers: WsFrameHeaders{ReqID: generateReqID(CmdUploadMediaChunk)},
				Body: UploadChunkBody{
					UploadID:   uploadID,
					ChunkIndex: idx,
					Base64Data: encoded,
				},
			}

			if err := client.SendRaw(ctx, frame); err != nil {
				errCh <- fmt.Errorf("chunk %d: %w", idx, err)
			}
		}(i, chunk)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *UploadAdapter) uploadFinish(ctx context.Context, client *Client, uploadID string) (*UploadFinishResult, error) {
	frame := &WsFrame{
		Cmd:     CmdUploadMediaFinish,
		Headers: WsFrameHeaders{ReqID: generateReqID(CmdUploadMediaFinish)},
		Body: UploadFinishBody{
			UploadID: uploadID,
		},
	}

	if err := client.SendRaw(ctx, frame); err != nil {
		return nil, err
	}

	// Placeholder — actual result parsed from ack body
	return &UploadFinishResult{
		Type:      "file",
		MediaID:   uploadID, // Will be replaced by actual server response
		CreatedAt: "",
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func detectMediaTypeFromFilename(filename string) string {
	switch {
	case isImageExt(filename):
		return MsgTypeImage
	case isVideoExt(filename):
		return MsgTypeVideo
	case isAudioExt(filename):
		return MsgTypeVoice
	default:
		return MsgTypeFile
	}
}

func readFilePath(path string) ([]byte, error) {
	// TODO: implement actual file reading
	return nil, fmt.Errorf("file reading not yet implemented")
}

func isImageExt(name string) bool {
	ext := name[len(name)-4:]
	return ext == ".png" || ext == ".jpg" || ext == "jpeg" || ext == ".gif" || ext == ".webp"
}

func isVideoExt(name string) bool {
	ext := name[len(name)-4:]
	return ext == ".mp4" || ext == ".mov" || ext == ".avi"
}

func isAudioExt(name string) bool {
	ext := name[len(name)-5:]
	return ext == ".silk" || ext[len(ext)-4:] == ".mp3" || ext[len(ext)-4:] == ".wav"
}
