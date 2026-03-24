package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// HN top stories API — no auth, always available
const hnTopStoriesURL = "https://hacker-news.firebaseio.com/v0/topstories.json"
const hnItemURL = "https://hacker-news.firebaseio.com/v0/item/%d.json"

type Story struct {
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"`
	Type        string `json:"type"`
}

type metrics struct {
	mu              sync.RWMutex
	topStoryScore   float64
	topStoryComments float64
	scrapeCount     float64
	scrapeErrors    float64
	lastScrapeTime  float64 // unix timestamp
}

var m = &metrics{}

func scrape() {
	resp, err := http.Get(hnTopStoriesURL)
	if err != nil {
		m.mu.Lock()
		m.scrapeErrors++
		m.mu.Unlock()
		log.Printf("error fetching top stories: %v", err)
		return
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		m.mu.Lock()
		m.scrapeErrors++
		m.mu.Unlock()
		log.Printf("error decoding top stories: %v", err)
		return
	}

	if len(ids) == 0 {
		return
	}

	// Fetch only the top story to keep it simple
	itemResp, err := http.Get(fmt.Sprintf(hnItemURL, ids[0]))
	if err != nil {
		m.mu.Lock()
		m.scrapeErrors++
		m.mu.Unlock()
		log.Printf("error fetching top item: %v", err)
		return
	}
	defer itemResp.Body.Close()

	var story Story
	if err := json.NewDecoder(itemResp.Body).Decode(&story); err != nil {
		m.mu.Lock()
		m.scrapeErrors++
		m.mu.Unlock()
		log.Printf("error decoding story: %v", err)
		return
	}

	m.mu.Lock()
	m.topStoryScore = float64(story.Score)
	m.topStoryComments = float64(story.Descendants)
	m.scrapeCount++
	m.lastScrapeTime = float64(time.Now().Unix())
	m.mu.Unlock()

	log.Printf("scraped: score=%d comments=%d", story.Score, story.Descendants)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP hn_top_story_score Score of the current #1 Hacker News story\n")
	fmt.Fprintf(w, "# TYPE hn_top_story_score gauge\n")
	fmt.Fprintf(w, "hn_top_story_score %g\n\n", m.topStoryScore)

	fmt.Fprintf(w, "# HELP hn_top_story_comments Comment count of the current #1 Hacker News story\n")
	fmt.Fprintf(w, "# TYPE hn_top_story_comments gauge\n")
	fmt.Fprintf(w, "hn_top_story_comments %g\n\n", m.topStoryComments)

	fmt.Fprintf(w, "# HELP hn_scrape_total Total number of successful scrapes\n")
	fmt.Fprintf(w, "# TYPE hn_scrape_total counter\n")
	fmt.Fprintf(w, "hn_scrape_total %g\n\n", m.scrapeCount)

	fmt.Fprintf(w, "# HELP hn_scrape_errors_total Total number of failed scrapes\n")
	fmt.Fprintf(w, "# TYPE hn_scrape_errors_total counter\n")
	fmt.Fprintf(w, "hn_scrape_errors_total %g\n\n", m.scrapeErrors)

	fmt.Fprintf(w, "# HELP hn_last_scrape_timestamp_seconds Unix timestamp of the last successful scrape\n")
	fmt.Fprintf(w, "# TYPE hn_last_scrape_timestamp_seconds gauge\n")
	fmt.Fprintf(w, "hn_last_scrape_timestamp_seconds %g\n", m.lastScrapeTime)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func main() {
	// Initial scrape immediately, then every 60s
	scrape()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			scrape()
		}
	}()

	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/healthz", healthHandler)

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}