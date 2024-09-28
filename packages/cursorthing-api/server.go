package main

import (
	"context"
	"cursorthing-api/prism"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (server *Server) ListenAndServe(port int) error {
	p := prism.NewRouter()

	// normal REST endpoint
	p.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// prism endpoints
	p.HandlePrismFunc("join", joinHandler)
	p.HandlePrismFunc("leave", leaveHandler)

	p.ListenAndServe(port)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	p.Close(context.Background())

	return nil
}

func joinHandler(c *prism.Context) {

	fullUrl, err := c.TextParam()
	if err != nil {
		c.Error(err)
		return
	}

	norm, err := normalizeUrl(fullUrl)
	if err != nil {
		c.Error(err)
		return
	}

	// todo

	c.ResponseText(norm)
}

func leaveHandler(c *prism.Context) {
}

func normalizeUrl(fullUrl string) (string, error) {

	u, err := url.Parse(fullUrl)
	if err != nil {
		return "", err
	}

	if u.Scheme != "https" {
		return "", fmt.Errorf("scheme must be https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("host is required")
	}
	u.Host = strings.TrimPrefix(u.Host, "www.")

	return fmt.Sprintf("%s/%s", u.Host, strings.TrimPrefix(u.Path, "/")), nil
}
