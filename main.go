package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
)

type Storage struct {
	mu   sync.RWMutex
	data map[string]string
}

var store = Storage{
	data: make(map[string]string),
}

type RequestBody struct {
	URL string `json:"url"`
}

// GenerateShortCode creates a random short code
func GenerateShortCode(length int) (string, error) {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, length)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}
	return string(code), nil
}

//handlers

// ShortenURL POST handler
func ShortenURL(ctx *gin.Context) {
	var req RequestBody
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate URL
	_, err := url.ParseRequestURI(req.URL)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	// Generate unique short code
	var shortCode string
	for {
		shortCode, err = GenerateShortCode(6)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate short code"})
			return
		}
		store.mu.RLock()
		_, exists := store.data[shortCode]
		store.mu.RUnlock()
		if !exists {
			break
		}
	}

	// Store mapping
	store.mu.Lock()
	store.data[shortCode] = req.URL
	store.mu.Unlock()

	shortURL := fmt.Sprintf("http://localhost:8080/%s", shortCode)
	ctx.JSON(http.StatusOK, gin.H{"short_url": shortURL})
}

// RetrieveURL  GET	handler
func RetrieveURL(ctx *gin.Context) {
	shortCode := ctx.Param("short")
	store.mu.RLock()
	longURL, exists := store.data[shortCode]
	store.mu.RUnlock()

	if !exists {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
		return
	}

	ctx.Redirect(http.StatusFound, longURL)
}

// Basic testing
func runTests() {
	fmt.Println("Running basic tests...")

	// Test short code generation
	code, err := GenerateShortCode(6)
	if err != nil || len(code) != 6 {
		log.Fatal("Short code generation failed")
	}
	fmt.Println("Short code generation works")

	// Test storage
	store.mu.Lock()
	store.data["abc123"] = "https://example.com"
	store.mu.Unlock()

	store.mu.RLock()
	val, ok := store.data["abc123"]
	store.mu.RUnlock()
	if !ok || val != "https://example.com" {
		log.Fatal("Storage test failed")
	}
	fmt.Println("Storage works")

	// Test JSON marshalling
	res, _ := json.Marshal(gin.H{"short_url": "http://localhost:8080/abc123"})
	fmt.Println("JSON Response:", string(res))

	fmt.Println("All tests passed!")
}

func main() {
	runTests()

	r := gin.Default()
	r.POST("/shorten", ShortenURL)
	r.GET("/:short", RetrieveURL)

	fmt.Println("Server running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
