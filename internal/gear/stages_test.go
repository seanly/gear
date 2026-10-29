package gear

import (
	"os"
	"strings"
	"testing"
)

func TestShowTargetsExamples(t *testing.T) {
	got := mustTargetsFile(t, "../../examples/Dockerfile")
	want := "" +
		"build\n" +
		"  FROM alpine:3.20\n" +
		"\n" +
		"result\n" +
		"  FROM scratch\n" +
		"  COPY --from build\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestShowTargetsRootDockerfile(t *testing.T) {
	got := mustTargetsFile(t, "../../Dockerfile")
	want := "" +
		"build\n" +
		"  FROM golang:1.26\n" +
		"\n" +
		"result\n" +
		"  FROM scratch\n" +
		"  COPY --from build\n" +
		"\n" +
		"packages\n" +
		"  FROM build\n" +
		"\n" +
		"dist\n" +
		"  FROM scratch\n" +
		"  COPY --from packages\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestShowTargetsSyntax(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unnamed index",
			src:  "FROM alpine\nFROM scratch\nCOPY --from=0 /bin/sh /sh\n",
			want: "#0\n  FROM alpine\n\n#1\n  FROM scratch\n  COPY --from #0\n",
		},
		{
			name: "case and flag order",
			src:  "FROM alpine AS Build\nFROM scratch AS result\nCOPY --chown=root:root --from=BUILD /x /\n",
			want: "Build\n  FROM alpine\n\nresult\n  FROM scratch\n  COPY --from Build\n",
		},
		{
			name: "comments continuation heredoc",
			src: "" +
				"FROM alpine:3.20 \\\n    AS build\n" +
				"# FROM scratch AS hidden\n" +
				"RUN <<EOF\nFROM scratch AS fake\nEOF\n" +
				"RUN <<-EOF\n\tFROM scratch AS also\n\tEOF\n" +
				"RUN <<'END'\nFROM scratch AS quoted\nEND\n" +
				"FROM scratch AS result\n",
			want: "build\n  FROM alpine:3.20\n\nresult\n  FROM scratch\n",
		},
		{
			name: "arg defaults",
			src: "" +
				"ARG TAG=3.20\n" +
				"FROM alpine:${TAG} AS build\n" +
				"FROM scratch AS result\n" +
				"ARG SRC=build\n" +
				"COPY --from=${SRC} /x /\n",
			want: "build\n  FROM alpine:3.20\n\nresult\n  FROM scratch\n  COPY --from build\n",
		},
		{
			name: "unset var stays literal",
			src:  "ARG MISSING\nFROM alpine:${MISSING} AS build\n",
			want: "build\n  FROM alpine:${MISSING}\n",
		},
		{
			name: "stage arg does not leak",
			src: "" +
				"FROM alpine AS build\n" +
				"ARG SRC=build\n" +
				"FROM scratch AS result\n" +
				"COPY --from=${SRC} /x /\n",
			want: "build\n  FROM alpine\n\nresult\n  FROM scratch\n  COPY --from ${SRC}\n",
		},
		{
			name: "redeclared meta arg",
			src: "" +
				"ARG SRC=build\n" +
				"FROM alpine AS build\n" +
				"FROM scratch AS result\n" +
				"ARG SRC\n" +
				"COPY --from=${SRC} /x /\n",
			want: "build\n  FROM alpine\n\nresult\n  FROM scratch\n  COPY --from build\n",
		},
		{
			name: "onbuild and mount",
			src: "" +
				"FROM alpine AS build\n" +
				"ONBUILD COPY --from=build /x /y\n" +
				"FROM alpine AS run\n" +
				"RUN --mount=type=bind,from=build,source=/src,target=/src echo ok\n",
			want: "build\n  FROM alpine\n  ONBUILD COPY --from build\n\nrun\n  FROM alpine\n  RUN --mount from build\n",
		},
		{
			name: "image from is not a stage dep",
			src:  "FROM alpine AS build\nCOPY --from=alpine:3.20 /bin/sh /sh\nCOPY --from=missing /x /y\n",
			want: "build\n  FROM alpine\n  COPY --from missing\n",
		},
		{
			name: "escape directive",
			src:  "# escape=`\nFROM alpine AS build\nCOPY --from=build `\n/out /\n",
			want: "build\n  FROM alpine\n  COPY --from build\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ShowTargets(strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestShowTargetsErrors(t *testing.T) {
	if _, err := ShowTargets(strings.NewReader("ARG ONLY=1\n")); err == nil || !strings.Contains(err.Error(), "no stages") {
		t.Fatalf("no from: %v", err)
	}
	if _, err := ShowTargets(strings.NewReader("FROM alpine\nINCLUDE other.Dockerfile\n")); err == nil {
		t.Fatal("INCLUDE should fail")
	}
	if _, err := ShowTargets(strings.NewReader("FROM alpine\nADD --from=build /x /y\n")); err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("ADD --from: %v", err)
	}
	if _, err := ShowTargetsFile("Dockerfile.missing"); err == nil || !strings.Contains(err.Error(), "gear show targets --file <Dockerfile>") {
		t.Fatalf("missing: %v", err)
	}
}

func mustTargetsFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ShowTargets(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return got
}
