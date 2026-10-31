package services

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/akinalp/mqvi/config"
	"github.com/akinalp/mqvi/pkg/antivirus"
	"github.com/akinalp/mqvi/pkg/files"
	"github.com/akinalp/mqvi/repository"
)

type fakeMultipartFile struct {
	*bytes.Reader
}

func (f fakeMultipartFile) Close() error { return nil }

type fakeScanner struct {
	result antivirus.Result
	calls  int
}

func (s *fakeScanner) Scan(context.Context, string) antivirus.Result {
	s.calls++
	return s.result
}

type fakeScanCache struct {
	entries map[string]*repository.ScanHashCacheEntry
	upserts []string
}

func (c *fakeScanCache) Get(_ context.Context, sha256 string) (*repository.ScanHashCacheEntry, error) {
	if c.entries == nil {
		return nil, nil
	}
	return c.entries[sha256], nil
}

func (c *fakeScanCache) Upsert(_ context.Context, sha256, status string, signature *string) error {
	c.upserts = append(c.upserts, status)
	if c.entries == nil {
		c.entries = map[string]*repository.ScanHashCacheEntry{}
	}
	c.entries[sha256] = &repository.ScanHashCacheEntry{
		SHA256:    sha256,
		Status:    status,
		Signature: signature,
		ScannedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return nil
}

func (c *fakeScanCache) DeleteBefore(context.Context, string, string) (int, error) {
	return 0, nil
}

func newPipelineForTest(t *testing.T, scanner *fakeScanner, cache *fakeScanCache, cfg config.AntivirusConfig) UploadPipeline {
	t.Helper()
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = 1
	}
	if cfg.UnavailablePolicy == "" {
		cfg.UnavailablePolicy = antivirusPolicyAllowWithLog
	}
	if cfg.TooLargePolicy == "" {
		cfg.TooLargePolicy = antivirusPolicySkipWithLog
	}
	if cfg.CleanCacheTTLHours == 0 {
		cfg.CleanCacheTTLHours = 24
	}
	if cfg.InfectedCacheTTLDays == 0 {
		cfg.InfectedCacheTTLDays = 30
	}
	if cfg.CircuitFailureThreshold == 0 {
		cfg.CircuitFailureThreshold = 3
	}
	if cfg.CircuitWindowSeconds == 0 {
		cfg.CircuitWindowSeconds = 30
	}
	if cfg.CircuitOpenSeconds == 0 {
		cfg.CircuitOpenSeconds = 10
	}
	return NewUploadPipeline(files.NewLocator(t.TempDir(), ""), scanner, cache, nil, cfg)
}

func storeTestFile(t *testing.T, pipeline UploadPipeline, name string, body []byte, size int64) (*StoredUpload, error) {
	t.Helper()
	header := &multipart.FileHeader{
		Filename: name,
		Size:     size,
		Header:   textproto.MIMEHeader{"Content-Type": {"text/plain"}},
	}
	return pipeline.Store(context.Background(), files.KindMessage, "msg1", fakeMultipartFile{bytes.NewReader(body)}, header, 1024*1024)
}

func finalPath(t *testing.T, pipeline UploadPipeline, stored *StoredUpload) string {
	t.Helper()
	p := pipeline.(*uploadPipeline)
	path, err := p.locator.DiskPath(files.KindMessage, "msg1", filepath.Base(stored.RelativeURL))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUploadPipelineCleanFileMovesToFinalStorage(t *testing.T) {
	scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusClean}}
	cache := &fakeScanCache{}
	pipeline := newPipelineForTest(t, scanner, cache, config.AntivirusConfig{Enabled: true})

	stored, err := storeTestFile(t, pipeline, "hello.txt", []byte("hello"), 5)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	if scanner.calls != 1 {
		t.Fatalf("scanner calls = %d, want 1", scanner.calls)
	}
	if _, err := os.Stat(finalPath(t, pipeline, stored)); err != nil {
		t.Fatalf("final file missing: %v", err)
	}
	if len(cache.upserts) != 1 || cache.upserts[0] != string(antivirus.StatusClean) {
		t.Fatalf("cache upserts = %#v, want clean", cache.upserts)
	}
}

func TestUploadPipelineInfectedFileIsRejected(t *testing.T) {
	scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusInfected, Signature: "Eicar-Test-Signature"}}
	pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true})

	if _, err := storeTestFile(t, pipeline, "bad.txt", []byte("bad"), 3); err == nil {
		t.Fatal("Store succeeded for infected file")
	}
}

func TestUploadPipelineUnavailablePolicy(t *testing.T) {
	t.Run("allow", func(t *testing.T) {
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusUnavailable}}
		pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true, UnavailablePolicy: antivirusPolicyAllowWithLog})
		if _, err := storeTestFile(t, pipeline, "file.txt", []byte("ok"), 2); err != nil {
			t.Fatalf("Store returned error: %v", err)
		}
	})

	t.Run("reject", func(t *testing.T) {
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusUnavailable}}
		pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true, UnavailablePolicy: antivirusPolicyReject})
		if _, err := storeTestFile(t, pipeline, "file.txt", []byte("ok"), 2); err == nil {
			t.Fatal("Store succeeded while scanner was unavailable in reject policy")
		}
	})
}

func TestUploadPipelineTooLargePolicy(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusClean}}
		pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true, MaxScanSizeBytes: 1, TooLargePolicy: antivirusPolicySkipWithLog})
		if _, err := storeTestFile(t, pipeline, "big.txt", []byte("big"), 3); err != nil {
			t.Fatalf("Store returned error: %v", err)
		}
		if scanner.calls != 0 {
			t.Fatalf("scanner calls = %d, want 0", scanner.calls)
		}
	})

	t.Run("reject", func(t *testing.T) {
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusClean}}
		pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true, MaxScanSizeBytes: 1, TooLargePolicy: antivirusPolicyReject})
		if _, err := storeTestFile(t, pipeline, "big.txt", []byte("big"), 3); err == nil {
			t.Fatal("Store succeeded for too-large file in reject policy")
		}
		if scanner.calls != 0 {
			t.Fatalf("scanner calls = %d, want 0", scanner.calls)
		}
	})
}

func TestUploadPipelineUploadMaxUsesWrittenSize(t *testing.T) {
	scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusClean}}
	pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true})
	header := &multipart.FileHeader{
		Filename: "big.txt",
		Size:     1,
		Header:   textproto.MIMEHeader{"Content-Type": {"text/plain"}},
	}

	_, err := pipeline.Store(context.Background(), files.KindMessage, "msg1", fakeMultipartFile{bytes.NewReader([]byte("too big"))}, header, 3)
	if err == nil {
		t.Fatal("Store succeeded even though written size exceeded max")
	}
	if scanner.calls != 0 {
		t.Fatalf("scanner calls = %d, want 0", scanner.calls)
	}
}

func TestUploadPipelineScannerTooLargeResultUsesTooLargePolicy(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusTooLarge}}
		pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true, TooLargePolicy: antivirusPolicySkipWithLog})
		if _, err := storeTestFile(t, pipeline, "file.txt", []byte("ok"), 2); err != nil {
			t.Fatalf("Store returned error: %v", err)
		}
	})

	t.Run("reject", func(t *testing.T) {
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusTooLarge}}
		pipeline := newPipelineForTest(t, scanner, &fakeScanCache{}, config.AntivirusConfig{Enabled: true, TooLargePolicy: antivirusPolicyReject})
		if _, err := storeTestFile(t, pipeline, "file.txt", []byte("ok"), 2); err == nil {
			t.Fatal("Store succeeded after scanner size-limit result in reject policy")
		}
	})
}

func TestUploadPipelineHashCache(t *testing.T) {
	digest := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	t.Run("clean skips scanner", func(t *testing.T) {
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusInfected}}
		cache := &fakeScanCache{entries: map[string]*repository.ScanHashCacheEntry{
			digest: {SHA256: digest, Status: string(antivirus.StatusClean), ScannedAt: time.Now().UTC().Format(time.RFC3339)},
		}}
		pipeline := newPipelineForTest(t, scanner, cache, config.AntivirusConfig{Enabled: true})
		if _, err := storeTestFile(t, pipeline, "hello.txt", []byte("hello"), 5); err != nil {
			t.Fatalf("Store returned error: %v", err)
		}
		if scanner.calls != 0 {
			t.Fatalf("scanner calls = %d, want 0", scanner.calls)
		}
	})

	t.Run("infected rejects without scanner", func(t *testing.T) {
		sig := "cached"
		scanner := &fakeScanner{result: antivirus.Result{Status: antivirus.StatusClean}}
		cache := &fakeScanCache{entries: map[string]*repository.ScanHashCacheEntry{
			digest: {SHA256: digest, Status: string(antivirus.StatusInfected), Signature: &sig, ScannedAt: time.Now().UTC().Format(time.RFC3339)},
		}}
		pipeline := newPipelineForTest(t, scanner, cache, config.AntivirusConfig{Enabled: true})
		if _, err := storeTestFile(t, pipeline, "hello.txt", []byte("hello"), 5); err == nil {
			t.Fatal("Store succeeded for infected cache hit")
		}
		if scanner.calls != 0 {
			t.Fatalf("scanner calls = %d, want 0", scanner.calls)
		}
	})
}
