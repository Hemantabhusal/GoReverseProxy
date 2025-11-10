package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Production Configuration
	basePath := "/hemanta/proxy"
	masterHost := "www.mydomain.com"
	targetURLStr := "https://vritjobs.com/"
	listenAddr := ":7070"
	logDir := "/var/log/go-proxy"

	// Setup logging
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Warning: Could not create log directory: %v", err)
		log.Printf("Logging to stdout only")
	} else {
		logFile := filepath.Join(logDir, "proxy.log")
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("Warning: Could not open log file: %v", err)
		} else {
			log.SetOutput(io.MultiWriter(os.Stdout, f))
			log.Printf("Logging to: %s", logFile)
		}
	}

	log.Printf("=== Production Reverse Proxy Starting ===")
	log.Printf("Base Path: %s", basePath)
	log.Printf("Master Host: %s (HTTPS)", masterHost)
	log.Printf("Target URL: %s", targetURLStr)
	log.Printf("Listen: %s", listenAddr)
	log.Printf("Time: %s", time.Now().Format(time.RFC3339))

	targetURL, err := url.Parse(targetURLStr)
	if err != nil {
		log.Fatalf("Invalid target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify outgoing requests
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origPath := req.URL.Path

		// Strip base path before fetching
		strippedPath := strings.TrimPrefix(req.URL.Path, basePath)
		if strippedPath == "" {
			strippedPath = "/"
		}

		origDirector(req)
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = strippedPath
		req.Host = targetURL.Host

		log.Printf("→ [%s] %s → %s://%s%s", req.Method, origPath, req.URL.Scheme, req.URL.Host, req.URL.Path)
	}

	// Modify responses
	proxy.ModifyResponse = func(resp *http.Response) error {
		upPrefix := targetURL.Scheme + "://" + targetURL.Host
		masterPrefix := "https://" + masterHost

		// 1. Fix Location header (redirects)
		if loc := resp.Header.Get("Location"); loc != "" {
			newLoc := loc

			if strings.HasPrefix(loc, upPrefix) {
				// https://vritjobs.com/path → https://www.mydomain.com/hemanta/proxy/path
				newLoc = strings.Replace(loc, upPrefix, masterPrefix+basePath, 1)
			} else if strings.HasPrefix(loc, "/") {
				// /path → /hemanta/DEV2/path
				newLoc = basePath + loc
			}

			if newLoc != loc {
				resp.Header.Set("Location", newLoc)
				log.Printf("[%d] Redirect: %s → %s", resp.StatusCode, loc, newLoc)
			}
		}

		// 2. Fix cookies
		cookies := resp.Header.Values("Set-Cookie")
		if len(cookies) > 0 {
			resp.Header.Del("Set-Cookie")
			for _, c := range cookies {
				// Fix domain
				c = strings.ReplaceAll(c, "Domain="+targetURL.Host, "Domain="+masterHost)
				c = strings.ReplaceAll(c, "Domain=."+targetURL.Host, "Domain="+masterHost)

				// Fix path
				if !strings.Contains(c, "Path=") {
					c = c + "; Path=" + basePath + "/"
				} else {
					c = strings.ReplaceAll(c, "Path=/", "Path="+basePath+"/")
				}

				// Ensure Secure flag for HTTPS
				if !strings.Contains(c, "Secure") {
					c = c + "; Secure"
				}

				resp.Header.Add("Set-Cookie", c)
			}
		}

		// 3. Rewrite HTML/CSS/JS
		ct := resp.Header.Get("Content-Type")
		if strings.Contains(ct, "text/html") ||
			strings.Contains(ct, "text/css") ||
			strings.Contains(ct, "javascript") ||
			strings.Contains(ct, "application/json") {

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading body: %v", err)
				return err
			}
			resp.Body.Close()

			bodyStr := string(body)

			// Replace absolute URLs
			bodyStr = strings.ReplaceAll(bodyStr, upPrefix+"/", masterPrefix+basePath+"/")
			bodyStr = strings.ReplaceAll(bodyStr, "//"+targetURL.Host+"/", "//"+masterHost+basePath+"/")

			// Replace relative URLs in attributes
			bodyStr = strings.ReplaceAll(bodyStr, `href="/`, `href="`+basePath+`/`)
			bodyStr = strings.ReplaceAll(bodyStr, `src="/`, `src="`+basePath+`/`)
			bodyStr = strings.ReplaceAll(bodyStr, `action="/`, `action="`+basePath+`/`)
			bodyStr = strings.ReplaceAll(bodyStr, `data-url="/`, `data-url="`+basePath+`/`)

			// CSS urls
			bodyStr = strings.ReplaceAll(bodyStr, `url(/`, `url(`+basePath+`/`)
			bodyStr = strings.ReplaceAll(bodyStr, `url("/`, `url("`+basePath+`/`)
			bodyStr = strings.ReplaceAll(bodyStr, `url('/`, `url('`+basePath+`/`)

			resp.Body = io.NopCloser(bytes.NewBufferString(bodyStr))
			resp.ContentLength = int64(len(bodyStr))
			resp.Header.Set("Content-Length", strconv.Itoa(len(bodyStr)))

			log.Printf("[%d] Rewrote %s (%d bytes)", resp.StatusCode, ct, len(bodyStr))
		}

		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf(" ERROR [%s %s]: %v", r.Method, r.URL.Path, err)
		http.Error(w, "Proxy Error: Unable to reach backend", http.StatusBadGateway)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	log.Printf("Proxy ready and listening on %s", listenAddr)
	log.Printf("Access via: https://%s%s/", masterHost, basePath)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
