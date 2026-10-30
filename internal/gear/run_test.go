package gear

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMountGearSourceOverridesDapper(t *testing.T) {
	f := &fake{imageInspect: mustImageInspect(t, "/src",
		"GEAR_SOURCE=/code",
		"DAPPER_SOURCE=/dapper",
		"GEAR_RUN_ARGS=-v /var/cache/m2:/root/.m2",
		"DAPPER_RUN_ARGS=-v /other:/other",
		"GEAR_ENV=APP_NAME",
		"DAPPER_ENV=OTHER",
		"GEAR_DOCKER_SOCKET=true",
	)}
	err := Mount(f, Request{
		Target:     "build",
		SourceDir:  "/repo",
		TTY:        true,
		Command:    []string{"ci"},
		Volumes:    []string{"", "/extra:/cache"},
		Env:        []string{"APP_ENV=test"},
		Entrypoint: "bash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(f.calls[1], " ") != "buildx build --load -t gear-run --file Dockerfile --target build ." {
		t.Fatalf("load %v", f.calls[1])
	}
	got := strings.Join(f.calls[3], " ")
	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())
	want := strings.Join([]string{
		"run", "--rm", "-i", "-t",
		"-v", "/repo:/code",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-w", "/code",
		"-e", "GEAR_UID=" + uid,
		"-e", "GEAR_GID=" + gid,
		"-e", "DAPPER_UID=" + uid,
		"-e", "DAPPER_GID=" + gid,
		"-e", "APP_NAME",
		"-e", "APP_ENV=test",
		"-v", "/var/cache/m2:/root/.m2",
		"-v", "/extra:/cache",
		"--entrypoint", "bash",
		"gear-run", "ci",
	}, " ")
	if got != want {
		t.Fatalf("got %s", got)
	}
	if strings.Contains(got, "/dapper") || strings.Contains(got, "/other") || strings.Contains(got, "OTHER") {
		t.Fatalf("dapper values leaked: %s", got)
	}
}

func TestMountDapperSourceAndWorkdir(t *testing.T) {
	f := &fake{imageInspect: mustImageInspect(t, "/unused", "DAPPER_SOURCE=/code", "DAPPER_CP=pkg")}
	if err := Mount(f, Request{SourceDir: "/repo", Command: []string{"ci"}}); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(f.calls[3], " ")
	if !strings.Contains(got, "-v /repo/pkg:/code -w /code") {
		t.Fatalf("got %s", got)
	}
	if strings.Contains(got, "--entrypoint") {
		t.Fatalf("image entrypoint should stay: %s", got)
	}

	workdir := &fake{imageInspect: mustImageInspect(t, "/src")}
	if err := Mount(workdir, Request{SourceDir: "/repo"}); err != nil {
		t.Fatal(err)
	}
	got = strings.Join(workdir.calls[3], " ")
	if !strings.Contains(got, "-v /repo:/src -w /src") {
		t.Fatalf("workdir %s", got)
	}
}

func TestMountAbsoluteCP(t *testing.T) {
	f := &fake{imageInspect: mustImageInspect(t, "/", "GEAR_SOURCE=/src", "GEAR_CP=/data/src")}
	if err := Mount(f, Request{SourceDir: "/repo"}); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(f.calls[3], " ")
	if !strings.Contains(got, "-v /data/src:/src") {
		t.Fatalf("got %s", got)
	}
}

func TestMountRejectsRootDestination(t *testing.T) {
	f := &fake{imageInspect: mustImageInspect(t, "/")}
	if err := Mount(f, Request{SourceDir: "/repo"}); err == nil {
		t.Fatal("expected error")
	}
	for _, call := range f.calls {
		if len(call) > 0 && call[0] == "run" {
			t.Fatalf("docker run started: %v", f.calls)
		}
	}
}

func TestMountRejectsVolumeBeforeDocker(t *testing.T) {
	f := &fake{imageInspect: mustImageInspect(t, "/src")}
	for _, v := range []string{"nocolon", ":/only", "host:"} {
		err := Mount(f, Request{SourceDir: "/repo", Volumes: []string{v}})
		if err == nil || !strings.Contains(err.Error(), "must be host:container") {
			t.Fatalf("volume %q: %v", v, err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("docker called: %v", f.calls)
	}
}

func mustImageInspect(t *testing.T, workdir string, env ...string) string {
	t.Helper()
	b, err := json.Marshal(struct {
		Workdir string   `json:"workdir"`
		Env     []string `json:"env"`
	}{workdir, env})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
