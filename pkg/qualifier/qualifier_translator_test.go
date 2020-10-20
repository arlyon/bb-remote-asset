package qualifier_test

import (
	"fmt"
	"testing"

	remoteasset "github.com/bazelbuild/remote-apis/build/bazel/remote/asset/v1"
	"github.com/buildbarn/bb-remote-asset/pkg/qualifier"
)

func TestGitCommand(t *testing.T) {
	trans := qualifier.SimpleQualifierTranslator{}

	command, err := trans.QualifierToCommand([]remoteasset.Qualifier{
		remoteasset.Qualifier{Name: "resource_type", Value: "application/x-git"},
		remoteasset.Qualifier{Name: "vcs.branch", Value: "testing"},
	})

	fmt.Print(command("git@github.com:arlyon/graphics.git"), err)
}
