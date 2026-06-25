package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/v1/audio/transcriptions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, _, err := r.FormFile("file"); err != nil {
			http.Error(w, "file is required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text": "This transcript came from the local OpenAI-compatible ASR API mock.",
			"segments": []map[string]any{
				{
					"start": 0,
					"end":   4.2,
					"text":  "This transcript came from the local OpenAI-compatible ASR API mock.",
				},
			},
		})
	})

	log.Println("mock ASR server listening on :18081")
	log.Fatal(http.ListenAndServe(":18081", nil))
}
