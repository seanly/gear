package gear

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
)

// ShowTargetsFile reads a Dockerfile and lists its stages and stage dependencies.
// A missing file includes an example invocation in the error.
func ShowTargetsFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%w\n  gear show targets --file <Dockerfile>", err)
	}
	defer f.Close()
	return ShowTargets(f)
}

// ShowTargets lists stages and stage dependencies from Dockerfile text.
// Each stage name is followed by indented FROM, COPY --from, and RUN --mount lines.
func ShowTargets(r io.Reader) (string, error) {
	res, err := parser.Parse(r)
	if err != nil {
		return "", err
	}
	stages, metaArgs, err := instructions.Parse(res.AST, nil)
	if err != nil {
		return "", err
	}
	if len(stages) == 0 {
		return "", fmt.Errorf("no stages in Dockerfile")
	}

	lex := shell.NewLex(res.EscapeToken)
	lex.SkipUnsetEnv = true

	meta, err := newArgs(nil, metaArgs, lex)
	if err != nil {
		return "", err
	}

	labels := make([]string, len(stages))
	for i, st := range stages {
		labels[i] = stageLabel(st, i)
	}

	var b strings.Builder
	for i, st := range stages {
		base, err := expandWord(lex, st.BaseName, meta)
		if err != nil {
			return "", err
		}
		obj := base
		if n, ok := instructions.HasStage(stages[:i], base); ok {
			obj = labels[n]
		}
		writeStage(&b, i, labels[i])
		writeDetail(&b, "FROM "+obj)

		stageArgs := map[string]*string{}
		if err := walkCommands(&b, st.Commands, i, stages, labels, meta, stageArgs, lex, ""); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func walkCommands(b *strings.Builder, cmds []instructions.Command, idx int, stages []instructions.Stage, labels []string, meta, stageArgs map[string]*string, lex *shell.Lex, prefix string) error {
	for _, cmd := range cmds {
		switch c := cmd.(type) {
		case *instructions.ArgCommand:
			if err := applyArgs(stageArgs, meta, []instructions.ArgCommand{*c}, lex); err != nil {
				return err
			}
		case *instructions.CopyCommand:
			if err := writeRef(b, prefix+"COPY --from", c.From, idx, stages, labels, stageArgs, lex); err != nil {
				return err
			}
		case *instructions.RunCommand:
			for _, m := range instructions.GetMounts(c) {
				if m == nil {
					continue
				}
				if err := writeRef(b, prefix+"RUN --mount from", m.From, idx, stages, labels, stageArgs, lex); err != nil {
					return err
				}
			}
		case *instructions.OnbuildCommand:
			inner, err := commandsIn(c.Expression)
			if err != nil {
				return err
			}
			if err := walkCommands(b, inner, idx, stages, labels, meta, stageArgs, lex, "ONBUILD "); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeRef(b *strings.Builder, phrase, raw string, idx int, stages []instructions.Stage, labels []string, args map[string]*string, lex *shell.Lex) error {
	if raw == "" {
		return nil
	}
	expanded, err := expandWord(lex, raw, args)
	if err != nil {
		return err
	}
	obj, ok := resolveRef(expanded, stages[:idx+1], labels[:idx+1])
	if !ok {
		return nil
	}
	writeDetail(b, phrase+" "+obj)
	return nil
}

func writeStage(b *strings.Builder, index int, name string) {
	if index > 0 {
		b.WriteByte('\n')
	}
	fmt.Fprintf(b, "%s\n", name)
}

func writeDetail(b *strings.Builder, line string) {
	fmt.Fprintf(b, "  %s\n", line)
}

func resolveRef(ref string, stages []instructions.Stage, labels []string) (string, bool) {
	if ref == "" {
		return "", false
	}
	if isDigits(ref) {
		n, _ := strconv.Atoi(ref)
		if n >= 0 && n < len(stages) {
			return labels[n], true
		}
		return ref, true
	}
	if n, ok := instructions.HasStage(stages, ref); ok {
		return labels[n], true
	}
	if strings.ContainsAny(ref, ":/@") {
		return "", false
	}
	return ref, true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func stageLabel(st instructions.Stage, index int) string {
	if st.Name == "" {
		return fmt.Sprintf("#%d", index)
	}
	if name := originalAS(st.SourceCode); name != "" {
		return name
	}
	return st.Name
}

func originalAS(source string) string {
	fields := strings.Fields(source)
	if len(fields) >= 2 && strings.EqualFold(fields[len(fields)-2], "as") {
		return fields[len(fields)-1]
	}
	return ""
}

func commandsIn(expr string) ([]instructions.Command, error) {
	res, err := parser.Parse(strings.NewReader("FROM scratch\n" + expr + "\n"))
	if err != nil {
		return nil, err
	}
	stages, _, err := instructions.Parse(res.AST, nil)
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return nil, nil
	}
	return stages[0].Commands, nil
}

func newArgs(inherit map[string]*string, cmds []instructions.ArgCommand, lex *shell.Lex) (map[string]*string, error) {
	args := map[string]*string{}
	if err := applyArgs(args, inherit, cmds, lex); err != nil {
		return nil, err
	}
	return args, nil
}

func applyArgs(dst, inherit map[string]*string, cmds []instructions.ArgCommand, lex *shell.Lex) error {
	for _, cmd := range cmds {
		type update struct {
			key string
			val *string
		}
		var ups []update
		for _, arg := range cmd.Args {
			if arg.Value != nil {
				expanded, err := expandWord(lex, *arg.Value, dst)
				if err != nil {
					return err
				}
				v := expanded
				ups = append(ups, update{key: arg.Key, val: &v})
				continue
			}
			if inherit != nil {
				if prev, ok := inherit[arg.Key]; ok {
					ups = append(ups, update{key: arg.Key, val: prev})
					continue
				}
			}
			ups = append(ups, update{key: arg.Key, val: nil})
		}
		for _, u := range ups {
			dst[u.key] = u.val
		}
	}
	return nil
}

func expandWord(lex *shell.Lex, word string, args map[string]*string) (string, error) {
	if word == "" || !strings.Contains(word, "$") {
		return word, nil
	}
	got, _, err := lex.ProcessWord(word, shell.EnvsFromSlice(envPairs(args)))
	return got, err
}

func envPairs(args map[string]*string) []string {
	pairs := make([]string, 0, len(args))
	for k, v := range args {
		if v == nil {
			continue
		}
		pairs = append(pairs, k+"="+*v)
	}
	return pairs
}
