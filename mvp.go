package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

func main() {
	// Set master URL (what user sees) and upstream target
	masterHost := flag.String("master", "localhost:7070", "master host (what users see)")
	target := flag.String("target", "https://vritjobs.com", "upstream target URL")
	listen := flag.String("listen", ":7070", "listen address")
	flag.Parse()

	targetURL, err := url.Parse(*target)
	if err != nil {
		log.Fatalf("invalid target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Director rewrites the request to upstream
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		// Forward original path exactly
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.Host = targetURL.Host
		req.Header.Set("X-Forwarded-Host", *masterHost)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	}

	// ModifyResponse rewrites Location, cookies, and basic HTML
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 1) Rewrite Location headers
		if loc := resp.Header.Get("Location"); loc != "" {
			if strings.HasPrefix(loc, targetURL.Scheme+"://"+targetURL.Host) {
				newLoc := strings.Replace(loc, targetURL.Scheme+"://"+targetURL.Host, "http://"+*masterHost, 1)
				resp.Header.Set("Location", newLoc)
			}
		}

		// 2) Rewrite Set-Cookie domain
		cookies := resp.Header.Values("Set-Cookie")
		if len(cookies) > 0 {
			resp.Header.Del("Set-Cookie")
			for _, c := range cookies {
				c = strings.ReplaceAll(c, "Domain="+targetURL.Host, "Domain="+*masterHost)
				resp.Header.Add("Set-Cookie", c)
			}
		}

		// 3) Basic HTML body rewrite for absolute links
		ct := resp.Header.Get("Content-Type")
		if strings.Contains(ct, "text/html") {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body.Close()
			bodyStr := string(bodyBytes)

			upPrefix := targetURL.Scheme + "://" + targetURL.Host
			masterPrefix := "http://" + *masterHost

			bodyStr = strings.ReplaceAll(bodyStr, upPrefix, masterPrefix)
			bodyStr = strings.ReplaceAll(bodyStr, "//"+targetURL.Host, "//"+*masterHost)

			resp.Body = io.NopCloser(bytes.NewBufferString(bodyStr))
			resp.ContentLength = int64(len(bodyStr))
			resp.Header.Set("Content-Length", strconv.Itoa(len(bodyStr)))
		}
		return nil
	}

	// Serve all requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	log.Printf("Starting Go reverse proxy: master=%s  target=%s  listen=%s\n", *masterHost, targetURL.String(), *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}
