// Package luasteps provides godog step definitions to run Lua scripts that can
// read, create, update and clear variables shared with github.com/godogx/vars.
//
// # Vars Setup
//
// Share a single instance of vars.Steps so that Lua scripts and other godog
// steps see the same variables.
//
//	vs := &vars.Steps{}
//	ls := &luasteps.Steps{VS: vs}
//
//	ScenarioInitializer: func(s *godog.ScenarioContext) {
//		vs.Register(s)
//		ls.Register(s)
//	}
//
// # Lua API
//
// Scripts run with the Lua standard library available and a "vars" module
// bound to the shared variables. Variable names include the vars prefix
// (default "$"), same as in gherkin steps of github.com/godogx/vars.
//
//	vars.set("$foo", "bar")     -- create or update a variable
//	local v, found = vars.get("$foo")
//	local exists = vars.has("$foo")
//	vars.delete("$foo")        -- remove a variable
//	local all = vars.all()     -- table of all variables
//
// # Assertions And Failures
//
// Lua's built-in assert(condition, message) is the natural way to make a
// script fail: any Lua runtime error, including a failed assert, aborts the
// script and fails the godog step with that error message.
//
//	assert(vars.has("$order"), "$order must be set before this step")
//	assert(vars.get("$total") > 0, "total must be positive")
//
// # Attachments
//
// godog.attach(name, body, mediaType) attaches a piece of test context to the
// step's result, the same mechanism github.com/godogx/httpsteps uses to expose
// HTTP request/response details. It shows up in report formats that render
// attachments (e.g. cucumber JSON, pretty).
//
//	godog.attach("note", "plain text body")                  -- MediaType defaults to "text/plain"
//	godog.attach("note", "plain text body", "text/html")     -- explicit MediaType
//	godog.attach("order", {id = 42, total = 29.97})           -- a table is JSON-encoded,
//	                                                           -- MediaType defaults to "application/json"
//
// Attachments made before a script fails (e.g. a failed assert) are still
// recorded, so they remain useful for debugging the failure.
//
// # Advanced Lua Configuration
//
// Steps.NewState creates the *lua.LState used for each script run, so it is
// the extension point for lower-level control: sandboxing via lua.Options,
// tuning limits, or registering extra Go functions/modules (by hand, or with
// a binding library like github.com/layeh/gopher-luar).
//
//	ls := &luasteps.Steps{
//		VS: vs,
//		NewState: func() *lua.LState {
//			// Sandbox scripts: no os/io libraries available to Lua.
//			L := lua.NewState(lua.Options{SkipOpenLibs: true})
//			lua.OpenBase(L)
//			lua.OpenString(L)
//			lua.OpenTable(L)
//			lua.OpenMath(L)
//
//			return L
//		},
//	}
//
// If NewState is nil, lua.NewState() is used, which opens the full standard
// library. The "vars" module is always registered after NewState returns.
//
// # Step Definitions
//
// Run an inline Lua script.
//
//	When I run lua script
//	  """
//	  vars.set("$foo", "bar")
//	  """
//
// Run a Lua script from a file.
//
//	When I run lua script from file
//	  """
//	  testdata/script.lua
//	  """
package luasteps
