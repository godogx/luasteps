package luasteps //nolint:testpackage // whitebox tests exercise unexported run/runScriptFromFile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
	"github.com/godogx/vars"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestSteps_run(t *testing.T) {
	s := &Steps{VS: &vars.Steps{}}

	ctx, err := s.run(context.Background(), `
		vars.set("$foo", "bar")
		vars.set("$num", 42)
		assert(vars.has("$foo"))
		assert(not vars.has("$missing"))

		local v = vars.get("$foo")
		assert(v == "bar", "unexpected value: "..tostring(v))
	`)
	require.NoError(t, err)

	_, v := s.VS.Vars(ctx)

	foo, found := v.Get("$foo")
	assert.True(t, found)
	assert.Equal(t, "bar", foo)

	num, found := v.Get("$num")
	assert.True(t, found)
	assert.Equal(t, int64(42), num)
}

func TestSteps_run_delete(t *testing.T) {
	s := &Steps{VS: &vars.Steps{}}

	ctx, err := s.run(context.Background(), `vars.set("$foo", "bar")`)
	require.NoError(t, err)

	ctx, err = s.run(ctx, `vars.delete("$foo")`)
	require.NoError(t, err)

	_, v := s.VS.Vars(ctx)

	val, found := v.Get("$foo")
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestSteps_run_scriptError(t *testing.T) {
	s := &Steps{}

	_, err := s.run(context.Background(), `this is not valid lua`)
	assert.Error(t, err)
}

func TestSteps_runScriptFromFile(t *testing.T) {
	s := &Steps{}

	path := filepath.Join(t.TempDir(), "script.lua")
	require.NoError(t, os.WriteFile(path, []byte(`vars.set("$x", 1)`), 0o600))

	ctx, err := s.runScriptFromFile(context.Background(), path)
	require.NoError(t, err)

	_, v := vars.Vars(ctx)

	x, found := v.Get("$x")
	assert.True(t, found)
	assert.Equal(t, int64(1), x)
}

func TestSteps_runScriptFromFile_missing(t *testing.T) {
	s := &Steps{}

	_, err := s.runScriptFromFile(context.Background(), filepath.Join(t.TempDir(), "missing.lua"))
	assert.Error(t, err)
}

func TestSteps_run_customNewState_sandboxesStdlib(t *testing.T) {
	s := &Steps{
		NewState: func() *lua.LState {
			// Only base, string, table and math libs are available: no os or io.
			L := lua.NewState(lua.Options{SkipOpenLibs: true})
			lua.OpenBase(L)
			lua.OpenString(L)
			lua.OpenTable(L)
			lua.OpenMath(L)

			return L
		},
	}

	_, err := s.run(context.Background(), `return string.upper("ok")`)
	assert.NoError(t, err, "string lib should be available")

	_, err = s.run(context.Background(), `os.time()`)
	assert.Error(t, err, "os lib should not be available in a sandboxed state")
}

func TestSteps_run_customNewState_addsGlobalFunction(t *testing.T) {
	s := &Steps{
		VS: &vars.Steps{},
		NewState: func() *lua.LState {
			L := lua.NewState()
			L.SetGlobal("double", L.NewFunction(func(L *lua.LState) int {
				L.Push(lua.LNumber(L.CheckNumber(1) * 2))

				return 1
			}))

			return L
		},
	}

	ctx, err := s.run(context.Background(), `vars.set("$doubled", double(21))`)
	require.NoError(t, err)

	_, v := s.VS.Vars(ctx)
	doubled, found := v.Get("$doubled")
	assert.True(t, found)
	assert.Equal(t, int64(42), doubled)
}

func TestSteps_run_attach_string(t *testing.T) {
	s := &Steps{}

	ctx, err := s.run(context.Background(), `godog.attach("note", "hello from lua")`)
	require.NoError(t, err)

	att := godog.Attachments(ctx)
	require.Len(t, att, 1)
	assert.Equal(t, "note", att[0].FileName)
	assert.Equal(t, "text/plain", att[0].MediaType)
	assert.Equal(t, "hello from lua", string(att[0].Body))
}

func TestSteps_run_attach_table_asJSON(t *testing.T) {
	s := &Steps{}

	ctx, err := s.run(context.Background(), `godog.attach("order", {id = 42, total = 29.97})`)
	require.NoError(t, err)

	att := godog.Attachments(ctx)
	require.Len(t, att, 1)
	assert.Equal(t, "order", att[0].FileName)
	assert.Equal(t, "application/json", att[0].MediaType)
	assert.JSONEq(t, `{"id": 42, "total": 29.97}`, string(att[0].Body))
}

func TestSteps_run_attach_explicitMediaType(t *testing.T) {
	s := &Steps{}

	ctx, err := s.run(context.Background(), `godog.attach("page", "<html></html>", "text/html")`)
	require.NoError(t, err)

	att := godog.Attachments(ctx)
	require.Len(t, att, 1)
	assert.Equal(t, "text/html", att[0].MediaType)
}

func TestSteps_run_attach_survivesScriptFailure(t *testing.T) {
	s := &Steps{}

	ctx, err := s.run(context.Background(), `
		godog.attach("partial-progress", "made it this far")
		assert(false, "boom")
	`)
	assert.Error(t, err)

	att := godog.Attachments(ctx)
	require.Len(t, att, 1)
	assert.Equal(t, "partial-progress", att[0].FileName)
}
