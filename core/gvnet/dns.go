package gvnet

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"gitlab.com/tozd/go/errors"
)

func searchDomains(ctx context.Context) ([]string, error) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		f, err := os.Open("/etc/resolv.conf")
		if err != nil {
			return nil, errors.Errorf("opening resolv.conf: %w", err)
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		searchPrefix := "search "
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), searchPrefix) {
				return parseSearchString(ctx, sc.Text(), searchPrefix), nil
			}
		}
		if err := sc.Err(); err != nil {
			return nil, errors.Errorf("scanning resolv.conf: %w", err)
		}
	}
	return nil, errors.New("only Linux and macOS are supported currently")
}

// Parse and sanitize search list
// macOS has limitation on number of domains (6) and general string length (256 characters)
// since glibc 2.26 Linux has no limitation on 'search' field
func parseSearchString(ctx context.Context, text, searchPrefix string) []string {
	// macOS allow only 265 characters in search list
	if runtime.GOOS == "darwin" && len(text) > 256 {
		slog.WarnContext(ctx, "Search domains list is too long, it should not exceed 256 chars on macOS - truncating", "length", len(text))
		text = text[:256]
		lastSpace := strings.LastIndex(text, " ")
		if lastSpace != -1 {
			text = text[:lastSpace]
		}
	}

	searchDomains := strings.Split(strings.TrimPrefix(text, searchPrefix), " ")
	slog.DebugContext(ctx, "Using search domains", "domains", searchDomains)

	// macOS allow only 6 domains in search list
	if runtime.GOOS == "darwin" && len(searchDomains) > 6 {
		slog.WarnContext(ctx, "Search domains list is too long, it should not exceed 6 domains on macOS - truncating", "length", len(searchDomains))
		searchDomains = searchDomains[:6]
	}

	return searchDomains
}
