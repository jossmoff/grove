// Package acceptance_test drives the grove CLI end-to-end via godog/Cucumber.
//
// Each scenario gets an isolated world: fresh temp directories, a dedicated
// config file, and a dedicated cache dir. The grove binary under test is
// resolved from GROVE_BINARY (set by the Dockerfile) or built from source
// once in TestMain.
//
// Run locally:
//
//	go test -v -count=1 ./acceptance/...
//
// Run in Docker (isolated, CI-identical):
//
//	make acceptance-docker
package acceptance_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

// groveBin is set once in TestMain and shared across all scenarios.
var groveBin string

func TestMain(m *testing.M) {
	bin, err := resolveGroveBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "grove binary: %v\n", err)
		os.Exit(1)
	}
	groveBin = bin
	os.Exit(m.Run())
}

// resolveGroveBinary returns the path to the grove binary to test.
// GROVE_BINARY overrides (used in Docker); otherwise builds from source.
func resolveGroveBinary() (string, error) {
	if b := os.Getenv("GROVE_BINARY"); b != "" {
		return b, nil
	}
	bin := filepath.Join(os.TempDir(), fmt.Sprintf("grove-acceptance-%d", os.Getpid()))
	cmd := exec.Command("go", "build", "-o", bin, "github.com/jossmoff/grove/cmd/grove")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}
	return bin, nil
}

func TestAcceptance(t *testing.T) {
	suite := godog.TestSuite{
		Name: "grove",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			w := &World{}
			sc.Before(func(_ context.Context, _ *godog.Scenario) (context.Context, error) {
				return context.Background(), w.setup()
			})
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				w.teardown()
				return ctx, nil
			})
			registerSteps(sc, w)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run acceptance tests")
	}
}
