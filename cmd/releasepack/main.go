package main

// @ACP O ACP.RELEASE

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type target struct{ GOOS, GOARCH string }

var targets = []target{{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}, {"windows", "arm64"}}

func main() {
	version := flag.String("version", "dev", "binary version")
	out := flag.String("out", "dist", "output directory")
	commit := flag.String("commit", env("GITHUB_SHA", "unknown"), "source commit")
	date := flag.String("date", time.Now().UTC().Format(time.RFC3339), "build date")
	flag.Parse()
	if err := os.RemoveAll(*out); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}
	names := []string{}
	module, err := modulePath()
	if err != nil {
		fatal(err)
	}
	for _, t := range targets {
		name, err := buildTarget(t, *version, *commit, *date, *out, module)
		if err != nil {
			fatal(err)
		}
		names = append(names, name)
		fmt.Println("BUILT", name)
	}
	sort.Strings(names)
	var lines []string
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(*out, name))
		if err != nil {
			fatal(err)
		}
		sum := sha256.Sum256(b)
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+name)
	}
	if err := os.WriteFile(filepath.Join(*out, "checksums.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		fatal(err)
	}
}

func buildTarget(t target, version, commit, date, out, module string) (string, error) {
	temp, err := os.MkdirTemp("", "acp-release-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temp)
	binary := "acp"
	if t.GOOS == "windows" {
		binary += ".exe"
	}
	binPath := filepath.Join(temp, binary)
	buildInfo := strings.TrimSpace(module) + "/internal/buildinfo"
	ld := fmt.Sprintf("-s -w -X %s.Version=%s -X %s.Commit=%s -X %s.Date=%s", buildInfo, version, buildInfo, commit, buildInfo, date)
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ld, "-o", binPath, "./cmd/acp")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.GOOS, "GOARCH="+t.GOARCH)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build %s/%s: %w", t.GOOS, t.GOARCH, err)
	}
	extras := []string{"README.md", "LICENSE", "PROTOCOL.md"}
	base := fmt.Sprintf("acp_%s_%s", t.GOOS, t.GOARCH)
	if t.GOOS == "windows" {
		name := base + ".zip"
		return name, writeZip(filepath.Join(out, name), binPath, binary, extras)
	}
	name := base + ".tar.gz"
	return name, writeTar(filepath.Join(out, name), binPath, binary, extras)
}

func writeTar(dest, binPath, binary string, extras []string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	if err := tarFile(tw, binPath, binary, 0o755); err != nil {
		return err
	}
	for _, x := range extras {
		if err := tarFile(tw, x, x, 0o644); err != nil {
			return err
		}
	}
	return nil
}
func tarFile(tw *tar.Writer, path, name string, mode int64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	h := &tar.Header{Name: name, Mode: mode, Size: st.Size(), ModTime: time.Unix(0, 0)}
	if err := tw.WriteHeader(h); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}
func writeZip(dest, binPath, binary string, extras []string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	if err := zipFile(zw, binPath, binary, 0o755); err != nil {
		return err
	}
	for _, x := range extras {
		if err := zipFile(zw, x, x, 0o644); err != nil {
			return err
		}
	}
	return nil
}
func zipFile(zw *zip.Writer, path, name string, mode os.FileMode) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	h.SetMode(mode)
	h.SetModTime(time.Unix(0, 0))
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
func modulePath() (string, error) {
	b, err := exec.Command("go", "list", "-m").Output()
	if err != nil {
		return "", fmt.Errorf("resolve module path: %w", err)
	}
	m := strings.TrimSpace(string(b))
	if m == "" {
		return "", fmt.Errorf("empty module path")
	}
	return m, nil
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "releasepack:", err); os.Exit(1) }
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
