package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": `{"one_line":"Mock LLM summarized the ASR transcript.","outline":["ASR returned transcript text","LLM produced structured fields"],"quotes":["Mock LLM summarized the ASR transcript."],"viewpoints":["The MVP can call OpenAI-compatible local services."],"analysis":"This response proves the summarizer adapter is wired into the pipeline."}`,
					},
				},
			},
		})
	})

	log.Println("mock LLM server listening on :18082")
	log.Fatal(http.ListenAndServe(":18082", nil))
}
