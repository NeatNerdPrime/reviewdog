package reviewdog

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/difffilter"
)

const diffContent = `--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900
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

func TestFilterCheckByAddedLines(t *testing.T) {
	results := []*CheckResult{
		{
			Path: "sample.new.txt",
			Lnum: 1,
		},
		{
			Path: "sample.new.txt",
			Lnum: 2,
		},
		{
			Path: "nonewline.new.txt",
			Lnum: 1,
		},
		{
			Path: "nonewline.new.txt",
			Lnum: 3,
		},
	}
	want := []*FilteredCheck{
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 1,
			},
			InDiff: false,
		},
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 2,
			},
			InDiff:   true,
			LnumDiff: 3,
		},
		{
			CheckResult: &CheckResult{
				Path: "nonewline.new.txt",
				Lnum: 1,
			},
			InDiff: false,
		},
		{
			CheckResult: &CheckResult{
				Path: "nonewline.new.txt",
				Lnum: 3,
			},
			InDiff:   true,
			LnumDiff: 5,
		},
	}
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContent))
	got := FilterCheck(results, filediffs, 0, "", difffilter.ModeAdded)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

// All lines that are in diff are taken into account
func TestFilterCheckByDiffContext(t *testing.T) {
	results := []*CheckResult{
		{
			Path: "sample.new.txt",
			Lnum: 1,
		},
		{
			Path: "sample.new.txt",
			Lnum: 2,
		},
		{
			Path: "sample.new.txt",
			Lnum: 3,
		},
	}
	want := []*FilteredCheck{
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 1,
			},
			InDiff:   true,
			LnumDiff: 1,
		},
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 2,
			},
			InDiff:   true,
			LnumDiff: 3,
		},
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 3,
			},
			InDiff:   true,
			LnumDiff: 4,
		},
	}
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContent))
	got := FilterCheck(results, filediffs, 0, "", difffilter.ModeDiffContext)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}
