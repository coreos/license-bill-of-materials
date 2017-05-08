package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type testResult struct {
	Package  string
	Licenses []*testResultRawLicense
	Err      string
}

type testResultRawLicense struct {
	License string
	Score   int
	Extra   int
	Missing int
}

func listTestLicenses(pkgs []string) ([]testResult, error) {
	gopath, err := filepath.Abs("testdata")
	if err != nil {
		return nil, err
	}
	gpackages, err := listPackagesWithLicenses(gopath, pkgs)
	if err != nil {
		return nil, err
	}
	res := []testResult{}
	for _, gp := range gpackages {
		tr := testResult{
			Package: gp.PackageName,
		}
		trls := []*testResultRawLicense{}
		for _, rl := range gp.RawLicenses {
			trl := testResultRawLicense{}
			if rl.Template != nil {
				trl.License = rl.Template.Title
				trl.Score = int(100 * rl.Score)
			}
			trl.Extra = len(rl.ExtraWords)
			trl.Missing = len(rl.MissingWords)
			trls = append(trls, &trl)
		}
		tr.Licenses = trls
		if gp.Err != "" {
			tr.Err = "some error"
		}
		res = append(res, tr)
	}
	return res, nil
}

func compareTestLicenses(pkgs []string, wanted []testResult) error {
	stringify := func(res []testResult) string {
		parts := []string{}
		for _, r := range res {
			s := fmt.Sprintf("%s:", r.Package)
			for i, rl := range r.Licenses {
				if i > 0 {
					s += ";"
				}
				s += fmt.Sprintf(" \"%s\" %d%%", rl.License, rl.Score)
				if r.Err != "" {
					s += " " + r.Err
				}
				if rl.Extra > 0 {
					s += fmt.Sprintf(" +%d", rl.Extra)
				}
				if rl.Missing > 0 {
					s += fmt.Sprintf(" -%d", rl.Missing)
				}
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, "\n")
	}

	licenses, err := listTestLicenses(pkgs)
	if err != nil {
		return err
	}
	got := stringify(licenses)
	expected := stringify(wanted)
	if got != expected {
		return fmt.Errorf("licenses do not match:\n%s\n!=\n%s", got, expected)
	}
	return nil
}

func TestNoDependencies(t *testing.T) {
	err := compareTestLicenses([]string{"colors/red"}, []testResult{
		{Package: "colors/red", Licenses: []*testResultRawLicense{
			{License: "MIT License", Score: 98, Missing: 2},
		},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Multiple licenses should be detected
func TestMultipleLicenses(t *testing.T) {
	err := compareTestLicenses([]string{"colors/blue"}, []testResult{
		{Package: "colors/blue", Licenses: []*testResultRawLicense{
			{License: "MIT License", Score: 98, Missing: 2},
			{License: "Apache License 2.0", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoLicense(t *testing.T) {
	err := compareTestLicenses([]string{"colors/green"}, []testResult{
		{Package: "colors/green", Licenses: []*testResultRawLicense{
			{License: "", Score: 0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMainWithDependencies(t *testing.T) {
	// It also tests license retrieval in parent directory.
	err := compareTestLicenses([]string{"colors/cmd/paint"}, []testResult{
		{Package: "colors/cmd/paint", Licenses: []*testResultRawLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/red", Licenses: []*testResultRawLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMainWithAliasedDependencies(t *testing.T) {
	err := compareTestLicenses([]string{"colors/cmd/mix"}, []testResult{
		{Package: "colors/cmd/mix", Licenses: []*testResultRawLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/red", Licenses: []*testResultRawLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
		{Package: "couleurs/red", Licenses: []*testResultRawLicense{
			{License: "GNU Lesser General Public License v2.1", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMissingPackage(t *testing.T) {
	_, err := listTestLicenses([]string{"colors/missing"})
	if err == nil {
		t.Fatal("no error on missing package")
	}
	if _, ok := err.(*MissingError); !ok {
		t.Fatalf("MissingError expected")
	}
}

func TestMismatch(t *testing.T) {
	err := compareTestLicenses([]string{"colors/yellow"}, []testResult{
		{Package: "colors/yellow", Licenses: []*testResultRawLicense{
			{License: "Microsoft Reciprocal License", Score: 25, Extra: 106,
				Missing: 131}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoBuildableGoSourceFiles(t *testing.T) {
	_, err := listTestLicenses([]string{"colors/cmd"})
	if err == nil {
		t.Fatal("no error on missing package")
	}
	if _, ok := err.(*MissingError); !ok {
		t.Fatalf("MissingError expected")
	}
}

func TestBroken(t *testing.T) {
	err := compareTestLicenses([]string{"colors/broken"}, []testResult{
		{Package: "colors/broken", Licenses: []*testResultRawLicense{
			{License: "GNU General Public License v3.0", Score: 100}},
		},
		{Package: "colors/missing", Err: "some error", Licenses: []*testResultRawLicense{
			{License: "", Score: 0}},
		},
		{Package: "colors/red", Licenses: []*testResultRawLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBrokenDependency(t *testing.T) {

	err := compareTestLicenses([]string{"colors/purple"}, []testResult{
		{Package: "colors/broken", Licenses: []*testResultRawLicense{
			{License: "GNU General Public License v3.0", Score: 100}},
		},
		{Package: "colors/missing", Err: "some error", Licenses: []*testResultRawLicense{
			{License: "", Score: 0}},
		},
		{Package: "colors/purple", Licenses: []*testResultRawLicense{
			{License: "", Score: 0}},
		},
		{Package: "colors/red", Licenses: []*testResultRawLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPackageExpression(t *testing.T) {
	err := compareTestLicenses([]string{"colors/cmd/..."}, []testResult{
		{Package: "colors/cmd/mix", Licenses: []*testResultRawLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/cmd/paint", Licenses: []*testResultRawLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/red", Licenses: []*testResultRawLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
		{Package: "couleurs/red", Licenses: []*testResultRawLicense{
			{License: "GNU Lesser General Public License v2.1", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCleanLicenseData(t *testing.T) {
	data := `The MIT License (MIT)

	Copyright (c) 2013 Ben Johnson

	Some other lines.
	And more.
	`
	cleaned := string(cleanLicenseData([]byte(data)))
	wanted := "the mit license (mit)\n\n\tsome other lines.\n\tand more.\n\t"
	if wanted != cleaned {
		t.Fatalf("license data mismatch:\n%q\n!=\n%q", cleaned, wanted)
	}
}

func TestStandardPackages(t *testing.T) {
	err := compareTestLicenses([]string{"encoding/json", "cmd/addr2line"}, []testResult{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOverrides(t *testing.T) {
	wl := []projectAndLicenses{
		{Project: "colors/broken", Licenses: []License{
			{Type: "GNU General Public License v3.0", Confidence: 1}},
		},
		{Project: "colors/missing", Licenses: []License{
			{Type: "override missing", Confidence: 1}},
		},
		{Project: "colors/red", Licenses: []License{
			{Type: "override existing", Confidence: 1}},
		},
	}
	override := `[
		{"project": "colors/missing", "licenses": [{"type": "override missing"}]},
		{"project": "colors/red", "licenses": [{"type": "override existing"}]}
	]`

	wd, derr := os.Getwd()
	if derr != nil {
		t.Fatal(derr)
	}
	oldenv := os.Getenv("GOPATH")
	defer os.Setenv("GOPATH", oldenv)
	os.Setenv("GOPATH", filepath.Join(wd, "testdata"))

	c, e := pkgsToLicenses([]string{"colors/broken"}, override)
	if len(e) != 0 {
		t.Fatalf("got %+v errors, expected nothing", e)
	}
	for i := range c {
		if !reflect.DeepEqual(wl[i], c[i]) {
			t.Errorf("#%d:\ngot      %+v,\nexpected %+v", i, c[i], wl[i])
		}
	}
}

func TestLongestPrefix(t *testing.T) {
	tests := []struct {
		gpackages []GoPackage
		wpfx      string
	}{
		{
			[]GoPackage{
				{PackageName: "a/b/c"},
				{PackageName: "a/b/c/d"},
			},
			"a/b/c",
		},
		{
			[]GoPackage{
				{PackageName: "a/b/c"},
				{PackageName: "a/b/c/d"},
				{PackageName: "a/b/c/d/e"},
			},
			"a/b/c",
		},
		{
			[]GoPackage{
				{PackageName: "a/b"},
				{PackageName: "a/b/c/d/f"},
				{PackageName: "a/b/c/d/e"},
			},
			"a/b",
		},
	}

	for i, tt := range tests {
		if s := longestCommonPrefix(tt.gpackages); s != tt.wpfx {
			t.Errorf("#%d: got %q, expected %q", i, s, tt.wpfx)
		}
	}
}
