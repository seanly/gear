package gear

import (
	"strings"
	"testing"
)

func TestParseMirror(t *testing.T) {
	m, err := parseMirror("https://mirror.example.com/")
	if err != nil {
		t.Fatal(err)
	}
	if m.Host != "mirror.example.com" || m.PlainHTTP || m.builderName() != "gear-mirror-example-com" {
		t.Fatalf("%+v %s", m, m.builderName())
	}
	if m.builderImage() != "mirror.example.com/moby/buildkit:buildx-stable-1" {
		t.Fatal(m.builderImage())
	}

	plain, err := parseMirror("http://127.0.0.1:5000")
	if err != nil {
		t.Fatal(err)
	}
	if !plain.PlainHTTP || !strings.Contains(plain.config(), "http = true") {
		t.Fatal(plain.config())
	}

	if _, err := parseMirror("mirror.example.com/library"); err == nil {
		t.Fatal("path should be rejected")
	}
}

func TestBuildCreatesMirrorBuilder(t *testing.T) {
	f := &fake{inspectMissing: true}
	err := Build(f, Request{Target: "result", RegistryMirror: "mirror.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.config, `mirrors = ["mirror.example.com"]`) {
		t.Fatalf("config %s", f.config)
	}
	create := strings.Join(f.calls[2], " ")
	if !strings.Contains(create, "buildx create --name gear-mirror-example-com --driver docker-container") {
		t.Fatalf("create %s", create)
	}
	if !strings.Contains(create, "--driver-opt image=mirror.example.com/moby/buildkit:buildx-stable-1") {
		t.Fatalf("create %s", create)
	}
	build := strings.Join(f.calls[3], " ")
	want := "buildx build --builder gear-mirror-example-com --file Dockerfile --target result --output type=local,dest=dist ."
	if build != want {
		t.Fatalf("build %s", build)
	}
}

func TestBuildReusesMirrorBuilder(t *testing.T) {
	f := &fake{}
	if err := Build(f, Request{RegistryMirror: "mirror.example.com"}); err != nil {
		t.Fatal(err)
	}
	for _, call := range f.calls {
		if len(call) > 1 && call[1] == "create" {
			t.Fatalf("builder already exists: %v", f.calls)
		}
	}
}
