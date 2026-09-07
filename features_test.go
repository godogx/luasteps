package luasteps_test

import (
	"bytes"
	"testing"

	"github.com/cucumber/godog"
	"github.com/godogx/luasteps"
	"github.com/godogx/vars"
	"github.com/stretchr/testify/assert"
)

func TestSteps_Register_features(t *testing.T) {
	vs := &vars.Steps{}
	ls := &luasteps.Steps{VS: vs}

	buf := bytes.NewBuffer(nil)

	suite := godog.TestSuite{
		Name: "LuaContext",
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			vs.Register(s)
			ls.Register(s)
		},
		Options: &godog.Options{
			Format: "pretty",
			Output: buf,
			Paths:  []string{"testdata/Lua.feature"},
			Strict: true,
		},
	}

	if status := suite.Run(); status != 0 {
		t.Fatal(buf.String())
	}
}

// A failed Lua assertion (or any other Lua runtime error) must fail the
// godog step, and the error message must reach the test output, so a user
// gets a useful failure instead of a silently passing scenario.
func TestSteps_run_failedAssertionFailsStep(t *testing.T) {
	ls := &luasteps.Steps{}

	feature := godog.Feature{
		Name: "failing.feature",
		Contents: []byte(`Feature: failing assertion
  Scenario: assert stops the scenario
    When I run lua script
      """
      vars.set("$total", 10)
      assert(vars.get("$total") == 20, "expected $total to be 20")
      """
`),
	}

	buf := bytes.NewBuffer(nil)

	suite := godog.TestSuite{
		Name: "LuaContext",
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			ls.Register(s)
		},
		Options: &godog.Options{
			Format:          "pretty",
			Output:          buf,
			FeatureContents: []godog.Feature{feature},
			Strict:          true,
		},
	}

	status := suite.Run()

	assert.NotEqual(t, 0, status, "a failed lua assertion should fail the scenario")
	assert.Contains(t, buf.String(), "expected $total to be 20")
}
