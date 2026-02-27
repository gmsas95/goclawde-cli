package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	clientID    = "daun_cli_P0Snpk4zibbssa4jY2g-Vs-W"
	redirectURI = "https://myrai.blytz.cloud/oauth/callback"
)

func main() {
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	fmt.Println("🚀 Daun OAuth Helper")
	fmt.Println("====================")
	fmt.Println()
	fmt.Printf("Redirect URI: %s\n", redirectURI)
	fmt.Println()

	// Step 1: Show URL
	authURL := fmt.Sprintf(
		"https://daun.me/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&response_type=code&state=xyz123&code_challenge=%s&code_challenge_method=S256",
		clientID,
		url.QueryEscape(redirectURI),
		url.QueryEscape("posts:read,posts:write,users:read,me:read"),
		url.QueryEscape(codeChallenge),
	)

	fmt.Println("STEP 1: Update your Daun app settings")
	fmt.Println("--------------------------------------")
	fmt.Printf("Go to: https://daun.me/_syedakmal/settings/apps/01KJD1N3M1FBVMD9YRFYN5EWJX\n")
	fmt.Printf("Set Redirect URI to: %s\n", redirectURI)
	fmt.Println()

	fmt.Println("STEP 2: Open this URL and click Authorize:")
	fmt.Println("-------------------------------------------")
	fmt.Printf("%s\n\n", authURL)

	fmt.Println("STEP 3: After authorizing, you'll see JSON with 'code'. Copy the code.")
	fmt.Println()

	// Step 4: Manual code input
	fmt.Println("STEP 4: Paste the authorization code here:")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	authCode, _ := reader.ReadString('\n')
	authCode = strings.TrimSpace(authCode)

	if authCode == "" {
		fmt.Println("❌ No code provided")
		return
	}

	fmt.Println()
	fmt.Println("Now run this curl command to exchange for API key:")
	fmt.Println("---------------------------------------------------")
	fmt.Printf(`curl -X POST "https://daun.me/api/v2/oauth/token" \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "authorization_code",
    "client_id": "%s",
    "client_secret": "YOUR_CLIENT_SECRET_HERE",
    "code": "%s",
    "redirect_uri": "%s",
    "code_verifier": "%s"
  }'`+"\n\n", clientID, authCode, redirectURI, codeVerifier)

	fmt.Println("Copy YOUR_CLIENT_SECRET from Daun app settings!")
	fmt.Printf("\nCode Verifier (save this): %s\n", codeVerifier)
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
