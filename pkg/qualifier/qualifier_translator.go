package qualifier

import (
	"fmt"
	"strings"

	remoteasset "github.com/bazelbuild/remote-apis/build/bazel/remote/asset/v1"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type QualifierTranslator interface {
	// Given an array of qualifiers, creates a function
	// that can produce commands to satisfy that qualifier combination.
	// A failure to provide the resource type
	QualifiersToCommand(qualifiers []remoteasset.Qualifier) (func(string) string, error)
}

type SimpleQualifierTranslator struct {
}

func makeMap(qualifiers []remoteasset.Qualifier) map[string]string {
	qual := make(map[string]string)
	for _, q := range qualifiers {
		qual[q.GetName()] = q.GetValue()
	}

	return qual
}

func (qt *SimpleQualifierTranslator) QualifierToCommand(qArr []remoteasset.Qualifier) (func(string) remoteexecution.Command, error) {
	qualifiers := makeMap(qArr)
	resource_type, ok := qualifiers["resource_type"]
	if !ok {
		return nil, fmt.Errorf("missing resource_type")
	}

	switch resource_type {
	case "application/x-git":
		return gitCommand(qualifiers), nil
	case "application/octet-stream":
		return octetStreamCommand(qualifiers), nil
	}

	return nil, fmt.Errorf("unhandled resource_type")
}

// Fetches an asset from a given git repo. Supported qualifiers:
// - vcs.branch: The branch to use
// - vsc.commit: The specific commit
//
// Note that supplying both is valid, however only if the
// requested commit exists on the branch.
func gitCommand(qualifiers map[string]string) func(string) remoteexecution.Command {
	return func(url string) remoteexecution.Command {
		script := fmt.Sprintf("git clone %s repo", url)
		if branch, ok := qualifiers["vcs.branch"]; ok {
			script = script + fmt.Sprintf(" --single-branch --branch %s", branch)
		}
		if commit, ok := qualifiers["vcs.commit"]; ok {
			script = script + fmt.Sprintf(" && cd repo && git checkout %s && cd ..", commit)
		}
		return remoteexecution.Command{
			Arguments:   []string{"sh", "-c", script},
			OutputPaths: []string{"repo"},
		}
	}
}

// Fetches an asset from a given url. Supported qualifiers:
// - auth.basic.username: authentication with a basic username
// - auth.basic.password: authentication with a basic password
// - checksum.sri: verify the checksum after downloading
func octetStreamCommand(qualifiers map[string]string) func(string) remoteexecution.Command {
	return func(url string) remoteexecution.Command {
		script := fmt.Sprintf("wget -O out %s", url)
		if username, ok := qualifiers["auth.basic.username"]; ok {
			script = script + fmt.Sprintf("--http-user=%s", username)
		}
		if password, ok := qualifiers["auth.basic.password"]; ok {
			script = script + fmt.Sprintf("--http-password=%s", password)

		}
		if checksum, ok := qualifiers["checksum.sri"]; ok {
			protocol, base64 := parseChecksum(checksum)
			script = script + fmt.Sprintf(" && openssl dgst -%s -binary out | openssl base64 -A | grep %s", protocol, base64)
		}

		return remoteexecution.Command{
			Arguments:   []string{"sh", "-c", script},
			OutputPaths: []string{"out"},
		}
	}
}

func parseChecksum(c string) (string, string) {
	parts := strings.Split(c, "-")
	return parts[0], parts[1]
}