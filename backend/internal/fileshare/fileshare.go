package fileshare

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/drmcs/backend/internal/crypto"
	"github.com/drmcs/backend/internal/models"
	"github.com/drmcs/backend/internal/storage"
)

const (
	chunkSize       = 256 * 1024 // 256KB per chunk
	defaultExpiry   = 2 * time.Hour
	transferDir     = "./transfers"
	maxFileSize     = 50 * 1024 * 1024 // 50MB
	cleanupInterval = 15 * time.Minute
)

// Transfer manages temporary file sharing between nodes
type Transfer struct {
	nodeID     string
	store      *storage.SQLiteStore
	privateKey ed25519.PrivateKey
	mu         sync.RWMutex
	stopCh     chan struct{}
	activeDir  string
}

// NewTransfer creates a new file transfer manager
func NewTransfer(nodeID string, store *storage.SQLiteStore, privKey ed25519.PrivateKey) *Transfer {
	return &Transfer{
		nodeID:     nodeID,
		store:      store,
		privateKey: privKey,
		stopCh:     make(chan struct{}),
		activeDir:  transferDir,
	}
}

// Start initializes the file transfer service
func (t *Transfer) Start(port int) {
	// Create transfer directory
	if err := os.MkdirAll(t.activeDir, 0755); err != nil {
		log.Printf("Failed to create transfer directory: %v", err)
	}

	go t.cleanupLoop()
	log.Printf("File transfer service started on port %d", port)
}

// Stop halts the file transfer service
func (t *Transfer) Stop() {
	close(t.stopCh)
}

// UploadFile prepares a file for sharing on the mesh network
func (t *Transfer) UploadFile(filePath string, expiry time.Duration) (*models.FileTransfer, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check file size
	if fileInfo.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", fileInfo.Size(), maxFileSize)
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Compute checksum
	checksum := crypto.HashContent(content)

	// Calculate chunk count
	chunkCount := (len(content) + chunkSize - 1) / chunkSize

	// Generate file ID
	fileID := crypto.HashContent([]byte(fmt.Sprintf("%s:%s:%d", t.nodeID, fileInfo.Name(), time.Now().UnixNano())))

	if expiry <= 0 {
		expiry = defaultExpiry
	}

	fileTransfer := &models.FileTransfer{
		FileID:      fileID,
		SenderID:    t.nodeID,
		Filename:    fileInfo.Name(),
		FileSize:    fileInfo.Size(),
		ContentType: detectContentType(fileInfo.Name()),
		ChunkCount:  chunkCount,
		Checksum:    checksum,
		ExpiresAt:   time.Now().Add(expiry),
		Status:      models.StatusActive,
		Progress:    0.0,
	}

	// Save file transfer record
	if err := t.store.SaveFileTransfer(fileTransfer); err != nil {
		return nil, fmt.Errorf("failed to save file transfer: %w", err)
	}

	// Save chunks to disk
	chunksDir := filepath.Join(t.activeDir, fileID)
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunks directory: %w", err)
	}

	for i := 0; i < chunkCount; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}

		chunk := &models.FileChunk{
			ChunkID:  i,
			FileID:   fileID,
			Data:     content[start:end],
			Checksum: crypto.HashContent(content[start:end]),
			Size:     end - start,
		}

		chunkPath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", i))
		chunkData, _ := json.Marshal(chunk)
		if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write chunk %d: %w", i, err)
		}
	}

	fileTransfer.Progress = 100.0
	t.store.SaveFileTransfer(fileTransfer)

	log.Printf("File uploaded: %s (%d bytes, %d chunks)", fileInfo.Name(), fileInfo.Size(), chunkCount)
	return fileTransfer, nil
}

// DownloadFile assembles file chunks and saves the complete file
func (t *Transfer) DownloadFile(fileID, outputPath string) error {
	// Get file transfer record
	ft, err := t.store.GetFileTransfer(fileID)
	if err != nil {
		return fmt.Errorf("file transfer not found: %w", err)
	}

	if ft.Status != models.StatusActive {
		return fmt.Errorf("file transfer is not active (status: %s)", ft.Status)
	}

	// Read and assemble chunks
	chunksDir := filepath.Join(t.activeDir, fileID)
	var fileBuffer bytes.Buffer

	for i := 0; i < ft.ChunkCount; i++ {
		chunkPath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read chunk %d: %w", i, err)
		}

		var chunk models.FileChunk
		if err := json.Unmarshal(chunkData, &chunk); err != nil {
			return fmt.Errorf("failed to decode chunk %d: %w", i, err)
		}

		// Verify chunk integrity
		if crypto.HashContent(chunk.Data) != chunk.Checksum {
			return fmt.Errorf("chunk %d checksum mismatch", i)
		}

		fileBuffer.Write(chunk.Data)
	}

	// Verify complete file integrity
	if crypto.HashContent(fileBuffer.Bytes()) != ft.Checksum {
		return fmt.Errorf("file checksum mismatch")
	}

	// Write output file
	if err := os.WriteFile(outputPath, fileBuffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Update status
	ft.Status = "completed"
	t.store.SaveFileTransfer(ft)

	log.Printf("File downloaded: %s -> %s (%d bytes)", ft.Filename, outputPath, ft.FileSize)
	return nil
}

// ListAvailableFiles returns all active file transfers
func (t *Transfer) ListAvailableFiles() ([]*models.FileTransfer, error) {
	// In production, query all peers for their file lists
	// For now, return local files
	return nil, nil
}

// GetTransferStatus returns the status of a file transfer
func (t *Transfer) GetTransferStatus(fileID string) (*models.FileTransfer, error) {
	return t.store.GetFileTransfer(fileID)
}

// DeleteFile removes a file transfer and its chunks
func (t *Transfer) DeleteFile(fileID string) error {
	// Remove chunks directory
	chunksDir := filepath.Join(t.activeDir, fileID)
	if err := os.RemoveAll(chunksDir); err != nil {
		return fmt.Errorf("failed to remove chunks: %w", err)
	}

	// Update status
	ft, err := t.store.GetFileTransfer(fileID)
	if err == nil {
		ft.Status = models.StatusExpired
		t.store.SaveFileTransfer(ft)
	}

	return nil
}

func (t *Transfer) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.cleanupExpiredFiles()
		}
	}
}

func (t *Transfer) cleanupExpiredFiles() {
	entries, err := os.ReadDir(t.activeDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ft, err := t.store.GetFileTransfer(entry.Name())
		if err != nil {
			continue
		}

		if time.Now().After(ft.ExpiresAt) {
			os.RemoveAll(filepath.Join(t.activeDir, entry.Name()))
			ft.Status = models.StatusExpired
			t.store.SaveFileTransfer(ft)
			log.Printf("Expired file cleaned up: %s", ft.Filename)
		}
	}
}

func detectContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".doc", ".docx":
		return "application/msword"
	case ".txt":
		return "text/plain"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}