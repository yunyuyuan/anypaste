// Command anypaste is a tiny, dependency-free CLI for an anypaste server.
//
// It talks to the server over plain HTTP: unary RPCs use the Connect protocol
// (a single JSON POST per call), and files use the /file/* endpoints. Only the
// Go standard library is used, so it cross-compiles to a small static binary.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"yunyuyuan/anypaste/internal/uploadproto"

	"golang.org/x/term"
)

const usage = `anypaste - tiny CLI for an anypaste server

Usage:
  anypaste login   [--server URL] [--password PW]   Log in (session lasts 1h)
  anypaste ls      [--server URL]                   List pastes
  anypaste up      [--server URL] [-m TEXT] [FILE]
                                                    Create a paste, optionally uploading FILE
  anypaste down ID [--server URL] [-o OUT]          Download the file of paste ID
  anypaste logout                                   Forget the stored token
  anypaste help                                     Show this help

Config is stored at:
  <user-config-dir>/anypaste/config.json   (server URL + token)

Environment:
  ANYPASTE_SERVER     default server URL
  ANYPASTE_PASSWORD   password for "login" (skips the prompt)

Examples:
  anypaste login --server http://localhost:8080
  echo hi | anypaste up -m -            # read content from stdin
  anypaste up -m "a note"
  anypaste up ./report.pdf
  anypaste up -m "with file" ./report.pdf
  anypaste ls
  anypaste down AbC123 -o ./report.pdf
`

type config struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "anypaste", "config.json"), nil
}

func loadConfig() config {
	var c config
	p, err := configPath()
	if err != nil {
		return c
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	return c
}

func saveConfig(c config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// 0600: the token is a credential
	return os.WriteFile(p, data, 0o600)
}

// resolveServer picks the server URL from the flag, then config, then env, and
// normalizes it: the user supplies just the host (e.g. https://paste.example.com)
// and we append the /api prefix every endpoint lives under. An explicit /api
// (the old form) is kept as-is so existing configs keep working.
func resolveServer(flagVal string, c config) string {
	s := flagVal
	if s == "" {
		s = c.Server
	}
	if s == "" {
		s = os.Getenv("ANYPASTE_SERVER")
	}
	s = strings.TrimRight(s, "/")
	if s == "" {
		return ""
	}
	if !strings.HasSuffix(s, "/api") {
		s += "/api"
	}
	return s
}

// --- Connect protocol (unary JSON) -----------------------------------------

type connectError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// callRPC POSTs req to a Connect unary procedure and decodes the reply.
func callRPC(server, token, procedure string, req, out any) error {
	if server == "" {
		return fmt.Errorf("no server configured (run `anypaste login --server URL`)")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequest(http.MethodPost, server+procedure, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
	}()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var ce connectError
		if json.Unmarshal(data, &ce) == nil && ce.Message != "" {
			if ce.Code == "unauthenticated" {
				return fmt.Errorf("%s (run `anypaste login`)", ce.Message)
			}
			return fmt.Errorf("%s", ce.Message)
		}
		return fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

// --- message shapes (protobuf JSON: int64 is encoded as a string) ----------

type pasteItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	FileName string `json:"fileName"`
}

type listResp struct {
	List []pasteItem `json:"list"`
}

type createReq struct {
	Content string `json:"content"`
}

type createResp struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

// --- commands ---------------------------------------------------------------

func cmdLogin(args []string) error {
	fs := newFlagSet("login")
	server := fs.String("server", "", "server URL")
	password := fs.String("password", "", "password (otherwise prompted)")
	_ = fs.Parse(args)

	c := loadConfig()
	srv := resolveServer(*server, c)
	if srv == "" {
		return fmt.Errorf("missing --server (e.g. http://localhost:8080)")
	}

	pw := *password
	if pw == "" {
		pw = os.Getenv("ANYPASTE_PASSWORD")
	}
	if pw == "" {
		var err error
		if pw, err = promptPassword(); err != nil {
			return err
		}
	}

	// CLI sessions are short-lived (1h); the server clamps this.
	body, _ := json.Marshal(map[string]any{"password": pw, "ttl_seconds": 3600})
	resp, err := http.Post(srv+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
	}()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", strings.TrimSpace(string(data)))
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &tok); err != nil || tok.Token == "" {
		return fmt.Errorf("login: unexpected response")
	}

	if err := saveConfig(config{Server: srv, Token: tok.Token}); err != nil {
		return err
	}
	fmt.Println("Logged in. Token saved.")
	return nil
}

// promptPassword reads a password from the terminal without echoing it. If
// stdin is not a terminal (e.g. piped), it falls back to reading a line.
func promptPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, "Password: ")
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr) // the Enter keypress wasn't echoed
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(line, "\r\n"), nil
}

func cmdLogout([]string) error {
	c := loadConfig()
	c.Token = ""
	if err := saveConfig(c); err != nil {
		return err
	}
	fmt.Println("Logged out.")
	return nil
}

func cmdList(args []string) error {
	fs := newFlagSet("ls")
	server := fs.String("server", "", "server URL")
	_ = fs.Parse(args)

	c := loadConfig()
	srv := resolveServer(*server, c)

	var out listResp
	if err := callRPC(srv, c.Token, "/paste.v1.PasteService/ListPastes", struct{}{}, &out); err != nil {
		return err
	}
	if len(out.List) == 0 {
		fmt.Println("(no pastes)")
		return nil
	}
	for _, it := range out.List {
		kind := "text"
		extra := oneLine(it.Content, 60)
		if it.FileName != "" {
			kind = "file"
			extra = it.FileName
		}
		fmt.Printf("%-8s  %-4s  %s\n", it.ID, kind, extra)
	}
	return nil
}

func cmdUp(args []string) error {
	fs := newFlagSet("up")
	server := fs.String("server", "", "server URL")
	message := fs.String("m", "", `text content ("-" reads stdin)`)
	_ = fs.Parse(args)

	c := loadConfig()
	srv := resolveServer(*server, c)

	var file string
	if rest := fs.Args(); len(rest) > 0 {
		file = rest[0]
	}
	if file == "" && *message == "" {
		return fmt.Errorf("nothing to upload: pass a FILE and/or -m TEXT")
	}

	content := *message
	if content == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		content = string(data)
	}
	// File paste with no message: use the file name as content
	if content == "" && file != "" {
		content = filepath.Base(file)
	}

	req := createReq{Content: content}

	var cr createResp
	if err := callRPC(srv, c.Token, "/paste.v1.PasteService/CreatePaste", req, &cr); err != nil {
		return err
	}
	if !cr.Success || cr.ID == "" {
		return fmt.Errorf("create failed")
	}

	if file != "" {
		if err := uploadFile(srv, c.Token, cr.ID, file); err != nil {
			return fmt.Errorf("paste %s created but upload failed: %w", cr.ID, err)
		}
	}
	fmt.Println(cr.ID)
	return nil
}

// maxChunkRetries bounds per-chunk retries; a flaky link converges instead of
// failing the whole upload on the first network blip.
const maxChunkRetries = 5

// uploadFile sends path to the server as a sequence of resumable chunks (see
// internal/uploadproto). Small per-request bodies clear proxies with tight
// size/timeout limits (e.g. Cloudflare free tier), and a dropped connection
// resumes from the server's offset instead of restarting.
func uploadFile(server, token, id, path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	total := info.Size()
	encName := url.QueryEscape(filepath.Base(path))

	// Resume from whatever the server already holds for this paste.
	offset, err := uploadOffset(server, token, id)
	if err != nil {
		return err
	}

	// sendChunk posts one chunk with bounded retries, re-syncing the offset from
	// the server between attempts (covers partially-received chunks and 409s).
	sendChunk := func(off, size int64) (int64, error) {
		for attempt := 0; ; attempt++ {
			next, sendErr := postChunk(server, token, id, f, off, size, total, encName)
			if sendErr == nil {
				return next, nil
			}
			if attempt >= maxChunkRetries {
				return 0, fmt.Errorf("at offset %d: %w", off, sendErr)
			}
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			if synced, serr := uploadOffset(server, token, id); serr == nil {
				off = synced
				size = min(uploadproto.ChunkSize, total-off)
			}
		}
	}

	for offset < total {
		next, err := sendChunk(offset, min(uploadproto.ChunkSize, total-offset))
		if err != nil {
			return err
		}
		offset = next
		fmt.Fprintf(os.Stderr, "\ruploading %3d%%", int(offset*100/total))
	}
	// A zero-byte file still needs one request to create and finalize it.
	if total == 0 {
		if _, err := sendChunk(0, 0); err != nil {
			return err
		}
	}
	if total > 0 {
		fmt.Fprintln(os.Stderr)
	}
	return nil
}

// uploadOffset asks the server (HEAD) how many bytes it already has for this
// paste, i.e. where to start or resume.
func uploadOffset(server, token, id string) (int64, error) {
	req, err := http.NewRequest(http.MethodHead, server+"/file/upload/"+id, nil)
	if err != nil {
		return 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return strconv.ParseInt(resp.Header.Get(uploadproto.HeaderUploadOffset), 10, 64)
}

// postChunk sends bytes [off, off+size) of f and returns the server's new
// offset. A SectionReader keeps the file streamed, never fully buffered.
func postChunk(server, token, id string, f *os.File, off, size, total int64, encName string) (int64, error) {
	var body io.Reader = http.NoBody
	if size > 0 {
		body = io.NewSectionReader(f, off, size)
	}
	req, err := http.NewRequest(http.MethodPost, server+"/file/upload/"+id, body)
	if err != nil {
		return 0, err
	}
	req.ContentLength = size
	req.Header.Set(uploadproto.HeaderUploadOffset, strconv.FormatInt(off, 10))
	req.Header.Set(uploadproto.HeaderUploadLength, strconv.FormatInt(total, 10))
	req.Header.Set(uploadproto.HeaderUploadFilename, encName)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		data, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return strconv.ParseInt(resp.Header.Get(uploadproto.HeaderUploadOffset), 10, 64)
}

func cmdDown(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: anypaste down ID [-o OUT]")
	}
	id := args[0]
	fs := newFlagSet("down")
	server := fs.String("server", "", "server URL")
	out := fs.String("o", "", "output path (default: server filename)")
	_ = fs.Parse(args[1:])

	c := loadConfig()
	srv := resolveServer(*server, c)
	if srv == "" {
		return fmt.Errorf("no server configured (run `anypaste login --server URL`)")
	}

	resp, err := http.Get(srv + "/file/download/" + id)
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	dst := *out
	if dst == "" {
		dst = filenameFromResponse(resp, id)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		err = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", dst)
	return nil
}

// --- helpers ----------------------------------------------------------------

// newFlagSet returns a flag set that prints the top-level usage on error,
// so every subcommand surfaces the same help text.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	return fs
}

func oneLine(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

func filenameFromResponse(resp *http.Response, fallback string) string {
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if name := params["filename"]; name != "" {
				return filepath.Base(name)
			}
		}
	}
	return fallback
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "login":
		err = cmdLogin(os.Args[2:])
	case "logout":
		err = cmdLogout(os.Args[2:])
	case "ls", "list":
		err = cmdList(os.Args[2:])
	case "up", "upload":
		err = cmdUp(os.Args[2:])
	case "down", "download":
		err = cmdDown(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
