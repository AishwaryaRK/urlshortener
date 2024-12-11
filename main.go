package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"sync"
)

type UrlShortener struct {
	urlMap         sync.Map
	invertedUrlMap sync.Map
	hostname       string
	len            int
}

func shortenUrl(w http.ResponseWriter, r *http.Request, urlShortener *UrlShortener) {
	w.Header().Set("Content-Type", "application/json")
	var data map[string]string
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "error reading http request body", http.StatusBadRequest)
		return
	}
	originalUrl, ok := data["url"]
	if !ok {
		http.Error(w, "http request body doesn't contain url param", http.StatusBadRequest)
		return
	}
	_, err = url.Parse(originalUrl)
	if err != nil {
		http.Error(w, "invalid url passed in request", http.StatusBadRequest)
		return
	}
	shortenedUrl, ok := urlShortener.urlMap.Load(originalUrl)
	if ok {
		u, ok := shortenedUrl.(string)
		if !ok {
			http.Error(w, "invalid shortened url found", http.StatusInternalServerError)
			return
		}
		response := map[string]string{"shortenedUrl": u}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&response)
		return
	}
	u, err := urlShortener.createShortenedUrl(originalUrl)
	if err != nil {
		http.Error(w, "error generating shortened url", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"shortenedUrl": u})
}

func (u *UrlShortener) createShortenedUrl(originalUrl string) (string, error) {
	s, err := u.generateRandomStr()
	if err != nil {
		return "", err
	}
	shortenedUrl := url.URL{
		Scheme: "https",
		Host:   u.hostname,
		Path:   s,
	}
	u.urlMap.Store(originalUrl, shortenedUrl.String())
	u.invertedUrlMap.Store(shortenedUrl.String(), originalUrl)
	return shortenedUrl.String(), nil
}

func (u *UrlShortener) generateRandomStr() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randonStr := make([]byte, u.len)
	for i := range randonStr {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		randonStr[i] = charset[n.Int64()]
	}
	return string(randonStr), nil
}

func getShortenedUrl(w http.ResponseWriter, r *http.Request, urlShortener *UrlShortener) {
	w.Header().Set("Content-Type", "application/json")
	query := r.URL.Query()
	shortenedUrl := query.Get("shortenedUrl")
	if shortenedUrl == "" {
		http.Error(w, "http request body doesn't contain shortenedUrl param", http.StatusBadRequest)
		return
	}
	parsedUrl, err := url.Parse(shortenedUrl)
	if err != nil {
		http.Error(w, "invalid url passed in request", http.StatusBadRequest)
		return
	}
	if parsedUrl.Host != urlShortener.hostname {
		http.Error(w, "invalid shortened url host passed in request", http.StatusBadRequest)
		return
	}
	originalUrl, ok := urlShortener.invertedUrlMap.Load(shortenedUrl)
	if !ok {
		http.Error(w, "invalid shortened url host passed in request", http.StatusBadRequest)
		return
	}
	originalUrlStr, ok := originalUrl.(string)
	if !ok {
		http.Error(w, "invalid original url found", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"originalUrl": originalUrlStr})
}

func main() {
	urlShortener := UrlShortener{hostname: "urlshortener.com", len: 5}
	http.HandleFunc("/shortenurl", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getShortenedUrl(w, r, &urlShortener)

		case http.MethodPost:
			shortenUrl(w, r, &urlShortener)

		default:
			http.Error(w, "Invalid http operation"+r.Method, http.StatusBadRequest)
		}
	})

	log.Println("Server starting on localhost 8080 ...")
	http.ListenAndServe("localhost:8080", nil)
}

// ➜  urlshortener go build .
// ➜  urlshortener ./urlshortener
// 2024/12/11 16:46:05 Server starting on localhost 8080 ...

// ➜  ~ curl -X POST -H "Content-Type: application/json" -d '{"url":"https://github.com/nebuxadnezzar/take-home-assignments/blob/main/esusu/README.md"}' http://localhost:8080/shortenurl
// {"shortenedUrl":"https://urlshortener.com/4ZedE"}
// ➜  ~ curl -X GET -H "Accept: application/json" "http://localhost:8080/shortenurl?shortenedUrl=https://urlshortener.com/4ZedE"
// {"originalUrl":"https://github.com/nebuxadnezzar/take-home-assignments/blob/main/esusu/README.md"}
// ➜  ~
