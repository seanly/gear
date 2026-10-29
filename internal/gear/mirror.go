package gear

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const buildkitImage = "moby/buildkit:buildx-stable-1"

// mirror is a Docker Hub mirror. BuildKit sends docker.io pulls to Host.
type mirror struct {
	Host      string
	PlainHTTP bool
}

func parseMirror(raw string) (mirror, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return mirror{}, fmt.Errorf("registry mirror is empty")
	}
	plainHTTP := false
	host := raw
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return mirror{}, fmt.Errorf("registry mirror %q: %w", raw, err)
		}
		switch u.Scheme {
		case "https":
		case "http":
			plainHTTP = true
		default:
			return mirror{}, fmt.Errorf("registry mirror %q must be https or http", raw)
		}
		if u.User != nil {
			return mirror{}, fmt.Errorf("registry mirror %q must not include credentials", raw)
		}
		if u.Path != "" && u.Path != "/" {
			return mirror{}, fmt.Errorf("registry mirror %q must be a host, for example mirror.example.com", raw)
		}
		host = u.Host
	}
	host = strings.TrimSuffix(host, "/")
	if host == "" || strings.Contains(host, "/") || !validMirrorHost(host) {
		return mirror{}, fmt.Errorf("registry mirror %q must be a host, for example mirror.example.com", raw)
	}
	return mirror{Host: host, PlainHTTP: plainHTTP}, nil
}

func validMirrorHost(host string) bool {
	for _, r := range host {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == ':' {
			continue
		}
		return false
	}
	return true
}

func (m mirror) builderName() string {
	var b strings.Builder
	b.WriteString("gear-")
	for _, r := range strings.ToLower(m.Host) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

func (m mirror) builderImage() string {
	return m.Host + "/" + buildkitImage
}

func (m mirror) config() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[registry.\"docker.io\"]\n")
	fmt.Fprintf(&b, "  mirrors = [%q]\n", m.Host)
	if m.PlainHTTP {
		fmt.Fprintf(&b, "  http = true\n\n")
		fmt.Fprintf(&b, "[registry.%q]\n", m.Host)
		fmt.Fprintf(&b, "  http = true\n")
	}
	return b.String()
}

func createBuilderArgv(name, configPath, image string) []string {
	return []string{
		"buildx", "create",
		"--name", name,
		"--driver", "docker-container",
		"--buildkitd-config", configPath,
		"--driver-opt", "image=" + image,
	}
}

func ensureMirror(r Runner, req *Request) error {
	if strings.TrimSpace(req.RegistryMirror) == "" {
		return nil
	}
	m, err := parseMirror(req.RegistryMirror)
	if err != nil {
		return err
	}
	name := m.builderName()
	req.Builder = name
	if builderExists(r, name) {
		return nil
	}
	dir, err := os.MkdirTemp("", "gear-buildkit-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "buildkitd.toml")
	if err := os.WriteFile(path, []byte(m.config()), 0o644); err != nil {
		return err
	}
	err = r.Run(nil, createBuilderArgv(name, path, m.builderImage())...)
	if err != nil && !builderExists(r, name) {
		return err
	}
	return nil
}

func builderExists(r Runner, name string) bool {
	_, err := r.Output("buildx", "inspect", name)
	return err == nil
}
