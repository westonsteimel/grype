package version

import (
	"fmt"
	"strings"

	hashiVer "github.com/anchore/go-version"
)

// ruby packages such as activerecord and sprockets don't strictly follow semver
// note: this may result in missed matches for versioned betas
var normalizer = strings.NewReplacer(".alpha", "-alpha", ".beta", "-beta", ".rc", "-rc")

type semanticConstraint struct {
	raw        string
	constraint hashiVer.Constraints
}

func newSemanticConstraint(constStr string) (semanticConstraint, error) {
	if constStr == "" {
		// an empty constraint is always satisfied
		return semanticConstraint{}, nil
	}

	normalized := normalizer.Replace(constStr)

	constraints, err := hashiVer.NewConstraint(normalized)
	if err != nil {
		return semanticConstraint{}, err
	}
	return semanticConstraint{
		raw:        normalized,
		constraint: constraints,
	}, nil
}

func (c semanticConstraint) supported(format Format) bool {
	return format == SemanticFormat
}

func (c semanticConstraint) Satisfied(version *Version) (bool, error) {
	if c.raw == "" && version != nil {
		// an empty constraint is always satisfied
		return true, nil
	} else if version == nil {
		if c.raw != "" {
			// a non-empty constraint with no version given should always fail
			return false, nil
		}
		return true, nil
	}

	if !c.supported(version.Format) {
		return false, fmt.Errorf("(semantic) unsupported format: %s", version.Format)
	}

	if version.rich.semVer == nil {
		return false, fmt.Errorf("no rich semantic version given: %+v", version)
	}
	return c.constraint.Check(version.rich.semVer.verObj), nil
}

func (c semanticConstraint) String() string {
	if c.raw == "" {
		return "none (semver)"
	}
	return fmt.Sprintf("%s (semver)", c.raw)
}
