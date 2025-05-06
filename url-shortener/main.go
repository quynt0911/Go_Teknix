package main

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	urlStore    = make(map[string]string) // Lưu trữ mã rút gọn và URL gốc
	visitCount  = make(map[string]int)    // Đếm số lượt truy cập theo mã
	mutex       = sync.Mutex{}
	shortLength = 6
	charset     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Xử lý các đường dẫn
	http.HandleFunc("/", indexHandler)             // Trang chính với form
	http.HandleFunc("/shorten", shortenHandler)    // Xử lý rút gọn URL
	http.HandleFunc("/shorturl/", redirectHandler) // Chuyển hướng từ mã rút gọn

	// Cung cấp file tĩnh cho giao diện
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server đang chạy tại http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Trang index.html với form nhập URL
func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Lấy shortURL từ cookie nếu có
	cookie, err := r.Cookie("shortURL")
	shortURL := ""
	visits := 0

	if err == nil {
		shortURL = cookie.Value
		mutex.Lock()
		visits = visitCount[strings.TrimPrefix(shortURL, "http://localhost:8080/shorturl/")]
		mutex.Unlock()
	}

	data := struct {
		ShortURL  string
		Visits    int
		Shortened bool
	}{
		ShortURL:  shortURL,
		Visits:    visits,
		Shortened: shortURL != "",
	}

	tmpl.Execute(w, data)
}

// Tạo URL rút gọn
func shortenHandler(w http.ResponseWriter, r *http.Request) {
	originalURL := r.URL.Query().Get("url")
	if originalURL == "" {
		http.Error(w, "Vui lòng nhập URL hợp lệ.", http.StatusBadRequest)
		return
	}

	// Kiểm tra và thêm "https://www." nếu URL không có tiền tố
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		originalURL = "https://www." + originalURL
	}

	// Tạo mã rút gọn
	shortCode := generateShortURL()
	shortURL := "http://localhost:8080/shorturl/" + shortCode

	// Lưu trữ URL và khởi tạo số lượt truy cập
	mutex.Lock()
	urlStore[shortCode] = originalURL
	visitCount[shortCode] = 0
	mutex.Unlock()

	// Lưu shortURL vào cookie để truy cập từ trang index
	http.SetCookie(w, &http.Cookie{
		Name:  "shortURL",
		Value: shortURL,
		Path:  "/",
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

// Chuyển hướng từ mã rút gọn đến URL gốc và đếm lượt truy cập
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := strings.TrimPrefix(r.URL.Path, "/shorturl/")

	mutex.Lock()
	originalURL, exists := urlStore[shortCode]
	if exists {
		visitCount[shortCode]++
	}
	mutex.Unlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

// Tạo mã rút gọn ngẫu nhiên
func generateShortURL() string {
	b := make([]byte, shortLength)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
