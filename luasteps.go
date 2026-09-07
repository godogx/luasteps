package luasteps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bool64/shared"
	"github.com/cucumber/godog"
	"github.com/godogx/vars"
	lua "github.com/yuin/gopher-lua"
)

// Steps provides godog step definitions to run Lua scripts with access to shared vars.
type Steps struct {
	// VS is shared vars steps, set it up to share variables with github.com/godogx/vars steps.
	// A nil VS is safe to use and creates a fresh, unshared vars store per scenario.
	VS *vars.Steps

	// NewState creates the *lua.LState used to run each script. Set it up for
	// lower-level control of Lua, for example:
	//   - lua.Options (SkipOpenLibs to sandbox scripts, CallStackSize/RegistrySize
	//     to tune limits) via lua.NewState(lua.Options{...}),
	//   - extra global functions or modules, e.g. bound with gopher-luar
	//     (github.com/layeh/gopher-luar) or a hand-written LGFunction.
	//
	// If nil, lua.NewState() is used, which opens the full Lua standard
	// library. The "vars" module is registered after NewState returns, so it
	// always overrides a global of the same name set up here.
	NewState func() *lua.LState
}

// Register adds steps to scenario context.
func (s *Steps) Register(sc *godog.ScenarioContext) {
	// When I run lua script
	//   """
	//   vars.set("$foo", "bar")
	//   """
	sc.Step(`^I run lua script$`, s.runScript)

	// When I run lua script from file
	//   """
	//   testdata/script.lua
	//   """
	sc.Step(`^I run lua script from file$`, s.runScriptFromFile)
}

func (s *Steps) runScriptFromFile(ctx context.Context, filePath string) (context.Context, error) {
	src, err := os.ReadFile(filePath) //nolint:gosec // File inclusion via variable is intentional for tests.
	if err != nil {
		return ctx, fmt.Errorf("reading lua script file %s: %w", filePath, err)
	}

	return s.run(ctx, string(src))
}

func (s *Steps) runScript(ctx context.Context, src string) (context.Context, error) {
	return s.run(ctx, src)
}

func (s *Steps) run(ctx context.Context, src string) (context.Context, error) {
	ctx, v := s.VS.Vars(ctx)

	var L *lua.LState
	if s.NewState != nil {
		L = s.NewState()
	} else {
		L = lua.NewState()
	}

	defer L.Close()

	L.SetContext(ctx)
	registerVarsModule(L, v)

	var attachments []godog.Attachment

	registerGodogModule(L, &attachments)

	err := L.DoString(src)

	// Attach whatever was recorded even if the script failed partway through,
	// so the attachments still show up next to the failure in the report.
	if len(attachments) > 0 {
		ctx = godog.Attach(ctx, attachments...)
	}

	if err != nil {
		return ctx, fmt.Errorf("running lua script: %w", err)
	}

	return ctx, nil
}

// registerVarsModule exposes get/set/has/delete/all functions of v as a global "vars" table in l.
func registerVarsModule(l *lua.LState, v *shared.Vars) {
	mod := l.NewTable()

	l.SetFuncs(mod, map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			val, found := v.Get(L.CheckString(1))
			L.Push(toLua(L, val))
			L.Push(lua.LBool(found))

			return 2
		},
		"set": func(L *lua.LState) int {
			v.Set(L.CheckString(1), fromLua(L.CheckAny(2)))

			return 0
		},
		"has": func(L *lua.LState) int {
			_, found := v.Get(L.CheckString(1))
			L.Push(lua.LBool(found))

			return 1
		},
		"delete": func(L *lua.LState) int {
			v.Delete(L.CheckString(1))

			return 0
		},
		"all": func(L *lua.LState) int {
			t := L.NewTable()
			for k, val := range v.GetAll() {
				t.RawSetString(k, toLua(L, val))
			}

			L.Push(t)

			return 1
		},
	})

	l.SetGlobal("vars", mod)
}

// registerGodogModule exposes a global "godog" table with an attach(name, body, mediaType) function.
// Recorded attachments are appended to attachments, for the caller to hand to godog.Attach.
func registerGodogModule(l *lua.LState, attachments *[]godog.Attachment) {
	mod := l.NewTable()

	l.SetFuncs(mod, map[string]lua.LGFunction{
		"attach": func(L *lua.LState) int {
			name := L.CheckString(1)
			val := L.CheckAny(2)
			mediaType := L.OptString(3, "")

			var body []byte

			if s, ok := val.(lua.LString); ok {
				body = []byte(s)

				if mediaType == "" {
					mediaType = "text/plain"
				}
			} else {
				b, err := json.Marshal(fromLua(val))
				if err != nil {
					L.RaiseError("attach %q: %v", name, err)
				}

				body = b

				if mediaType == "" {
					mediaType = "application/json"
				}
			}

			*attachments = append(*attachments, godog.Attachment{
				FileName:  name,
				Body:      body,
				MediaType: mediaType,
			})

			return 0
		},
	})

	l.SetGlobal("godog", mod)
}
