package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var composeFile string
	var baseURL string
	var timeoutSeconds int
	flag.StringVar(&composeFile, "compose-file", "deployments/docker-compose.yml", "docker compose file path")
	flag.StringVar(&baseURL, "base-url", "http://localhost:8080", "API base URL")
	flag.IntVar(&timeoutSeconds, "timeout-seconds", 180, "timeout in seconds")
	flag.Parse()

	if strings.TrimSpace(composeFile) == "" {
		fatalf("compose-file is required")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		fatalf("base-url is required")
	}
	if timeoutSeconds <= 0 {
		fatalf("timeout-seconds must be > 0")
	}

	timeout := time.Duration(timeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cwd, err := os.Getwd()
	if err != nil {
		fatalf("getwd: %v", err)
	}
	envPath := filepath.Join(cwd, ".env")
	backupPath := filepath.Join(cwd, ".env.bak.e2e")

	restoreEnv, err := writeTempEnv(envPath, backupPath)
	if err != nil {
		fatalf("prepare .env: %v", err)
	}
	defer func() {
		fmt.Println("==> restoring .env")
		if err := restoreEnv(); err != nil {
			fmt.Fprintf(os.Stderr, "WARN restore .env: %v\n", err)
		}
	}()

	fmt.Println("==> docker compose up")
	if err := runCmd(ctx, "docker", "compose", "-f", composeFile, "up", "-d", "--build"); err != nil {
		fatalf("docker compose up failed: %v", err)
	}

	httpc := &http.Client{Timeout: 25 * time.Second}

	fmt.Println("==> waiting for /readyz")
	if err := waitReady(ctx, httpc, baseURL, 2*time.Second); err != nil {
		fatalf("%v", err)
	}

	fmt.Println("==> register")
	email := "e2e+" + newUUID() + "@example.com"
	accessToken, err := register(ctx, httpc, baseURL, email, "E2E User", "P@ssw0rd!")
	if err != nil {
		fatalf("register: %v", err)
	}

	fmt.Println("==> create report")
	reportID, err := createReport(ctx, httpc, baseURL, accessToken, "E2E tus", "upload via tusd")
	if err != nil {
		fatalf("create report: %v", err)
	}
	fmt.Printf("report_id=%s\n", reportID)

	fmt.Println("==> tus create upload")
	meta := strings.Join([]string{
		"report_id " + b64(reportID),
		"filename " + b64("screen.png"),
		"content_type " + b64("image/png"),
		"idempotency_key " + b64("idem-"+newUUID()),
	}, ",")

	uploadURL, err := tusCreate(ctx, httpc, composeFile, baseURL+"/api/v1/uploads", accessToken, 4, meta)
	if err != nil {
		uploadURL, err = tusCreate(ctx, httpc, composeFile, baseURL+"/api/v1/uploads/", accessToken, 4, meta)
	}
	if err != nil {
		fatalf("tus create failed: %v", err)
	}
	fmt.Printf("upload_url=%s\n", uploadURL)

	fmt.Println("==> tus PATCH data")
	if err := tusPatch(ctx, httpc, uploadURL, accessToken, []byte{0x89, 0x50, 0x4E, 0x47}); err != nil {
		fatalf("tus patch: %v", err)
	}

	fmt.Println("==> wait for finalize + list attachments")
	attID, downloadURL, err := waitAttachment(ctx, httpc, baseURL, accessToken, reportID, 2*time.Second)
	if err != nil {
		fatalf("%v", err)
	}
	fmt.Printf("OK attachment_id=%s download_url=%s\n", attID, downloadURL)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeTempEnv(envPath, backupPath string) (restore func() error, _ error) {
	backupExists := false
	if _, err := os.Stat(envPath); err == nil {
		if err := copyFile(envPath, backupPath); err != nil {
			return nil, err
		}
		backupExists = true
	}

	tmp := strings.TrimSpace(`
APP_ENV=local
HTTP_ADDR=:8080
LOG_LEVEL=info
CORS_ALLOWED_ORIGINS=*
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20

POSTGRES_USER=bug
POSTGRES_PASSWORD=bug
POSTGRES_DB=bugdb
POSTGRES_PORT=5432

DATABASE_URL=postgres://bug:bug@postgres:5432/bugdb?sslmode=disable

JWT_ISSUER=bug-report-service
JWT_SECRET=e2e-secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

S3_ENDPOINT=http://minio:9000
S3_REGION=us-east-1
S3_BUCKET=bug-attachments
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin

HTTP_PORT=8080
MINIO_PORT=9000
MINIO_CONSOLE_PORT=9001
`) + "\n"

	if err := os.WriteFile(envPath, []byte(tmp), 0o644); err != nil {
		return nil, err
	}

	return func() error {
		if backupExists {
			if err := os.Rename(backupPath, envPath); err == nil {
				return nil
			}
			// if rename fails (e.g. across devices), fall back to copy
			if err := copyFile(backupPath, envPath); err != nil {
				return err
			}
			_ = os.Remove(backupPath)
			return nil
		}
		_ = os.Remove(envPath)
		return nil
	}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func waitReady(ctx context.Context, httpc *http.Client, baseURL string, interval time.Duration) error {
	type readyResp struct {
		Ready bool `json:"ready"`
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/readyz", nil)
		req.Header.Set("Accept", "application/json")
		resp, err := httpc.Do(req)
		if err == nil && resp != nil {
			var rr readyResp
			_ = json.NewDecoder(resp.Body).Decode(&rr)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK && rr.Ready {
				return nil
			}
		} else if resp != nil {
			_ = resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for readyz")
		case <-t.C:
		}
	}
}

func register(ctx context.Context, httpc *http.Client, baseURL, email, name, password string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"email":    email,
		"name":     name,
		"password": password,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return "", errors.New("no access_token from register")
	}
	return out.AccessToken, nil
}

func createReport(ctx context.Context, httpc *http.Client, baseURL, accessToken, title, description string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"title":       title,
		"description": description,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/reports", bytes.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.ID) == "" {
		return "", errors.New("no report id")
	}
	return out.ID, nil
}

func tusCreate(ctx context.Context, httpc *http.Client, composeFile, url, accessToken string, uploadLen int, meta string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Upload-Length", fmt.Sprintf("%d", uploadLen))
	req.Header.Set("Upload-Metadata", meta)
	req.Header.Set("Expect", "")

	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		allow := resp.Header.Get("Allow")
		if allow != "" {
			fmt.Printf("Allow: %s\n", allow)
		}
		_ = dumpDockerLogsBestEffort(ctx, composeFile)
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("tus create failed: %d url=%s body=%s", resp.StatusCode, url, strings.TrimSpace(string(b)))
	}
	loc := strings.TrimSpace(resp.Header.Get("Location"))
	if loc == "" {
		return "", errors.New("no Location header from tus create")
	}
	if strings.HasPrefix(loc, "/") {
		// keep same scheme/host as base URL (url contains full base)
		// url is like http://host/api/v1/uploads
		prefix := strings.SplitN(url, "/api/", 2)[0]
		return prefix + loc, nil
	}
	return loc, nil
}

func dumpDockerLogsBestEffort(ctx context.Context, composeFile string) error {
	fmt.Println("==> docker logs (api) tail")
	c2, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c2, "docker", "compose", "-f", composeFile, "logs", "--no-color", "--tail", "120", "api")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return nil
}

func tusPatch(ctx context.Context, httpc *http.Client, uploadURL, accessToken string, data []byte) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, uploadURL, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Upload-Offset", "0")
	req.Header.Set("Content-Type", "application/offset+octet-stream")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("Expect", "")

	resp, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("unexpected patch status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if got := strings.TrimSpace(resp.Header.Get("Upload-Offset")); got != fmt.Sprintf("%d", len(data)) {
		return fmt.Errorf("unexpected upload offset after patch: %s", got)
	}
	return nil
}

func waitAttachment(ctx context.Context, httpc *http.Client, baseURL, accessToken, reportID string, interval time.Duration) (attID, downloadURL string, err error) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		attID, downloadURL, ok := tryListAttachments(ctx, httpc, baseURL, accessToken, reportID)
		if ok {
			return attID, downloadURL, nil
		}
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("timeout waiting for attachment finalize")
		case <-t.C:
		}
	}
}

func tryListAttachments(ctx context.Context, httpc *http.Client, baseURL, accessToken, reportID string) (attID, downloadURL string, ok bool) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/reports/"+reportID+"/attachments", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := httpc.Do(req)
	if err != nil {
		return "", "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", false
	}
	var out struct {
		Items []struct {
			ID          string `json:"id"`
			DownloadURL string `json:"download_url"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", false
	}
	if len(out.Items) < 1 {
		return "", "", false
	}
	id := strings.TrimSpace(out.Items[0].ID)
	dl := strings.TrimSpace(out.Items[0].DownloadURL)
	if id == "" || dl == "" {
		return "", "", false
	}
	return id, dl, true
}

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	// worst-case fallback (shouldn't happen)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}
