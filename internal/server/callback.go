package server

import (
	"fmt"
	"net/http"
	"time"
)

// CallbackServer handles the OAuth callback from GitHub
type CallbackServer struct {
	Port     int
	server   *http.Server
	codeChan chan string
	errChan  chan error
}

// New creates a new callback server on the specified port
func New(port int) *CallbackServer {
	return &CallbackServer{
		Port:     port,
		codeChan: make(chan string, 1),
		errChan:  make(chan error, 1),
	}
}

// Start begins listening for the OAuth callback
func (s *CallbackServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	return nil
}

// WaitForCode blocks until a code is received or timeout occurs
func (s *CallbackServer) WaitForCode(timeout time.Duration) (string, error) {
	select {
	case code := <-s.codeChan:
		return code, nil
	case err := <-s.errChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout: no callback received within %v", timeout)
	}
}

// Close shuts down the server
func (s *CallbackServer) Close() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Missing code parameter")
		s.errChan <- fmt.Errorf("no code in callback")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Success</title></head>
<body style="font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0;">
<div style="text-align: center;">
<h1 style="color: #22c55e;">GitHub App Created!</h1>
<p>You can close this window and return to your terminal.</p>
</div>
</body>
</html>`)

	s.codeChan <- code
}
