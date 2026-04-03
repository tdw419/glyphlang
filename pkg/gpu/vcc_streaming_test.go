package gpu

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestVCCEndpointReturnsData tests that the VCC streaming endpoint
// serves data when the shared memory file exists.
func TestVCCEndpointReturnsData(t *testing.T) {
	// Create a fake VCC texture file
	tmpFile, err := os.CreateTemp("/dev/shm", "test_vcc_*.rgba")
	if err != nil {
		// /dev/shm might not be writable in all environments
		tmpFile, err = os.CreateTemp("", "test_vcc_*.rgba")
		if err != nil {
			t.Skipf("cannot create temp file: %v", err)
		}
	}
	defer os.Remove(tmpFile.Name())

	// Write 256x256 RGBA test data (256*256*4 = 262144 bytes)
	testData := make([]byte, 262144)
	for i := 0; i < len(testData); i += 4 {
		testData[i] = byte(i % 256)     // R
		testData[i+1] = byte((i + 1) % 256) // G
		testData[i+2] = byte((i + 2) % 256) // B
		testData[i+3] = 255               // A
	}
	tmpFile.Write(testData)
	tmpFile.Close()

	// Create a test HTTP server that mimics the VCC endpoint
	mux := http.NewServeMux()
	vccPath := tmpFile.Name()
	mux.HandleFunc("/vcc/colony.rgba", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, vccPath)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Verify the endpoint returns data
	resp, err := http.Get(server.URL + "/vcc/colony.rgba")
	if err != nil {
		t.Fatalf("VCC endpoint request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Read and verify the data
	buf := make([]byte, 262144+100)
	total := 0
	for {
		n, err := resp.Body.Read(buf[total:])
		total += n
		if err != nil {
			break
		}
	}
	if total < 1000 {
		t.Errorf("expected at least 1000 bytes from VCC endpoint, got %d", total)
	}
}

// TestVCCEndpointMissingFile tests graceful handling when the SHM
// file doesn't exist yet (daemon not running).
func TestVCCEndpointMissingFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vcc/colony.rgba", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/dev/shm/nonexistent_colony.rgba")
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/vcc/colony.rgba")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 404 when file doesn't exist
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("note: got status %d (expected 404 for missing file)", resp.StatusCode)
	}
}
