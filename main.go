// webhook-relay forwards git webhook payloads to configurable HTTP targets.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type target struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type rule struct {
	Repo   string   `json:"repo"`
	Events []string `json:"events"`
	Target target   `json:"target"`
}

type config struct {
	Listen string `json:"listen"`
	Rules  []rule `json:"rules"`
}

func loadConfig(path string) (*config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	return &cfg, nil
}

func (r rule) matches(event, repo string) bool {
	if !strings.EqualFold(r.Repo, repo) {
		return false
	}
	for _, e := range r.Events {
		if e == "*" || strings.EqualFold(e, event) {
			return true
		}
	}
	return false
}

func (t target) deliver(event string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, t.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Relay-Event", event)
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("target %s answered %s", t.URL, resp.Status)
	}
	return nil
}

func handler(cfg *config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprintf(w, "ok rules=%d
", len(cfg.Rules))
			return
		}
		event := r.Header.Get("X-GitHub-Event")
		if event == "" {
			event = r.Header.Get("X-Gitea-Event")
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		var payload struct {
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		}
		_ = json.Unmarshal(body, &payload)
		matched := 0
		for _, rl := range cfg.Rules {
			if rl.matches(event, payload.Repository.FullName) {
				if err := rl.Target.deliver(event, body); err != nil {
					log.Printf("deliver %s -> %s: %v", event, rl.Target.URL, err)
				}
				matched++
			}
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "relayed=%d
", matched)
	}
}

func main() {
	cfgPath := flag.String("config", "config.json", "path to the relay config")
	flag.Parse()
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", handler(cfg))
	log.Printf("webhook-relay listening on %s with %d rules", cfg.Listen, len(cfg.Rules))
	log.Fatal(http.ListenAndServe(cfg.Listen, nil))
}
