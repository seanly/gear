package gear

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const (
	// DefaultFile is used when --file is omitted.
	DefaultFile = "Dockerfile"
	// DefaultOutput is the host directory that receives the exported stage.
	DefaultOutput = "dist"
	// DebugImage is the local tag used by gear debug.
	DebugImage = "gear-debug"
	// RunImage is the local tag used by gear run.
	RunImage     = "gear-run"
	defaultShell = "/bin/sh"
)

// Runner executes the docker CLI. Tests substitute a fake.
type Runner interface {
	Run(stdin io.Reader, args ...string) error
	Output(args ...string) ([]byte, error)
	Exec(args ...string) error
}

// Request is one build or debug invocation.
type Request struct {
	File      string
	Target    string
	Output    string
	Context   string
	BuildArgs []string
	Secret    string
	CacheFrom string
	CacheTo   string
	// BuildContexts are extra named contexts, each "name=value", passed through to buildx.
	BuildContexts  []string
	Platform       string
	RegistryMirror string
	// Builder is the buildx instance selected for RegistryMirror.
	Builder string
	Shell   string
	Command []string
	TTY     bool
	// SourceDir is the host directory gear run mounts. Empty for build and debug.
	SourceDir string
	// Volumes are extra docker run -v values, host:container[:opts].
	Volumes []string
	// Env are extra docker run -e values.
	Env []string
	// Entrypoint overrides the image entrypoint when set.
	Entrypoint string
}

func (r Request) file() string {
	if r.File == "" {
		return DefaultFile
	}
	return r.File
}

func (r Request) output() string {
	if r.Output == "" {
		return DefaultOutput
	}
	return r.Output
}

func (r Request) context() string {
	if r.Context == "" {
		return "."
	}
	return r.Context
}

func (r Request) shell() string {
	if r.Shell == "" {
		return defaultShell
	}
	return r.Shell
}

// BuildArgv is `docker buildx build` exporting one stage to a directory.
func BuildArgv(r Request) []string {
	args := []string{"buildx", "build"}
	args = append(args, builderArgs(r)...)
	args = append(args, "--file", r.file())
	if r.Target != "" {
		args = append(args, "--target", r.Target)
	}
	args = append(args, "--output", "type=local,dest="+r.output())
	args = append(args, commonArgs(r)...)
	args = append(args, r.context())
	return args
}

// DebugBuildArgv loads the stage into the local engine so docker run can see it.
func DebugBuildArgv(r Request) []string {
	args := []string{"buildx", "build"}
	args = append(args, builderArgs(r)...)
	args = append(args, "--load", "-t", DebugImage, "--file", r.file())
	if r.Target != "" {
		args = append(args, "--target", r.Target)
	}
	args = append(args, commonArgs(r)...)
	args = append(args, r.context())
	return args
}

// DebugRunArgv starts a shell in the loaded stage. -t is added only on a TTY.
func DebugRunArgv(r Request) []string {
	args := []string{"run", "--rm", "-i"}
	if r.TTY {
		args = append(args, "-t")
	}
	args = append(args, "--entrypoint", r.shell(), DebugImage)
	args = append(args, r.Command...)
	return args
}

func builderArgs(r Request) []string {
	if r.Builder == "" {
		return nil
	}
	return []string{"--builder", r.Builder}
}

func commonArgs(r Request) []string {
	var args []string
	for _, v := range r.BuildArgs {
		if v == "" {
			continue
		}
		args = append(args, "--build-arg", v)
	}
	if r.Secret != "" {
		args = append(args, "--secret", r.Secret)
	}
	if r.CacheFrom != "" {
		args = append(args, "--cache-from", r.CacheFrom)
	}
	if r.CacheTo != "" {
		args = append(args, "--cache-to", r.CacheTo)
	}
	if r.Platform != "" {
		args = append(args, "--platform", r.Platform)
	}
	for _, v := range r.BuildContexts {
		if v == "" {
			continue
		}
		args = append(args, "--build-context", v)
	}
	return args
}

// validateBuildContexts rejects values that are not name=value.
// Empty entries are ignored. The value itself is not interpreted.
func validateBuildContexts(values []string) error {
	for _, v := range values {
		if v == "" {
			continue
		}
		name, _, ok := strings.Cut(v, "=")
		if !ok || name == "" {
			return fmt.Errorf("build context %q must be name=path, for example --build-context m2=/var/cache/m2", v)
		}
	}
	return nil
}

// EnsureBuildx fails when the docker CLI has no buildx plugin.
// Classic docker build cannot export a stage to a directory.
func EnsureBuildx(r Runner) error {
	if _, err := r.Output("buildx", "version"); err != nil {
		return fmt.Errorf("docker buildx is required: %w", err)
	}
	return nil
}

// Build exports the requested stage.
func prepare(r Runner, req *Request) error {
	if err := validateBuildContexts(req.BuildContexts); err != nil {
		return err
	}
	if err := EnsureBuildx(r); err != nil {
		return err
	}
	return ensureMirror(r, req)
}

// Build exports the requested stage.
func Build(r Runner, req Request) error {
	if err := prepare(r, &req); err != nil {
		return err
	}
	return r.Run(nil, BuildArgv(req)...)
}

// Debug loads the stage and replaces this process with a shell.
func Debug(r Runner, req Request) error {
	if err := prepare(r, &req); err != nil {
		return err
	}
	if err := r.Run(nil, DebugBuildArgv(req)...); err != nil {
		return err
	}
	return r.Exec(DebugRunArgv(req)...)
}

type dockerRunner struct {
	bin string
}

// NewDockerRunner looks up the docker CLI on PATH.
func NewDockerRunner() (Runner, error) {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker not found: %w", err)
	}
	return dockerRunner{bin: bin}, nil
}

func (d dockerRunner) Run(stdin io.Reader, args ...string) error {
	slog.Debug("running", "bin", d.bin, "args", args)
	cmd := exec.Command(d.bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if stdin != nil {
		cmd.Stdin = stdin
	} else {
		cmd.Stdin = os.Stdin
	}
	return asExit(cmd.Run())
}

func (d dockerRunner) Output(args ...string) ([]byte, error) {
	slog.Debug("output", "bin", d.bin, "args", args)
	cmd := exec.Command(d.bin, args...)
	out, err := cmd.CombinedOutput()
	return out, asExit(err)
}

func (d dockerRunner) Exec(args ...string) error {
	argv := append([]string{"docker"}, args...)
	slog.Debug("exec", "bin", d.bin, "args", argv)
	return syscall.Exec(d.bin, argv, os.Environ())
}

func asExit(err error) error {
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return &ExitError{Code: ee.ExitCode(), Err: err}
	}
	return err
}
