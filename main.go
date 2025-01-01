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

type Shortener interface {
	createShortenedUrl(string) (string, error)
	loadFromUrlMap(string) (any, bool)
	loadFromInvertedUrlMap(string) (any, bool)
	isValidHostname(string) bool
}

type UrlShortener struct {
	urlMap         sync.Map
	invertedUrlMap sync.Map
	hostname       string
	len            int
}

func (u *UrlShortener) loadFromUrlMap(originalUrl string) (any, bool) {
	return u.urlMap.Load(originalUrl)
}

func (u *UrlShortener) loadFromInvertedUrlMap(shortenedUrl string) (any, bool) {
	return u.invertedUrlMap.Load(shortenedUrl)
}

func (u *UrlShortener) isValidHostname(host string) bool {
	return u.hostname != host
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

func getShortenedUrl(w http.ResponseWriter, r *http.Request, urlShortener Shortener) {
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
	shortenedUrl, ok := urlShortener.loadFromUrlMap(originalUrl)
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

func getOriginalUrl(w http.ResponseWriter, r *http.Request, urlShortener Shortener) {
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
	if !urlShortener.isValidHostname(parsedUrl.Host) {
		http.Error(w, "invalid shortened url host passed in request", http.StatusBadRequest)
		return
	}
	originalUrl, ok := urlShortener.loadFromInvertedUrlMap(shortenedUrl)
	if !ok {
		http.Error(w, "invalid shortened url host passed in request", http.StatusBadRequest)
		return
	}
	originalUrlStr, ok := originalUrl.(string)
	if !ok {
		http.Error(w, "invalid original url found", http.StatusInternalServerError)
		//w.WriteHeader(http.StatusInternalServerError)
		//w.Write([]byte("invalid original url found"))
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
			getOriginalUrl(w, r, &urlShortener)

		case http.MethodPost:
			getShortenedUrl(w, r, &urlShortener)

		default:
			//w.WriteHeader(http.StatusMethodNotAllowed)
			//w.Write([]byte("method not allowed"))
			//return
			http.Error(w, "Invalid http operation"+r.Method, http.StatusMethodNotAllowed)
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
