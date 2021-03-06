package cyclonedx

import (
	"bytes"
	"flag"
	"regexp"
	"testing"

	"github.com/anchore/grype/grype/presenter/models"

	"github.com/anchore/go-testutils"
	"github.com/anchore/grype/grype/match"
	"github.com/anchore/grype/grype/pkg"
	"github.com/anchore/grype/grype/vulnerability"
	"github.com/anchore/stereoscope/pkg/imagetest"
	syftPkg "github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/source"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var update = flag.Bool("update", false, "update the *.golden files for json presenters")

func createResults() (match.Matches, []pkg.Package) {
	// the catalog is needed to assign the package IDs
	catalog := syftPkg.NewCatalog(
		syftPkg.Package{
			Name:    "package-1",
			Version: "1.0.1",
			Type:    syftPkg.DebPkg,
		},
		syftPkg.Package{
			Name:    "package-2",
			Version: "2.0.1",
			Type:    syftPkg.DebPkg,
			Licenses: []string{
				"MIT",
				"Apache-v2",
			},
		})

	packages := pkg.FromCatalog(catalog)

	var pkg1 = packages[0]
	var pkg2 = packages[1]

	var match1 = match.Match{
		Type: match.ExactDirectMatch,
		Vulnerability: vulnerability.Vulnerability{
			ID:        "CVE-1999-0001",
			Namespace: "source-1",
		},
		Package: pkg1,
		MatchDetails: []match.Details{
			{
				Matcher: match.DpkgMatcher,
			},
		},
	}

	var match2 = match.Match{
		Type: match.ExactIndirectMatch,
		Vulnerability: vulnerability.Vulnerability{
			ID:        "CVE-1999-0002",
			Namespace: "source-2",
		},
		Package: pkg2,
		MatchDetails: []match.Details{
			{
				Matcher: match.DpkgMatcher,
				SearchedBy: map[string]interface{}{
					"some": "key",
				},
			},
		},
	}

	matches := match.NewMatches()

	matches.Add(pkg1, match1, match2)

	return matches, packages
}

func TestCycloneDxPresenterImage(t *testing.T) {
	var buffer bytes.Buffer

	matches, packages := createResults()

	img := imagetest.GetFixtureImage(t, "docker-archive", "image-simple")
	s, err := source.NewFromImage(img, "user-input")

	// This accounts for the non-deterministic digest value that we end up with when
	// we build a container image dynamically during testing. Ultimately, we should
	// use a golden image as a test fixture in place of building this image during
	// testing. At that time, this line will no longer be necessary.
	//
	// This value is sourced from the "version" node in "./test-fixtures/snapshot/TestCycloneDxImgsPresenter.golden"
	s.Metadata.ImageMetadata.ManifestDigest = "sha256:2731251dc34951c0e50fcc643b4c5f74922dad1a5d98f302b504cf46cd5d9368"

	pres := NewPresenter(matches, packages, &s.Metadata, models.NewMetadataMock())
	// run presenter
	err = pres.Present(&buffer)
	if err != nil {
		t.Fatal(err)
	}

	actual := buffer.Bytes()
	if *update {
		testutils.UpdateGoldenFileContents(t, actual)
	}

	var expected = testutils.GetGoldenFileContents(t)

	// remove dynamic values, which are tested independently
	actual = redact(actual)
	expected = redact(expected)

	if !bytes.Equal(expected, actual) {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(expected), string(actual), true)
		t.Errorf("mismatched output:\n%s", dmp.DiffPrettyText(diffs))
	}

}

func TestCycloneDxPresenterDir(t *testing.T) {
	var buffer bytes.Buffer
	matches, packages := createResults()

	s, err := source.NewFromDirectory("/some/path")
	if err != nil {
		t.Fatal(err)
	}

	pres := NewPresenter(matches, packages, &s.Metadata, models.NewMetadataMock())

	// run presenter
	err = pres.Present(&buffer)
	if err != nil {
		t.Fatal(err)
	}

	actual := buffer.Bytes()
	if *update {
		testutils.UpdateGoldenFileContents(t, actual)
	}

	var expected = testutils.GetGoldenFileContents(t)

	// remove dynamic values, which are tested independently
	actual = redact(actual)
	expected = redact(expected)

	if !bytes.Equal(expected, actual) {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(expected), string(actual), true)
		t.Errorf("mismatched output:\n%s", dmp.DiffPrettyText(diffs))
	}

}

func redact(s []byte) []byte {
	serialPattern := regexp.MustCompile(`serialNumber="[a-zA-Z0-9\-:]+"`)
	refPattern := regexp.MustCompile(`ref="[a-zA-Z0-9\-:]+"`)
	rfc3339Pattern := regexp.MustCompile(`([0-9]+)-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]|60)(\.[0-9]+)?(([Zz])|([\+|\-]([01][0-9]|2[0-3]):[0-5][0-9]))`)

	for _, pattern := range []*regexp.Regexp{serialPattern, rfc3339Pattern, refPattern} {
		s = pattern.ReplaceAll(s, []byte("redacted"))
	}
	return s
}
