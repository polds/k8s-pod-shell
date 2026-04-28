package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/polds/k8s-pod-shell/internal/app"
	"github.com/polds/k8s-pod-shell/internal/kube"
	"k8s.io/client-go/kubernetes"
)

type Server struct {
	cfg        app.Config
	kube       kubernetes.Interface
	assets     fs.FS
	readyProbe atomic.Bool
}

func New(cfg app.Config, cs kubernetes.Interface, embedded embed.FS) (*Server, error) {
	dist, err := fs.Sub(embedded, "web/dist")
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, kube: cs, assets: dist}, nil
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/healthz", s.healthz)
	mux.HandleFunc("/api/v1/readyz", s.readyz)
	mux.HandleFunc("/api/v1/info", s.info)
	mux.HandleFunc("/api/v1/pods", s.pods)
	mux.HandleFunc("/api/v1/exec", s.exec)
	mux.HandleFunc("/", s.spa)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	if s.readyProbe.Load() {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "not ready", http.StatusServiceUnavailable)
}

func (s *Server) info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"target": map[string]string{
			"kind":      s.cfg.TargetKind,
			"name":      s.cfg.TargetName,
			"namespace": s.targetNamespace(),
		},
		"version": s.cfg.Version,
		"gitSha":  s.cfg.GitSHA,
	})
}

func (s *Server) pods(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	pods, err := kube.ResolvePods(ctx, s.kube, kube.Target{
		Kind:      s.cfg.TargetKind,
		Name:      s.cfg.TargetName,
		Namespace: s.targetNamespace(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.readyProbe.Store(true)
	writeJSON(w, pods)
}

func (s *Server) exec(w http.ResponseWriter, r *http.Request) {
	if !s.isAllowedOrigin(r.Header.Get("Origin")) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	pod := r.URL.Query().Get("pod")
	if pod == "" {
		http.Error(w, "pod query param required", http.StatusBadRequest)
		return
	}
	container := r.URL.Query().Get("container")
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: s.cfg.AllowedOrigins,
	})
	if err != nil {
		return
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "session ended") }()
	conn.SetReadLimit(1024 * 1024)

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.IdleTimeout)
	defer cancel()

	if err := RunExecSession(ctx, conn, s.kube, s.targetNamespace(), pod, container); err != nil {
		_ = sendStatus(conn, map[string]any{"type": "error", "msg": err.Error()})
		_ = conn.Close(websocket.StatusInternalError, "exec failed")
	}
}

func (s *Server) spa(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	path := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
	if path == "." || path == "" {
		path = "index.html"
	}
	b, err := fs.ReadFile(s.assets, path)
	if err != nil {
		b, err = fs.ReadFile(s.assets, "index.html")
		if err != nil {
			http.Error(w, "frontend assets missing", http.StatusInternalServerError)
			return
		}
	}
	if strings.HasSuffix(path, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	}
	if strings.HasSuffix(path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	_, _ = w.Write(b)
}

func (s *Server) targetNamespace() string {
	if s.cfg.TargetNS != "" {
		return s.cfg.TargetNS
	}
	return "default"
}

func (s *Server) isAllowedOrigin(origin string) bool {
	if len(s.cfg.AllowedOrigins) == 0 {
		return true
	}
	for _, o := range s.cfg.AllowedOrigins {
		if o == origin {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func sendStatus(conn *websocket.Conn, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	frame := append([]byte{0x02}, body...)
	return conn.Write(context.Background(), websocket.MessageBinary, frame)
}

type wsWriter struct{ conn *websocket.Conn }

func (w wsWriter) Write(p []byte) (int, error) {
	frame := append([]byte{0x00}, p...)
	if err := w.conn.Write(context.Background(), websocket.MessageBinary, frame); err != nil {
		return 0, err
	}
	return len(p), nil
}

var errUnsupportedControl = errors.New("unsupported control frame")

func readFrameData(data []byte) (byte, []byte, error) {
	if len(data) == 0 {
		return 0, nil, io.EOF
	}
	return data[0], data[1:], nil
}
