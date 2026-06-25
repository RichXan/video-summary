package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"video-summary-mvp/internal/domain"
)

func TestHTTPASRTranscriberCallsOpenAICompatibleEndpoint(t *testing.T) {
	t.Parallel()

	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(audioPath, []byte("fake-wav"), 0o600); err != nil {
		t.Fatal(err)
	}

	var sawFile bool
	var sawModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		sawModel = r.FormValue("model") == "whisper-1"
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		_ = file.Close()
		sawFile = true

		_ = json.NewEncoder(w).Encode(map[string]any{
			"text": "hello from open source asr",
			"segments": []map[string]any{
				{"start": 0, "end": 1.5, "text": "hello from open source asr"},
			},
		})
	}))
	defer server.Close()

	transcriber := HTTPASRTranscriber{
		BaseURL: server.URL,
		Model:   "whisper-1",
	}
	transcript, err := transcriber.Transcribe(context.Background(), domain.MediaAsset{AudioPath: audioPath})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}

	if !sawFile || !sawModel {
		t.Fatalf("request did not include file/model: file=%v model=%v", sawFile, sawModel)
	}
	if len(transcript) != 1 || transcript[0].Text != "hello from open source asr" {
		t.Fatalf("unexpected transcript: %#v", transcript)
	}
}

func TestHTTPASRTranscriberFallsBackToTextOnlyResponse(t *testing.T) {
	t.Parallel()

	audioPath := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(audioPath, []byte("fake-wav"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "plain transcript"})
	}))
	defer server.Close()

	transcriber := HTTPASRTranscriber{BaseURL: server.URL}
	transcript, err := transcriber.Transcribe(context.Background(), domain.MediaAsset{AudioPath: audioPath})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}

	if len(transcript) != 1 || transcript[0].Text != "plain transcript" {
		t.Fatalf("unexpected transcript: %#v", transcript)
	}
}
