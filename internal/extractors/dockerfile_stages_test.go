package extractors

import (
	"maps"
	"testing"
)

func TestDockerfileStagesFinalStageLines(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected []int
	}{
		{
			name:     "single stage",
			lines:    []string{"FROM alpine:3.18", "RUN apk add curl"},
			expected: []int{0},
		},
		{
			name:     "stage consumed via COPY --from is not final",
			lines:    []string{"FROM golang AS build", "RUN go build", "FROM alpine", "COPY --from=build /app /app"},
			expected: []int{2},
		},
		{
			name:     "unconsumed stage that is not the last one is final",
			lines:    []string{"FROM golang AS base", "FROM distroless AS prod", "COPY --from=base /app /app", "FROM golangci-lint AS test"},
			expected: []int{1, 3},
		},
		{
			name:     "stage built from another stage resolves to that stage's image",
			lines:    []string{"FROM node AS base", "FROM base AS prod"},
			expected: []int{0},
		},
		{
			name:     "stage names are case-insensitive",
			lines:    []string{"from golang as Build", "FROM alpine", "COPY --from=build /app /app"},
			expected: []int{1},
		},
		{
			name:     "stage consumed by index",
			lines:    []string{"FROM golang", "FROM alpine", "COPY --from=0 /app /app"},
			expected: []int{1},
		},
		{
			name:     "stage consumed via RUN --mount",
			lines:    []string{"FROM golang AS deps", "FROM alpine", "RUN --mount=type=bind,from=deps,target=/deps ls /deps"},
			expected: []int{1},
		},
		{
			name:     "scratch stage ships the images it copies from",
			lines:    []string{"FROM golang AS build", "FROM alpine AS certs", "FROM scratch", "COPY --from=build /app /app", "COPY --from=certs /etc/ssl /etc/ssl"},
			expected: []int{0, 1},
		},
		{
			name:     "scratch stage ships images copied through other scratch stages and its parents",
			lines:    []string{"FROM golang AS build", "FROM scratch AS binary", "COPY --from=build /app /app", "FROM alpine AS certs", "FROM binary AS final", "COPY --from=certs /etc/ssl /etc/ssl"},
			expected: []int{0, 3},
		},
		{
			name:     "COPY --from an external image is not a stage reference",
			lines:    []string{"FROM alpine", "COPY --from=nginx:latest /etc/nginx /etc/nginx"},
			expected: []int{0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stages := &dockerfileStages{}
			for lineNum, line := range test.lines {
				stages.parseLine(line, lineNum)
			}

			expected := make(map[int]bool)
			for _, line := range test.expected {
				expected[line] = true
			}

			if result := stages.finalStageLines(); !maps.Equal(result, expected) {
				t.Errorf("expected final stage lines %v, got %v", expected, result)
			}
		})
	}
}
