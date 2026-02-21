package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/browser"
)

// componentFunc is a function that renders fresh HTML from the current
// markdown content. It is called once at startup and again on every file
// change when watch mode is enabled.
type componentFunc func() templ.Component

// Server holds the state needed to serve a rendered markdown page.
type Server struct {
	componentFn   componentFunc
	baseDir       string
	allowedImages map[string]bool
	watchMode     bool
	filePath      string

	// mu protects html.
	mu   sync.RWMutex
	html []byte

	// sseClients is the set of active SSE subscriber channels.
	sseMu      sync.Mutex
	sseClients map[chan struct{}]struct{}
}

// New creates a Server that serves the given component as the root page,
// and allows only the listed relative image paths from baseDir.
//
// componentFn is called once immediately to render the initial HTML, and
// again on every file-change event when watch mode is enabled.
func New(componentFn componentFunc, baseDir string, allowedImages map[string]bool, watchMode bool, filePath string) *Server {
	s := &Server{
		componentFn:   componentFn,
		baseDir:       baseDir,
		allowedImages: allowedImages,
		watchMode:     watchMode,
		filePath:      filePath,
		sseClients:    make(map[chan struct{}]struct{}),
	}
	s.html = s.render(componentFn())
	return s
}

// render renders a templ component to raw HTML bytes.
func (s *Server) render(c templ.Component) []byte {
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		log.Printf("render error: %v", err)
	}
	return buf.Bytes()
}

// broadcast sends a reload signal to all connected SSE clients.
func (s *Server) broadcast() {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for ch := range s.sseClients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// watch starts an fsnotify watcher on filePath. On each write event the
// markdown is re-rendered and connected browsers are notified via SSE.
func (s *Server) watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("watch: create watcher: %v", err)
	}
	if err := watcher.Add(s.filePath); err != nil {
		log.Fatalf("watch: add file %q: %v", s.filePath, err)
	}
	log.Printf("Watching %s for changes…", s.filePath)

	// Debounce: wait for a short quiet period before re-rendering.
	var (
		debounce   *time.Timer
		debounceMu sync.Mutex
	)
	fire := func() {
		debounceMu.Lock()
		defer debounceMu.Unlock()
		if debounce != nil {
			debounce.Stop()
		}
		debounce = time.AfterFunc(50*time.Millisecond, func() {
			component := s.componentFn()
			html := s.render(component)
			s.mu.Lock()
			s.html = html
			s.mu.Unlock()
			s.broadcast()
			log.Printf("Reloaded %s", s.filePath)
		})
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				fire()
			}
			// Some editors (vim, IntelliJ) rename-replace files on save.
			// Re-add the watcher after a rename in case the inode changed.
			if event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
				// Wait briefly for the new file to appear then re-watch.
				go func() {
					time.Sleep(100 * time.Millisecond)
					_ = watcher.Add(s.filePath)
					fire()
				}()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watch error: %v", err)
		}
	}
}

// Start registers the HTTP handlers, prints the startup message, opens the
// browser, and blocks serving on addr (e.g. "localhost:3000").
func (s *Server) Start(addr string) {
	fs := http.FileServer(http.Dir(s.baseDir))

	mux := http.NewServeMux()

	// Root: serve the latest rendered HTML.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			s.mu.RLock()
			html := s.html
			s.mu.RUnlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if _, err := w.Write(html); err != nil {
				log.Printf("write response: %v", err)
			}
			return
		}
		// Strip leading slash to match the path as written in the markdown.
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if s.allowedImages[reqPath] || s.allowedImages["./"+reqPath] {
			fs.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	// SSE reload endpoint — only registered when watch mode is on.
	if s.watchMode {
		mux.HandleFunc("/__reload", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")

			ch := make(chan struct{}, 1)
			s.sseMu.Lock()
			s.sseClients[ch] = struct{}{}
			s.sseMu.Unlock()

			defer func() {
				s.sseMu.Lock()
				delete(s.sseClients, ch)
				s.sseMu.Unlock()
			}()

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming not supported", http.StatusInternalServerError)
				return
			}

			// Send a heartbeat comment every 15 s to keep the connection alive.
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-r.Context().Done():
					return
				case <-ch:
					fmt.Fprintf(w, "data: reload\n\n")
					flusher.Flush()
				case <-ticker.C:
					fmt.Fprintf(w, ": heartbeat\n\n")
					flusher.Flush()
				}
			}
		})

		go s.watch()
	}

	fmt.Printf("Server starting at http://%s\n", addr)
	if s.watchMode {
		fmt.Printf("Watch mode enabled — browser will reload on file changes\n")
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := browser.OpenURL("http://" + addr); err != nil {
			log.Printf("open browser: %v", err)
		}
	}()

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
