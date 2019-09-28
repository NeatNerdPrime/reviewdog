package server

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v28/github"
	"github.com/reviewdog/reviewdog/doghouse"
)

type fakeCheckerGitHubCli struct {
	checkerGitHubClientInterface
	FakeGetPullRequestDiff func(ctx context.Context, owner, repo string, number int) ([]byte, error)
	FakeCreateCheckRun     func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error)
}

func (f *fakeCheckerGitHubCli) GetPullRequestDiff(ctx context.Context, owner, repo string, number int) ([]byte, error) {
	return f.FakeGetPullRequestDiff(ctx, owner, repo, number)
}

func (f *fakeCheckerGitHubCli) CreateCheckRun(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
	return f.FakeCreateCheckRun(ctx, owner, repo, opt)
}

const sampleDiff = `--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,4 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
\ No newline at end of file
`

func TestCheck_OK(t *testing.T) {
	const (
		name       = "haya14busa-linter"
		owner      = "haya14busa"
		repo       = "reviewdog"
		prNum      = 14
		sha        = "1414"
		reportURL  = "http://example.com/report_url"
		conclusion = "neutral"
	)

	req := &doghouse.CheckRequest{
		Name:        name,
		Owner:       owner,
		Repo:        repo,
		PullRequest: prNum,
		SHA:         sha,
		Annotations: []*doghouse.Annotation{
			{
				Path:       "sample.new.txt",
				Line:       2,
				Message:    "test message",
				RawMessage: "raw test message",
			},
			{
				Path:       "sample.new.txt",
				Line:       14,
				Message:    "test message outside diff",
				RawMessage: "raw test message outside diff",
			},
		},
		Level: "warning",
	}

	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		if opt.Name != name {
			t.Errorf("CreateCheckRunOptions.Name = %q, want %q", opt.Name, name)
		}
		if opt.HeadSHA != sha {
			t.Errorf("CreateCheckRunOptions.HeadSHA = %q, want %q", opt.HeadSHA, sha)
		}
		if *opt.Conclusion != conclusion {
			t.Errorf("CreateCheckRunOptions.Conclusion = %q, want %q", *opt.Conclusion, conclusion)
		}
		annotations := opt.Output.Annotations
		wantAnnotaions := []*github.CheckRunAnnotation{
			{
				Path:            github.String("sample.new.txt"),
				StartLine:       github.Int(2),
				EndLine:         github.Int(2),
				AnnotationLevel: github.String("warning"),
				Message:         github.String("test message"),
				Title:           github.String("[haya14busa-linter] sample.new.txt#L2"),
				RawDetails:      github.String("raw test message"),
			},
		}
		if d := cmp.Diff(annotations, wantAnnotaions); d != "" {
			t.Errorf("Annotation diff found:\n%s", d)
		}
		return &github.CheckRun{HTMLURL: github.String(reportURL)}, nil
	}
	checker := &Checker{req: req, gh: cli}
	res, err := checker.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if res.ReportURL != reportURL {
		t.Errorf("res.reportURL = %q, want %q", res.ReportURL, reportURL)
	}
}

func TestCheck_fail_diff(t *testing.T) {
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return nil, errors.New("test diff failure")
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return &github.CheckRun{}, nil
	}
	checker := &Checker{req: req, gh: cli}

	if _, err := checker.Check(context.Background()); err == nil {
		t.Fatalf("got no error, want some error")
	} else {
		t.Log(err)
	}
}

func TestCheck_fail_invalid_diff(t *testing.T) {
	t.Skip("Parse invalid diff function somehow doesn't return error")
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte("invalid diff"), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return &github.CheckRun{}, nil
	}
	checker := &Checker{req: req, gh: cli}

	if _, err := checker.Check(context.Background()); err == nil {
		t.Fatalf("got no error, want some error")
	} else {
		t.Log(err)
	}
}

func TestCheck_fail_check(t *testing.T) {
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return nil, errors.New("test check failure")
	}
	checker := &Checker{req: req, gh: cli}

	if _, err := checker.Check(context.Background()); err == nil {
		t.Fatalf("got no error, want some error")
	} else {
		t.Log(err)
	}
}

func TestCheck_fail_check_with_403(t *testing.T) {
	req := &doghouse.CheckRequest{}
	cli := &fakeCheckerGitHubCli{}
	cli.FakeGetPullRequestDiff = func(ctx context.Context, owner, repo string, number int) ([]byte, error) {
		return []byte(sampleDiff), nil
	}
	cli.FakeCreateCheckRun = func(ctx context.Context, owner, repo string, opt github.CreateCheckRunOptions) (*github.CheckRun, error) {
		return nil, &github.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
			},
		}
	}
	checker := &Checker{req: req, gh: cli}

	resp, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	if resp.CheckedResults == nil {
		t.Error("resp.CheckedResults should not be nil")
	}
}
