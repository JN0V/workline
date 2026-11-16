package gitlab

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// gathered is the pattern of the job names whose tally the weekly sample
// gathers (the `.workline:tallies` step's jq test).
func gathered(t *testing.T) *regexp.Regexp {
	t.Helper()
	block := strings.Join(jobBlock(t, ".workline:tallies"), "\n")
	m := regexp.MustCompile(`\.name \| test\("([^"]+)"\)`).FindStringSubmatch(block)
	if m == nil {
		t.Fatal("the pattern of the jobs whose tally is gathered was not found")
	}
	re, err := regexp.Compile(m[1])
	if err != nil {
		t.Fatal(err)
	}
	return re
}

// judgesOfTally are the judge jobs whose apply job keeps a tally.
func judgesOfTally(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var judges []string
	for _, line := range strings.Split(string(data), "\n") {
		m := regexp.MustCompile(`^(workline:(?:[a-z-]+:)*)apply:`).FindStringSubmatch(line)
		if m == nil {
			continue
		}
		apply := strings.Join(jobBlock(t, m[1]+"apply"), "\n")
		if !strings.Contains(apply, "*tally") {
			continue
		}
		judge := m[1] + "judge"
		if regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(judge) + `:`).Match(data) {
			judges = append(judges, judge)
		}
	}
	if len(judges) == 0 {
		t.Fatal("no judge job whose apply job keeps a tally found")
	}
	return judges
}

// GitLab cancelled pipeline
// Given a GitLab pipeline cancelled after its judge job ran,
// When the weekly sample runs,
// Then that judge's tokens are counted: the judge job writes and keeps its
// own tally, and the weekly sample gathers it.
func TestJudgeJobKeepsItsOwnTally(t *testing.T) {
	re := gathered(t)
	for _, judge := range judgesOfTally(t) {
		block := strings.Join(jobBlock(t, judge), "\n")
		if !strings.Contains(block, "*tally") && !strings.Contains(block, "sample --tally") {
			t.Errorf("%s writes no tally of its own: a pipeline cancelled before its apply job loses what it spent", judge)
		}
		if !strings.Contains(block, "workline-tally.jsonl") {
			t.Errorf("%s does not keep workline-tally.jsonl as an artifact", judge)
		}
		if !re.MatchString(judge) {
			t.Errorf("the weekly sample does not gather the tally of %s (pattern %s)", judge, re)
		}
	}
}

// GitLab sample read fails
// Given `sample:read` fails after calling the agent,
// When the next week's sample runs,
// Then that failed read's tokens are counted: the tally of
// workline:sample:read is gathered from that job, not only from the job
// after it, which a failed read never starts.
func TestSampleReadTallyGatheredFromItsOwnJob(t *testing.T) {
	if re := gathered(t); !re.MatchString("workline:sample:read") {
		t.Errorf("the weekly sample does not gather the tally of workline:sample:read (pattern %s)", re)
	}
}
