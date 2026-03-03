# Security Model for Self-Hosted Myrai

## Philosophy

Myrai is designed as a **personal, self-hosted AI assistant** - not a multi-tenant SaaS. The security model prioritizes:

1. **Convenience** - Easy setup for the owner
2. **Defense in depth** - Multiple layers of protection
3. **Network-based security** - Rely on infrastructure (reverse proxy, VPN)
4. **Optional hardening** - Security features you can enable

## Threat Model

### What We're Protecting Against
- ✅ Unauthorized access to your AI conversations
- ✅ API key theft (your OpenAI/Kimi keys)
- ✅ Prompt injection from malicious inputs
- ✅ Accidental data exposure

### What We DON'T Protect Against (by design)
- ❌ Physical server access (use disk encryption)
- ❌ Network sniffing on local network (use HTTPS)
- ❌ Docker host compromise (use proper container security)

## Security Layers

### Layer 1: Network Security (Recommended)

**Option A: Local Network Only**
```yaml
# docker-compose.yml
services:
  goclawde:
    ports:
      - "127.0.0.1:8080:8080"  # Only localhost
```

**Option B: Reverse Proxy with Authelia/Authentik**
```nginx
# nginx.conf
server {
    listen 443 ssl;
    server_name myrai.yourdomain.com;
    
    # Forward to Authelia for auth
    location / {
        include /config/nginx/proxy.conf;
        include /config/nginx/authelia-server.conf;
        proxy_pass http://goclawde:8080;
    }
}
```

**Option C: VPN/WireGuard**
- Don't expose port 8080 at all
- Access only via VPN
- Most secure for remote access

### Layer 2: Authentication Modes

Myrai supports multiple auth modes based on your needs:

#### Mode 1: "Trust My Network" (Default)
- No password required
- Relies on network-level security
- Best for: Local-only, VPN-protected, or reverse-proxy-with-auth setups

```bash
# .env - No password set
# GOCLAWDE_ADMIN_PASSWORD=...
```

#### Mode 2: "Simple Password"
- Single password for dashboard access
- No username needed
- Session-based JWT tokens

```bash
# .env
GOCLAWDE_ADMIN_PASSWORD=your-secure-password
```

#### Mode 3: "Token-Based" (For API/scripts)
- Gateway token for programmatic access
- Used by Telegram bot, external scripts

```bash
# .env
GOCLAWDE_GATEWAY_TOKEN=$(openssl rand -hex 32)
```

### Layer 3: Input Protection

**Built-in protections (always enabled):**
- ✅ Prompt injection detection
- ✅ Secret scanning (prevents accidental API key exposure)
- ✅ Input size limits (100KB max)
- ✅ File upload restrictions

**Optional hardening:**
- Enable CORS restrictions
- Add IP allowlisting

## Implementation Plan

### 1. Fix Login Handler (Critical Bug Fix)

The current login handler has a bug - it doesn't check the password. Fix:

```go
func (s *Server) handleLogin(c *fiber.Ctx) error {
    var req struct {
        Password string `json:"password"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
    }

    // If no admin password configured, rely on network security
    if s.config.Security.AdminPassword == "" {
        return c.Status(400).JSON(fiber.Map{
            "error": "password authentication not configured",
            "hint": "Set GOCLAWDE_ADMIN_PASSWORD or use network-level auth",
        })
    }

    // Check password
    if req.Password != s.config.Security.AdminPassword {
        return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
    }

    // Generate token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": "admin",
        "iat": time.Now().Unix(),
        "exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
    })

    tokenString, err := token.SignedString([]byte(s.config.Security.JWTSecret))
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
    }

    return c.JSON(fiber.Map{"token": tokenString})
}
```

### 2. Sensible CORS Defaults

```go
// Default to same-origin for self-hosted
v.SetDefault("security.allow_origins", []string{
    "http://localhost:8080",
    "http://127.0.0.1:8080",
})

// Override with env var if user wants to expose
allowOrigins := os.Getenv("GOCLAWDE_SECURITY_ALLOW_ORIGINS")
if allowOrigins != "" {
    v.Set("security.allow_origins", strings.Split(allowOrigins, ","))
}
```

### 3. File Upload Security (Practical)

For self-hosted use, we need basic protection without being overly restrictive:

```go
func (s *Server) handleFileUpload(c *fiber.Ctx) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "no file provided"})
    }

    // Size limit (10MB default)
    if file.Size > 10*1024*1024 {
        return c.Status(413).JSON(fiber.Map{"error": "file too large (max 10MB)"})
    }

    // Sanitize filename
    filename := sanitizeFilename(file.Filename)
    
    // Store in data directory (not accessible via web)
    path := filepath.Join(s.config.Storage.DataDir, "files", 
        fmt.Sprintf("%s_%s", time.Now().Format("20060102_150405"), filename))
    
    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "failed to create directory"})
    }

    if err := c.SaveFile(file, path); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "failed to save file"})
    }

    return c.JSON(fiber.Map{
        "filename": filename,
        "path": path,
        "size": file.Size,
    })
}

func sanitizeFilename(name string) string {
    // Remove path separators
    name = filepath.Base(name)
    // Remove null bytes
    name = strings.ReplaceAll(name, "\x00", "")
    // Keep only safe characters
    name = regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(name, "_")
    return name
}
```

### 4. Security Recommendations in Docs

Add to README:

```markdown
## Security Recommendations

### For Local Use Only
```bash
# Only expose to localhost
docker run -p 127.0.0.1:8080:8080 myrai:latest
```

### With Reverse Proxy (Traefik/Nginx)
- Enable HTTPS/TLS
- Use Authelia/Authentik for SSO
- Rate limiting at proxy level

### Remote Access
**Option 1: WireGuard VPN (Recommended)**
- No port exposure
- Encrypted tunnel
- Device-level authentication

**Option 2: Cloudflare Tunnel**
```bash
cloudflared tunnel --no-autoupdate run --token YOUR_TOKEN
```

### API Keys
Store in `.env` file (not in git):
```bash
chmod 600 .env
```
```

### 5. Security Checklist on Startup

Add to main.go:

```go
func runSecurityChecks(cfg *config.Config) {
    log := logger.Get()
    
    // Check if running as root
    if os.Getuid() == 0 {
        log.Warn("Running as root is not recommended for security")
    }
    
    // Check if admin password is set
    if cfg.Security.AdminPassword == "" {
        log.Info("No admin password set - relying on network-level security")
        log.Info("To enable password auth, set GOCLAWDE_ADMIN_PASSWORD")
    }
    
    // Check if bound to all interfaces
    if cfg.Server.Address == "0.0.0.0" {
        log.Warn("Server bound to all interfaces (0.0.0.0)")
        log.Warn("Consider using reverse proxy with HTTPS")
    }
    
    // Check file permissions
    if info, err := os.Stat(".env"); err == nil {
        if info.Mode().Perm()&0077 != 0 {
            log.Warn(".env file has overly permissive permissions")
            log.Warn("Run: chmod 600 .env")
        }
    }
}
```

## Configuration Examples

### Minimal (Trust Network)
```env
KIMI_API_KEY=sk-...
# No password - use reverse proxy or VPN
```

### With Password
```env
KIMI_API_KEY=sk-...
GOCLAWDE_ADMIN_PASSWORD=your-secure-password
GOCLAWDE_JWT_SECRET=$(openssl rand -hex 32)
```

### Production Hardened
```env
KIMI_API_KEY=sk-...
GOCLAWDE_ADMIN_PASSWORD=$(openssl rand -base64 32)
GOCLAWDE_JWT_SECRET=$(openssl rand -hex 32)
GOCLAWDE_GATEWAY_TOKEN=$(openssl rand -hex 32)
GOCLAWDE_SECURITY_ALLOW_ORIGINS=https://myrai.yourdomain.com
GOCLAWDE_SERVER_ADDRESS=127.0.0.1
```

## Summary

For self-hosted apps like Myrai:

1. **Default to open** - Don't require password (user can add if needed)
2. **Warn clearly** - Tell users about security implications
3. **Guide to best practices** - Document reverse proxy, VPN options
4. **Keep it simple** - No complex auth flows, RBAC, etc.
5. **Layered defense** - Network + Application + Input validation

The goal is: **Easy to get started, easy to secure when needed.**
