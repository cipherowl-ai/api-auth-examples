package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

const (
	cipherowlApiUrl = "https://svc.cipherowl.ai"
)

type TokenCache struct {
	AccessToken string
	ExpiresAt   time.Time
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

var (
	clientID     string
	clientSecret string
	tokenCache   *TokenCache
)

func init() {
	// Initialize logging
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Initialize token cache
	tokenCache = &TokenCache{}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Get environment variables
	clientID = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
}

func getTokenFromCache() (string, error) {
	if tokenCache.AccessToken == "" {
		return "", fmt.Errorf("no token in cache")
	}

	if time.Now().After(tokenCache.ExpiresAt) {
		return "", fmt.Errorf("token expired")
	}

	log.Println("Get token from cache")
	return tokenCache.AccessToken, nil
}

func writeTokenToCache(token string) error {
	// Decode token without verifying signature to get expiration
	parsedToken, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("invalid exp claim")
	}

	tokenCache.AccessToken = token
	tokenCache.ExpiresAt = time.Unix(int64(exp), 0)

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
	url := fmt.Sprintf("%s/api/screen/v1/chains/evm/addresses/0xf4377eda661e04b6dda78969796ed31658d602d4?config=co-high_risk_hops_2",
		cipherowlApiUrl)

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
