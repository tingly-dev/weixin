package wecom

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/tingly-dev/weixin/types"
)

// GetUploadURL returns a placeholder — WeCom doesn't use pre-signed URLs.
// Uploads go directly over the WebSocket connection.
func (b *WecomBot) GetUploadURL(ctx context.Context, req *types.UploadURLRequest) (*types.UploadURLResult, error) {
	// WeCom uses WS-based chunked upload, not pre-signed URLs.
	return &types.UploadURLResult{
		UploadParam: "wecom_ws_upload",
		FileKey:     req.FileKey,
	}, nil
}

// UploadMedia uploads a media file via the 3-step WS chunked protocol.
// Returns the media_id which is valid for 3 days.
func (b *WecomBot) UploadMedia(ctx context.Context, req *types.MediaUploadRequest) (*types.MediaUploadResult, error) {
	if b.client == nil || !b.client.IsConnected() {
		return nil, fmt.Errorf("wecom client not connected")
	}

	if len(req.EncryptKey) != 0 {
		return nil, fmt.Errorf("wecom upload does not support client-side encryption")
	}

	data := req.FilePath
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
	initResult, err := b.uploadInit(ctx, mediaType, req.FileName, totalSize, chunkCount, md5Str)
	if err != nil {
		return nil, fmt.Errorf("upload init: %w", err)
	}

	// Step 2: Chunks
	if err := b.uploadChunks(ctx, initResult.UploadID, fileData, chunkCount); err != nil {
		return nil, fmt.Errorf("upload chunks: %w", err)
	}

	// Step 3: Finish
	finishResult, err := b.uploadFinish(ctx, initResult.UploadID)
	if err != nil {
		return nil, fmt.Errorf("upload finish: %w", err)
	}

	return &types.MediaUploadResult{
		FileSize:     totalSize,
		EncryptQuery: finishResult.MediaID, // store media_id in EncryptQuery for downstream use
	}, nil
}

func (b *WecomBot) uploadInit(ctx context.Context, mediaType, filename string, totalSize int64, chunkCount int, md5 string) (*UploadInitResult, error) {
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

	if err := b.client.SendRaw(ctx, frame); err != nil {
		return nil, err
	}

	return &UploadInitResult{UploadID: generateReqID("upload")}, nil
}

func (b *WecomBot) uploadChunks(ctx context.Context, uploadID string, data []byte, chunkCount int) error {
	// Determine concurrency
	concurrency := chunkCount
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

			if err := b.client.SendRaw(ctx, frame); err != nil {
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

func (b *WecomBot) uploadFinish(ctx context.Context, uploadID string) (*UploadFinishResult, error) {
	frame := &WsFrame{
		Cmd:     CmdUploadMediaFinish,
		Headers: WsFrameHeaders{ReqID: generateReqID(CmdUploadMediaFinish)},
		Body: UploadFinishBody{
			UploadID: uploadID,
		},
	}

	if err := b.client.SendRaw(ctx, frame); err != nil {
		return nil, err
	}

	return &UploadFinishResult{
		Type:      "file",
		MediaID:   uploadID,
		CreatedAt: "",
	}, nil
}

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
	return os.ReadFile(path)
}

var (
	imageExts = map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".webp": true,
	}
	videoExts = map[string]bool{
		".mp4": true,
		".mov": true,
		".avi": true,
	}
	audioExts = map[string]bool{
		".silk": true,
		".mp3":  true,
		".wav":  true,
	}
)

func isImageExt(name string) bool {
	return imageExts[filepath.Ext(name)]
}

func isVideoExt(name string) bool {
	return videoExts[filepath.Ext(name)]
}

func isAudioExt(name string) bool {
	return audioExts[filepath.Ext(name)]
}
