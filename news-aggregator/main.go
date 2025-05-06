package main

import (
	"context"
	"encoding/json"
	"fmt"

	// "html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/segmentio/kafka-go"
)

// Cấu trúc bài viết
type Article struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Source string `json:"source"`
}

type RateLimiter struct {
	requests map[string]int
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]int),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rl.mu.Lock()
		defer rl.mu.Unlock()

		ip := strings.Split(r.RemoteAddr, ":")[0]
		if rl.requests[ip] >= rl.limit {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		rl.requests[ip]++
		time.AfterFunc(rl.window, func() {
			rl.mu.Lock()
			rl.requests[ip]--
			rl.mu.Unlock()
		})

		next.ServeHTTP(w, r)
	})
}

func scrapeSource(url string, wg *sync.WaitGroup, ch chan<- Article) {
	defer wg.Done()

	res, err := http.Get(url)
	if err != nil {
		log.Println("Lỗi khi truy cập:", url, err)
		return
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Println("Lỗi đọc nội dung:", err)
		return
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Text())
		link, exists := s.Attr("href")

		if exists && title != "" {
			if !strings.HasPrefix(link, "/") && strings.Contains(link, "dantri.com.vn") {
				ch <- Article{
					Title:  title,
					URL:    link,
					Source: url,
				}
			}
		}
	})
}

func ScrapeNews(sources []string) []Article {
	var wg sync.WaitGroup
	ch := make(chan Article, 100)

	for _, source := range sources {
		wg.Add(1)
		go scrapeSource(source, &wg, ch)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var articles []Article
	for article := range ch {
		articles = append(articles, article)
	}

	return articles
}

func GetLatestArticles(w http.ResponseWriter, r *http.Request) {
	sources := []string{"https://vnexpress.net/", "https://dantri.com.vn/"}
	articles := ScrapeNews(sources)

	if len(articles) == 0 {
		log.Println("Không có bài viết nào")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

func publishNewsToKafka(article Article) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9093"},
		Topic:   "news_topic",
	})
	defer writer.Close()

	data, err := json.Marshal(article)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Value: data,
	}

	return writer.WriteMessages(context.Background(), msg)
}

func PublishNews(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		News Article `json:"news"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.News.Title == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	err = publishNewsToKafka(payload.News)
	if err != nil {
		http.Error(w, fmt.Sprintf("Kafka publish error: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "News published successfully")
}

func SetupRoutes() {
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./public"))))

	http.HandleFunc("/articles", GetLatestArticles)
	http.HandleFunc("/publish", PublishNews)
}

func main() {
	rateLimiter := NewRateLimiter(100, 1*time.Minute)
	SetupRoutes()

	http.Handle("/", rateLimiter.Limit(http.DefaultServeMux))

	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
