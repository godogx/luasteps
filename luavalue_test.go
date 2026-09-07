package luasteps //nolint:testpackage // whitebox tests exercise unexported toLua/fromLua/fromLuaTable

import (
	"reflect"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestToLuaFromLua_roundTrip(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	cases := []struct {
		name string
		in   any
		want any
	}{
		{"nil", nil, nil},
		{"bool", true, true},
		{"string", "abc", "abc"},
		{"int64", int64(42), int64(42)},
		{"float", 4.5, 4.5},
		{"whole float becomes int64", float64(3), int64(3)},
		{"array", []any{int64(1), "two", true}, []any{int64(1), "two", true}},
		{"map", map[string]any{"a": int64(1), "b": "two"}, map[string]any{"a": int64(1), "b": "two"}},
		{"uint64 beyond int64 range stays float64", uint64(1) << 63, float64(uint64(1) << 63)},
		{"huge float stays float64 instead of overflowing int64", 1e20, 1e20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fromLua(toLua(L, tc.in))

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestFromLuaTable_nonSequenceNumericKeysFallBackToMap(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// A non-integer numeric key must not be silently dropped: the table
	// isn't a clean 1..n sequence, so it must be treated as a map.
	tbl := L.NewTable()
	tbl.RawSetInt(1, lua.LString("x"))
	tbl.RawSet(lua.LNumber(1.5), lua.LString("y"))

	got, ok := fromLua(tbl).(map[string]any)
	if !ok {
		t.Fatalf("got %#v, want a map[string]any", fromLua(tbl))
	}

	want := map[string]any{"1": "x", "1.5": "y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
