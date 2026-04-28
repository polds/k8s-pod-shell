package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/polds/k8s-pod-shell/internal/app"
	"github.com/polds/k8s-pod-shell/internal/kube"
	"github.com/polds/k8s-pod-shell/internal/server"
)

//go:embed web/dist/*
var webAssets embed.FS

var (
	version = "dev"
	gitSHA  = "unknown"
)

func main() {
	cfg := app.LoadConfig()
	cfg.Version = version
	cfg.GitSHA = gitSHA
	cs, err := kube.NewClientset()
	if err != nil {
		log.Fatalf("kube client: %v", err)
	}
	srv, err := server.New(cfg, cs, webAssets)
	if err != nil {
		log.Fatalf("server init: %v", err)
	}
	log.Printf("kubeshell-web listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
