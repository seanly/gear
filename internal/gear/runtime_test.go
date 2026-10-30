package gear

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBuildArgvDefaults(t *testing.T) {
	got := strings.Join(BuildArgv(Request{Target: "result"}), " ")
	want := "buildx build --file Dockerfile --target result --output type=local,dest=dist ."
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestBuildArgvOptionalFlags(t *testing.T) {
	got := strings.Join(BuildArgv(Request{
		File:      "examples/Dockerfile",
		Target:    "result",
		Output:    "out",
		BuildArgs: []string{"TAG=1", "ARCH=amd64"},
		Secret:    "id=s",
		CacheFrom: "type=gha",
		CacheTo:   "type=gha,mode=max",
		Platform:  "linux/amd64",
	}), " ")
	want := strings.Join([]string{
		"buildx", "build", "--file", "examples/Dockerfile", "--target", "result",
		"--output", "type=local,dest=out",
		"--build-arg", "TAG=1", "--build-arg", "ARCH=amd64",
		"--secret", "id=s",
		"--cache-from", "type=gha", "--cache-to", "type=gha,mode=max",
		"--platform", "linux/amd64",
		".",
	}, " ")
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestBuildContextArgv(t *testing.T) {
	got := strings.Join(BuildArgv(Request{
		Target: "result",
		BuildContexts: []string{
			"",
			"m2=/var/cache/jenkins/java/maven/repository",
			"npm=/var/cache/jenkins/nodejs20",
		},
	}), " ")
	want := strings.Join([]string{
		"buildx", "build", "--file", "Dockerfile", "--target", "result",
		"--output", "type=local,dest=dist",
		"--build-context", "m2=/var/cache/jenkins/java/maven/repository",
		"--build-context", "npm=/var/cache/jenkins/nodejs20",
		".",
	}, " ")
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestBuildContextOmitted(t *testing.T) {
	got := BuildArgv(Request{Target: "result"})
	if strings.Contains(strings.Join(got, " "), "--build-context") {
		t.Fatalf("argv %v", got)
	}
}

func TestDebugBuildContext(t *testing.T) {
	build := strings.Join(DebugBuildArgv(Request{
		Target:        "build",
		BuildContexts: []string{"src=."},
	}), " ")
	want := "buildx build --load -t gear-debug --file Dockerfile --target build --build-context src=. ."
	if build != want {
		t.Fatalf("build %s", build)
	}
}

func TestBuildContextRejectedBeforeDocker(t *testing.T) {
	f := &fake{}
	for _, v := range []string{"nopath", "=onlyvalue"} {
		err := Build(f, Request{Target: "result", BuildContexts: []string{v}})
		if err == nil || !strings.Contains(err.Error(), "must be name=path") {
			t.Fatalf("context %q: %v", v, err)
		}
		err = Debug(f, Request{Target: "build", BuildContexts: []string{v}})
		if err == nil || !strings.Contains(err.Error(), "must be name=path") {
			t.Fatalf("debug context %q: %v", v, err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("docker called: %v", f.calls)
	}
}

func TestDebugArgv(t *testing.T) {
	build := strings.Join(DebugBuildArgv(Request{Target: "build", Secret: "id=s"}), " ")
	if build != "buildx build --load -t gear-debug --file Dockerfile --target build --secret id=s ." {
		t.Fatalf("build %s", build)
	}
	run := strings.Join(DebugRunArgv(Request{TTY: true, Command: []string{"ls", "-la"}}), " ")
	if run != "run --rm -i -t --entrypoint /bin/sh gear-debug ls -la" {
		t.Fatalf("run %s", run)
	}
	plain := DebugRunArgv(Request{})
	if strings.Contains(strings.Join(plain, " "), " -t ") {
		t.Fatalf("non-tty should omit -t: %v", plain)
	}
}

func TestBuildRequiresBuildxAndKeepsExitCode(t *testing.T) {
	missing := &fake{outputErr: errors.New("buildx: not found")}
	if err := Build(missing, Request{}); err == nil || !strings.Contains(err.Error(), "docker buildx is required") {
		t.Fatalf("missing buildx: %v", err)
	}

	fail := &fake{runErr: &ExitError{Code: 2, Err: errors.New("exit status 2")}}
	err := Build(fail, Request{Target: "result"})
	var ee *ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 2 {
		t.Fatalf("exit: %v", err)
	}
	if strings.Join(fail.calls[1], " ") != "buildx build --file Dockerfile --target result --output type=local,dest=dist ." {
		t.Fatalf("calls %v", fail.calls)
	}
}

func TestDebugLoadsThenRuns(t *testing.T) {
	f := &fake{}
	if err := Debug(f, Request{Target: "build", TTY: true}); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 3 {
		t.Fatalf("calls %v", f.calls)
	}
	if f.calls[2][0] != "run" {
		t.Fatalf("last %v", f.calls[2])
	}
}

func TestDebugStopsWhenBuildFails(t *testing.T) {
	f := &fake{runErr: &ExitError{Code: 1, Err: errors.New("exit status 1")}}
	if err := Debug(f, Request{Target: "build"}); err == nil {
		t.Fatal("expected build failure")
	}
	for _, call := range f.calls {
		if len(call) > 0 && call[0] == "run" {
			t.Fatalf("shell should not start: %v", f.calls)
		}
	}
}

type fake struct {
	calls          [][]string
	outputErr      error
	runErr         error
	inspectMissing bool
	config         string
	imageInspect   string
}

func (f *fake) Run(_ io.Reader, args ...string) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	for i, arg := range args {
		if arg == "--buildkitd-config" && i+1 < len(args) {
			b, err := os.ReadFile(args[i+1])
			if err != nil {
				return err
			}
			f.config = string(b)
		}
	}
	return f.runErr
}

func (f *fake) Output(args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
		if f.imageInspect == "" {
			return nil, errors.New("image inspect not stubbed")
		}
		return []byte(f.imageInspect), nil
	}
	if len(args) >= 2 && args[1] == "inspect" && f.inspectMissing {
		return nil, errors.New("no builder")
	}
	if f.outputErr != nil {
		return nil, f.outputErr
	}
	return []byte("github.com/docker/buildx v0.0.0\n"), nil
}

func (f *fake) Exec(args ...string) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	return nil
}
