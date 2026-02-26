# Daun OAuth Helper - Go Script

This Go script helps you get a Daun API key via OAuth flow.

## Prerequisites

1. **Go installed** on your local machine (not the VPS)
2. **Client ID** and **Client Secret** from Daun app settings

## Setup Steps

### 1. Get Your Credentials from Daun

1. Go to: `https://daun.me/{your_username}/settings/apps`
2. Click on your app (or create one)
3. Note down:
   - **Client ID** (should be: `daun_cli_P0Snpk4zibbssa4jY2g-Vs-W`)
   - **Client Secret** (keep this secret!)
4. Set **Redirect URI** to:
   ```
   http://localhost:8080/oauth/callback
   ```

### 2. Edit the Script

Open `scripts/oauth_helper.go` and replace:
```go
const (
    clientID     = "daun_cli_P0Snpk4zibbssa4jY2g-Vs-W"
    clientSecret = "YOUR_CLIENT_SECRET_HERE" // ← REPLACE THIS!
    ...
)
```

With your actual client secret:
```go
const (
    clientID     = "daun_cli_P0Snpk4zibbssa4jY2g-Vs-W"
    clientSecret = "daun_sec_xxxxxxxxxxxxxxxx" // Your actual secret
    ...
)
```

### 3. Run the Script

```bash
cd scripts
go run oauth_helper.go
```

### 4. Authorize in Browser

The script will print a URL like:
```
https://daun.me/oauth/authorize?client_id=daun_cli_P0Snpk4zibbssa4jY2g-Vs-W&...
```

1. **Copy and paste** this URL into your browser
2. Click **"Authorize"** on Daun
3. You'll be redirected to localhost
4. The script will capture the code and exchange it for an API key

### 5. Copy the API Key

The script will output:
```
🎉 SUCCESS!
============
Your API Key: daun_sec_xxxxxxxxxxxxxxxx

Add this to your Dokploy environment variables:
DAUN_API_KEY=daun_sec_xxxxxxxxxxxxxxxx
```

### 6. Update Dokploy

1. Go to your Dokploy dashboard
2. Navigate to your Myrai app → Environment
3. Update `DAUN_API_KEY` with the new key
4. Redeploy

## Troubleshooting

### "Port already in use"
Change the port in the script:
```go
const port = ":8081" // or any other port
```

### "Client secret invalid"
- Make sure you copied the correct client secret
- It should be from the same app as the client ID

### "Redirect URI mismatch"
- Make sure the redirect URI in Daun app settings matches exactly
- Must be: `http://localhost:8080/oauth/callback`

## Security Note

⚠️ **Never commit the oauth_helper.go file with your client secret!**

The script is in `.gitignore` to prevent accidental commits.
