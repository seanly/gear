package gear

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const runInspectFormat = `{"workdir":{{json .Config.WorkingDir}},"env":{{json .Config.Env}}}`

const dockerSocketMount = "/var/run/docker.sock:/var/run/docker.sock"

// runSpec is the bind mount gear run assembles from an image inspect.
type runSpec struct {
	host    string
	dest    string
	runArgs []string
	env     []string
	socket  bool
	uid     int
	gid     int
}

// RunLoadArgv loads the stage for gear run. The tag is gear-run, not gear-debug.
func RunLoadArgv(r Request) []string {
	args := []string{"buildx", "build"}
	args = append(args, builderArgs(r)...)
	args = append(args, "--load", "-t", RunImage, "--file", r.file())
	if r.Target != "" {
		args = append(args, "--target", r.Target)
	}
	args = append(args, commonArgs(r)...)
	args = append(args, r.context())
	return args
}

// RunArgv is docker run with the source directory mounted at spec.dest.
// The image entrypoint is kept unless Request.Entrypoint is set.
func RunArgv(r Request, spec runSpec) []string {
	args := []string{"run", "--rm", "-i"}
	if r.TTY {
		args = append(args, "-t")
	}
	args = append(args, "-v", spec.host+":"+spec.dest)
	if spec.socket {
		args = append(args, "-v", dockerSocketMount)
	}
	args = append(args, "-w", spec.dest)
	args = append(args,
		"-e", "GEAR_UID="+strconv.Itoa(spec.uid),
		"-e", "GEAR_GID="+strconv.Itoa(spec.gid),
		"-e", "DAPPER_UID="+strconv.Itoa(spec.uid),
		"-e", "DAPPER_GID="+strconv.Itoa(spec.gid),
	)
	for _, name := range spec.env {
		if name == "" {
			continue
		}
		args = append(args, "-e", name)
	}
	for _, name := range r.Env {
		if name == "" {
			continue
		}
		args = append(args, "-e", name)
	}
	args = append(args, spec.runArgs...)
	for _, v := range r.Volumes {
		if v == "" {
			continue
		}
		args = append(args, "-v", v)
	}
	if r.Entrypoint != "" {
		args = append(args, "--entrypoint", r.Entrypoint)
	}
	args = append(args, RunImage)
	args = append(args, r.Command...)
	return args
}

// Mount loads the stage, reads GEAR_ and DAPPER_ from the image, and replaces
// this process with docker run. GEAR_ wins when both prefixes are set.
func Mount(r Runner, req Request) error {
	if err := validateVolumes(req.Volumes); err != nil {
		return err
	}
	if err := prepare(r, &req); err != nil {
		return err
	}
	if err := r.Run(nil, RunLoadArgv(req)...); err != nil {
		return err
	}
	raw, err := r.Output("image", "inspect", "-f", runInspectFormat, RunImage)
	if err != nil {
		return err
	}
	spec, err := specFromImage(raw, req.SourceDir, os.Getuid(), os.Getgid())
	if err != nil {
		return err
	}
	return r.Exec(RunArgv(req, spec)...)
}

func specFromImage(raw []byte, sourceDir string, uid, gid int) (runSpec, error) {
	var cfg struct {
		Workdir string   `json:"workdir"`
		Env     []string `json:"env"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return runSpec{}, fmt.Errorf("read image config: %w", err)
	}
	env := imageEnv(cfg.Env)
	dest := firstEnv(env, "GEAR_SOURCE", "DAPPER_SOURCE")
	if dest == "" {
		dest = strings.TrimSpace(cfg.Workdir)
	}
	if dest == "" || dest == "/" {
		return runSpec{}, fmt.Errorf("mount destination must be a directory other than /, set GEAR_SOURCE or WORKDIR")
	}
	cp := firstEnv(env, "GEAR_CP", "DAPPER_CP")
	if cp == "" {
		cp = "."
	}
	return runSpec{
		host:    hostSource(sourceDir, cp),
		dest:    dest,
		runArgs: strings.Fields(firstEnv(env, "GEAR_RUN_ARGS", "DAPPER_RUN_ARGS")),
		env:     strings.Fields(firstEnv(env, "GEAR_ENV", "DAPPER_ENV")),
		socket:  firstEnv(env, "GEAR_DOCKER_SOCKET", "DAPPER_DOCKER_SOCKET") == "true",
		uid:     uid,
		gid:     gid,
	}, nil
}

func imageEnv(items []string) map[string]string {
	out := make(map[string]string, len(items))
	for _, item := range items {
		key, val, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func firstEnv(env map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(env[key]); v != "" {
			return v
		}
	}
	return ""
}

func hostSource(sourceDir, cp string) string {
	if filepath.IsAbs(cp) {
		return filepath.Clean(cp)
	}
	return filepath.Clean(filepath.Join(sourceDir, cp))
}

func validateVolumes(values []string) error {
	for _, v := range values {
		if v == "" {
			continue
		}
		host, rest, ok := strings.Cut(v, ":")
		container, _, _ := strings.Cut(rest, ":")
		if !ok || host == "" || container == "" {
			return fmt.Errorf("volume %q must be host:container, for example --volume /var/cache/m2:/root/.m2", v)
		}
	}
	return nil
}
