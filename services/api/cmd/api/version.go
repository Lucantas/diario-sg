package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"
)

func binaryVersion(l *slog.Logger) string {
	version, err := executableHash()
	if err != nil {
		l.Warn("sem versão do binário, os padrões guardados não serão reaproveitados", "error", err)
		return "processo-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return version
}

func executableHash() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}
