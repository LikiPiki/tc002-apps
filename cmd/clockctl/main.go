package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func main() {
	url := flag.String("url", os.Getenv("AWTRIX_URL"), "AWTRIX NG base URL (or AWTRIX_URL)")
	text := flag.String("text", "Hello from Go", "notification text")
	flag.Parse()

	client, err := awtrix.NewClient(*url)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = client.Notify(ctx, *text)
	}
	if err != nil {
		slog.Error("notification failed", "error", err)
		os.Exit(1)
	}
	slog.Info("notification sent")
}
