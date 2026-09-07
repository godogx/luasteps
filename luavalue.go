package luasteps

import (
	"fmt"
	"math"

	lua "github.com/yuin/gopher-lua"
)

// maxSafeInt64Float is 1<<63, the smallest float64 that would overflow int64.
const maxSafeInt64Float = 1 << 63

// toLua converts a Go value (as produced by encoding/json or shared.Vars) into a Lua value.
func toLua(l *lua.LState, v any) lua.LValue {
	if n, ok := toLuaNumber(v); ok {
		return n
	}

	switch vv := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(vv)
	case string:
		return lua.LString(vv)
	case []any:
		return toLuaArray(l, vv)
	case map[string]any:
		return toLuaMap(l, vv)
	default:
		return lua.LString(fmt.Sprint(vv))
	}
}

// toLuaNumber converts the numeric Go types shared.Vars can hold into a Lua number.
func toLuaNumber(v any) (lua.LNumber, bool) {
	switch vv := v.(type) {
	case int:
		return lua.LNumber(vv), true
	case int64:
		return lua.LNumber(vv), true
	case uint64:
		// shared.Vars.Set decodes large unsigned JSON numbers (that overflow
		// int64) into uint64, e.g. via DecodeJSONNumber.
		return lua.LNumber(vv), true
	case float64:
		return lua.LNumber(vv), true
	default:
		return 0, false
	}
}

func toLuaArray(l *lua.LState, vv []any) *lua.LTable {
	t := l.NewTable()
	for i, e := range vv {
		t.RawSetInt(i+1, toLua(l, e))
	}

	return t
}

func toLuaMap(l *lua.LState, vv map[string]any) *lua.LTable {
	t := l.NewTable()
	for k, e := range vv {
		t.RawSetString(k, toLua(l, e))
	}

	return t
}

// fromLua converts a Lua value into a plain Go value suitable for JSON encoding and shared.Vars storage.
func fromLua(v lua.LValue) any {
	switch vv := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(vv)
	case lua.LNumber:
		f := float64(vv)
		// 1<<63 is exactly representable in float64; converting anything at
		// or beyond it to int64 would overflow (undefined in Go), so such
		// values (and +/-Inf, which also fail this range check) stay float64.
		if f == math.Trunc(f) && f >= -maxSafeInt64Float && f < maxSafeInt64Float {
			return int64(f)
		}

		return f
	case lua.LString:
		return string(vv)
	case *lua.LTable:
		return fromLuaTable(vv)
	default:
		return v.String()
	}
}

// fromLuaTable converts a Lua table into a []any if it looks like a sequence
// (1..n integer keys with no gaps), or into a map[string]any otherwise.
func fromLuaTable(t *lua.LTable) any {
	n := t.Len()
	isArray := n > 0

	t.ForEach(func(k, _ lua.LValue) {
		kn, ok := k.(lua.LNumber)
		if !ok {
			isArray = false

			return
		}

		f := float64(kn)
		if f != math.Trunc(f) || f < 1 || f > float64(n) {
			isArray = false
		}
	})

	if isArray {
		arr := make([]any, 0, n)
		for i := 1; i <= n; i++ {
			arr = append(arr, fromLua(t.RawGetInt(i)))
		}

		return arr
	}

	m := make(map[string]any)

	t.ForEach(func(k, val lua.LValue) {
		m[k.String()] = fromLua(val)
	})

	return m
}
