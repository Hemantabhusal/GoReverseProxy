# Go Reverse Proxy

A powerful reverse proxy server that lets you **embed and access most of the website inside your own domain** while keeping your domain name visible in the browser. Perfect for domain masking, white-labeling applications, or hosting multiple services under a unified domain.

## What Does This Do?

🎭 **Domain Masking:** Access `https://target.com` through `https://mydomain.com/hemanta/proxy/` - visitors only see YOUR domain in the address bar!

🔗 **Seamless Integration:** The proxy acts as a transparent bridge, fetching content from the target website and serving it through your domain without any visible redirects.

🛠️ **Intelligent Rewriting:** Automatically rewrites all URLs, links, images, scripts, stylesheets, and API calls to work perfectly under your custom path. The target website doesn't even know it's being proxied!

**Example:** Turn `https://target.com/dashboard` into `https://mydomain.com/hemanta/proxy/dashboard` - everything works, cookies persist, and your domain stays in the URL bar!

## Features

- 🔄 URL rewriting for HTML, CSS, JavaScript, and JSON
- 🍪 Automatic cookie domain and path fixing
- 🔀 Redirect handling with path prefixing
- 📝 Request/response logging
- ⚙️ Environment-based configuration

## Apache2 Configuration

To expose the Go proxy through Apache2 with SSL termination, you'll need to configure Apache as a reverse proxy to forward requests to the Go proxy server.

**Sample configuration files are provided in the repository** - check the `proxy.conf` file and modify it according to your needs.

### Key Configuration Points

- **SSL Termination:** Apache handles HTTPS/SSL certificates
- **Proxy Pass:** Apache forwards requests to Go proxy (typically `http://127.0.0.1:7070`)
- **Modules Required:** `proxy`, `proxy_http`, `ssl`, `rewrite`

### Enable Apache Proxy Modules

**Debian/Ubuntu:**
```bash
sudo a2enmod proxy proxy_http ssl rewrite
sudo systemctl restart apache2
```

**RHEL/CentOS:**
```bash
# Modules are typically enabled by default
sudo systemctl restart httpd
```


### Architecture Overview

```
User Browser
    ↓ (HTTPS)
    ↓ https://www.yourdomain.com/hemanta/proxy/
    ↓
Apache2 (:443) - HTTPS, SSL termination
    ↓ (HTTP)
    ↓ http://127.0.0.1:7070/hemanta/proxy/
    ↓
Go Proxy (:7070) - URL Rewriting & Content Modification
    ↓ (HTTPS)
    ↓ https://www.target.com/
    ↓
Target Backend Server
```

**Benefits of this setup:**
- Apache handles SSL/TLS certificates (Let's Encrypt compatible)
- Go proxy handles complex URL rewriting and content modification
- Easy to add multiple proxied applications (different paths, same Apache server)
- Centralized logging and access control through Apache


### Ideas for Contributions

- Add rate limiting and throttling
- Implement health check endpoints
- Add support for WebSocket proxying
- Create configuration file support (YAML/TOML)
- Add metrics and monitoring integration
- Improve error handling and recovery
- Add unit and integration tests
- Documentation improvements

