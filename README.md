# Lua steps for godog

[![Build Status](https://github.com/godogx/luasteps/workflows/test-unit/badge.svg)](https://github.com/godogx/luasteps/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/godogx/luasteps/branch/master/graph/badge.svg)](https://codecov.io/gh/godogx/luasteps)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/godogx/luasteps)

This module implements [`godog`](https://github.com/cucumber/godog) step definitions that run
arbitrary [Lua](https://www.lua.org/) scripts (inline or from file), with access to variables
shared through [`github.com/godogx/vars`](https://github.com/godogx/vars).

It is useful for scenarios that need a bit of scripted logic (computing a value, transforming
a variable, building test data) that would be awkward to express with plain gherkin steps.

## Setup

```go
vs := &vars.Steps{}
ls := &luasteps.Steps{VS: vs} // share vars store with vars.Steps

ScenarioInitializer: func(s *godog.ScenarioContext) {
    vs.Register(s)
    ls.Register(s)
}
```

## Lua API

Scripts run with the full Lua standard library available, plus a `vars` module bound to the
shared variable store. Variable names include the vars prefix (default `$`), same as in
`github.com/godogx/vars` steps and tables.

```lua
vars.set("$foo", "bar")            -- create or update a variable
local v, found = vars.get("$foo")  -- read a variable
local exists = vars.has("$foo")
vars.delete("$foo")                -- remove a variable
local all = vars.all()             -- table of all variables
```

Values are converted between Go and Lua: strings, booleans, numbers, `nil`, and JSON-like
tables (arrays and objects) round-trip transparently.

Lua's built-in `assert(condition, message)` is the natural way to make a script fail: any Lua
runtime error, including a failed assert, fails the godog step with that error message.

```lua
assert(vars.has("$order"), "$order must be set before this step")
```

## Attachments

`godog.attach(name, body, mediaType)` attaches a piece of test context to the step's result,
the same mechanism [`godogx/httpsteps`](https://github.com/godogx/httpsteps) uses to expose
HTTP request/response details. It shows up in report formats that render attachments (e.g.
cucumber JSON, pretty).

```lua
godog.attach("note", "plain text body")               -- MediaType defaults to "text/plain"
godog.attach("note", "plain text body", "text/html")  -- explicit MediaType
godog.attach("order", {id = 42, total = 29.97})       -- a table is JSON-encoded,
                                                       -- MediaType defaults to "application/json"
```

Attachments made before a script fails (e.g. a failed `assert`) are still recorded, so they
remain useful for debugging the failure.

## Advanced Lua Configuration

`Steps.NewState` creates the `*lua.LState` used for each script run, the extension point for
lower-level control: sandboxing, tuning limits, or registering extra Go functions/modules (by
hand, or with a binding library like [`gopher-luar`](https://github.com/layeh/gopher-luar)).

```go
ls := &luasteps.Steps{
    VS: vs,
    NewState: func() *lua.LState {
        // Sandbox scripts: no os/io libraries available to Lua.
        L := lua.NewState(lua.Options{SkipOpenLibs: true})
        lua.OpenBase(L)
        lua.OpenString(L)
        lua.OpenTable(L)
        lua.OpenMath(L)

        return L
    },
}
```

If `NewState` is nil, `lua.NewState()` is used, which opens the full standard library. The
`vars` module is always registered after `NewState` returns.

## Step Definitions

Run an inline Lua script.

```gherkin
When I run lua script
  """
  vars.set("$foo", "bar")
  """
```

Run a Lua script from a file.

```gherkin
When I run lua script from file
  """
  testdata/script.lua
  """
```
