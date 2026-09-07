package luasteps_test

import (
	"github.com/cucumber/godog"
	"github.com/godogx/luasteps"
	"github.com/godogx/vars"
)

func ExampleSteps_Register() {
	vs := &vars.Steps{}
	ls := &luasteps.Steps{VS: vs}

	suite := godog.TestSuite{
		Name: "LuaContext",
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			vs.Register(s)
			ls.Register(s)
		},
		Options: &godog.Options{
			Format: "pretty",
			Paths:  []string{"features"},
			Strict: true,
		},
	}
	status := suite.Run()

	println(status)
}
