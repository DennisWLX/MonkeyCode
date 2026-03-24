package file

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	mgr := NewManager(logger)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.logger == nil {
		t.Error("logger is nil")
	}
}

func TestExists(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if !mgr.Exists(tmpFile.Name()) {
		t.Errorf("Exists(%s) = false; want true", tmpFile.Name())
	}

	if !mgr.Exists(tmpDir) {
		t.Errorf("Exists(%s) = false; want true", tmpDir)
	}

	if mgr.Exists("/non/existent/path") {
		t.Error("Exists(non-existent) = true; want false")
	}
}

func TestIsDir(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if !mgr.IsDir(tmpDir) {
		t.Errorf("IsDir(%s) = false; want true", tmpDir)
	}

	if mgr.IsDir(tmpFile.Name()) {
		t.Errorf("IsDir(%s) = true; want false", tmpFile.Name())
	}

	if mgr.IsDir("/non/existent/path") {
		t.Error("IsDir(non-existent) = true; want false")
	}
}

func TestSize(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("test content")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	size, err := mgr.Size(tmpFile.Name())
	if err != nil {
		t.Errorf("Size() error = %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("Size() = %d; want %d", size, len(content))
	}

	_, err = mgr.Size("/non/existent/file")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestCleanPath(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "/path/to/file",
			expected: "path/to/file",
		},
		{
			name:     "absolute path",
			input:    "/abs/path",
			expected: "abs/path",
		},
		{
			name:     "clean path",
			input:    "clean/path",
			expected: "clean/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mgr.CleanPath(tt.input)
			if result != tt.expected {
				t.Errorf("CleanPath(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTarGZ(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tmpDir, err := os.MkdirTemp("", "tartest-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tmpFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(archivePath)

	ctx := context.Background()
	err = mgr.TarGZ(ctx, tmpDir, archivePath)
	if err != nil {
		t.Errorf("TarGZ() error = %v", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		t.Errorf("archive file not created: %v", err)
	}

	if info.Size() == 0 {
		t.Error("archive file is empty")
	}
}

func TestUntarGZ(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tmpDir, err := os.MkdirTemp("", "tartest-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tmpFile, err := os.CreateTemp("", "test-*.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(archivePath)

	createArchive := func(srcDir, destFile string) error {
		outFile, err := os.Create(destFile)
		if err != nil {
			return err
		}
		defer outFile.Close()

		gzWriter := gzip.NewWriter(outFile)
		defer gzWriter.Close()

		tarWriter := tar.NewWriter(gzWriter)
		defer tarWriter.Close()

		return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, relPath)
			if err != nil {
				return err
			}
			header.Name = relPath

			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				if _, err := io.Copy(tarWriter, file); err != nil {
					return err
				}
			}

			return nil
		})
	}

	err = createArchive(tmpDir, archivePath)
	if err != nil {
		t.Fatal(err)
	}

	destDir, err := os.MkdirTemp("", "tartest-dest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(destDir)

	ctx := context.Background()
	err = mgr.UntarGZ(ctx, archivePath, destDir)
	if err != nil {
		t.Errorf("UntarGZ() error = %v", err)
	}

	file1Path := filepath.Join(destDir, "file1.txt")
	if _, err := os.Stat(file1Path); os.IsNotExist(err) {
		t.Error("file1.txt not extracted")
	}

	content, err := os.ReadFile(file1Path)
	if err != nil {
		t.Errorf("failed to read extracted file: %v", err)
	}

	if string(content) != "content1" {
		t.Errorf("file content = %s; want %s", string(content), "content1")
	}
}

func TestTarGZInvalidSource(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	ctx := context.Background()
	err := mgr.TarGZ(ctx, "/non/existent/dir", "/tmp/archive.tar.gz")
	if err == nil {
		t.Error("expected error for non-existent source")
	}
}

func TestTarGZSourceNotDirectory(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	ctx := context.Background()
	err = mgr.TarGZ(ctx, tmpFile.Name(), "/tmp/archive.tar.gz")
	if err == nil {
		t.Error("expected error when source is not a directory")
	}
}

func TestUntarGZInvalidSource(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	tmpDir, err := os.MkdirTemp("", "test-dest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	err = mgr.UntarGZ(ctx, "/non/existent/archive.tar.gz", tmpDir)
	if err == nil {
		t.Error("expected error for non-existent archive")
	}
}

func TestUploadValidation(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	ctx := context.Background()

	tests := []struct {
		name        string
		containerID string
		destPath    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty container ID",
			containerID: "",
			destPath:    "/dest",
			wantErr:     true,
			errContains: "container_id is required",
		},
		{
			name:        "empty dest path",
			containerID: "container-123",
			destPath:    "",
			wantErr:     true,
			errContains: "destination path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Upload(ctx, tt.containerID, tt.destPath, nil, 0644)
			if !tt.wantErr {
				return
			}
			if err == nil {
				t.Error("expected error")
				return
			}
			if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v; want to contain %s", err, tt.errContains)
			}
		})
	}
}

func TestDownloadValidation(t *testing.T) {
	mgr := NewManager(slog.New(slog.DiscardHandler))

	ctx := context.Background()

	tests := []struct {
		name        string
		containerID string
		srcPath     string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty container ID",
			containerID: "",
			srcPath:     "/src",
			wantErr:     true,
			errContains: "container_id is required",
		},
		{
			name:        "empty src path",
			containerID: "container-123",
			srcPath:     "",
			wantErr:     true,
			errContains: "source path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.Download(ctx, tt.containerID, tt.srcPath, nil)
			if !tt.wantErr {
				return
			}
			if err == nil {
				t.Error("expected error")
				return
			}
			if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v; want to contain %s", err, tt.errContains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
