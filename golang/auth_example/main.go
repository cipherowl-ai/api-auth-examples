package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

const (
	cipherowlApiUrl = "https://svc.cipherowl.ai"
)

var (
	cipherowlTokenPath string
	clientID           string
	clientSecret       string
)

type TokenCache struct {
	AccessToken string `json:"access_token"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func init() {
	// Initialize logging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Set up token path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error getting home directory:", err)
	}
	cipherowlTokenPath = filepath.Join(homeDir, ".cipherowl", "token-cache.json")

	// Get environment variables
	clientID = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
}

func getTokenFromCache() (string, error) {
	data, err := os.ReadFile(cipherowlTokenPath)
	if err != nil {
		return "", err
	}

	var tokenCache TokenCache
	if err := json.Unmarshal(data, &tokenCache); err != nil {
		return "", err
	}

	// Decode token without verifying signature
	token, _, err := new(jwt.Parser).ParseUnverified(tokenCache.AccessToken, jwt.MapClaims{})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return "", fmt.Errorf("invalid exp claim")
	}

	if time.Now().Unix() < int64(exp) {
		log.Println("Get token from cache")
		return tokenCache.AccessToken, nil
	}

	return "", fmt.Errorf("token expired")
}

func writeTokenToCache(token string) error {
	dir := filepath.Dir(cipherowlTokenPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tokenCache := TokenCache{AccessToken: token}
	data, err := json.Marshal(tokenCache)
	if err != nil {
		return err
	}

	if err := os.WriteFile(cipherowlTokenPath, data, 0644); err != nil {
		return err
	}

	log.Println("Write token to cache")
	return nil
}

func getTokenFromServer() (string, error) {
	url := fmt.Sprintf("%s/oauth/token", cipherowlApiUrl)

	payload := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"audience":      "svc.cipherowl.ai",
		"grant_type":    "client_credentials",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	log.Println("Get token from server")
	if err := writeTokenToCache(tokenResp.AccessToken); err != nil {
		log.Printf("Warning: Failed to cache token: %v", err)
	}

	return tokenResp.AccessToken, nil
}

func getToken() (string, error) {
	// Try to get token from cache first
	token, err := getTokenFromCache()
	if err == nil {
		return token, nil
	}

	// If cache fails, get from server
	return getTokenFromServer()
}

func main() {
	project := "partner"
	url := fmt.Sprintf("%s/api/v1/sanction?project=%s&chain=bitcoin_mainnet&address=12udabs2TkX7NXCSj6KpqXfakjE52ZPLhz",
		cipherowlApiUrl, project)

	token, err := getToken()
	if err != nil {
		log.Fatal("Error getting token:", err)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error making request:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("Server returned status:", resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatal("Error decoding response:", err)
	}

	prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Fatal("Error formatting JSON:", err)
	}

	fmt.Println(string(prettyJSON))
}
