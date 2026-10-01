package extractors

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	stageRegex          = regexp.MustCompile(`(?i)^\s*FROM\s+(?:--platform=\S+\s+)?(\S+)(?:\s+AS\s+(\S+))?`)
	stageReferenceRegex = regexp.MustCompile(`(?i)(?:--|[=,])from=([^\s,]+)`)
)

type dockerfileStage struct {
	name       string
	line       int
	parent     int // index of the stage this one is built FROM, -1 when built from an image
	scratch    bool
	sources    []int // indexes of the stages this one copies or mounts from
	referenced bool
}

// dockerfileStages tracks the build stages of a Dockerfile and whether another stage consumes them
// via FROM <stage>, COPY --from=<stage> or RUN --mount=from=<stage>.
type dockerfileStages struct {
	stages []dockerfileStage
}

func (s *dockerfileStages) parseLine(line string, lineNum int) {
	if match := stageRegex.FindStringSubmatch(line); match != nil {
		s.stages = append(s.stages, dockerfileStage{
			name:    strings.ToLower(match[2]),
			line:    lineNum,
			parent:  s.reference(match[1]),
			scratch: strings.EqualFold(match[1], "scratch"),
		})
		return
	}

	for _, match := range stageReferenceRegex.FindAllStringSubmatch(line, -1) {
		if index := s.reference(match[1]); index >= 0 {
			current := &s.stages[len(s.stages)-1]
			current.sources = append(current.sources, index)
		}
	}
}

// finalStageLines returns the lines of the FROM instructions that pull the base image of an unreferenced stage.
// Any unreferenced stage can be the shipped one: the last stage by default, any other via `docker build --target`.
func (s *dockerfileStages) finalStageLines() map[int]bool {
	lines := make(map[int]bool)
	for i, stage := range s.stages {
		if !stage.referenced {
			s.collectBaseImageLines(i, lines)
		}
	}
	return lines
}

// collectBaseImageLines adds the FROM line of the image the stage is built on. A stage built on scratch has no
// image of its own, so it ships the images of the stages it (or a stage it is built from) copies from.
func (s *dockerfileStages) collectBaseImageLines(index int, lines map[int]bool) {
	root := s.stages[index]
	sources := slices.Clone(root.sources)
	for root.parent >= 0 {
		root = s.stages[root.parent]
		sources = append(sources, root.sources...)
	}

	if !root.scratch {
		lines[root.line] = true
		return
	}
	for _, source := range sources {
		s.collectBaseImageLines(source, lines)
	}
}

// reference marks the stage named or indexed by ref as consumed and returns its index, or -1 when ref is not a stage.
func (s *dockerfileStages) reference(ref string) int {
	index := s.find(ref)
	if index >= 0 {
		s.stages[index].referenced = true
	}
	return index
}

func (s *dockerfileStages) find(ref string) int {
	ref = strings.ToLower(ref)
	if index := slices.IndexFunc(s.stages, func(stage dockerfileStage) bool { return stage.name == ref }); index >= 0 {
		return index
	}
	if index, err := strconv.Atoi(ref); err == nil && index >= 0 && index < len(s.stages) {
		return index
	}
	return -1
}
