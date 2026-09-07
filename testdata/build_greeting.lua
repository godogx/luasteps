-- Scripts loaded from file work exactly like inline scripts, they just come
-- from a separate .lua file, which is handy for logic too long for a docstring.
--
-- This script builds a small JSON-shaped result and stores it as a single
-- variable, so it can be asserted field by field with JSON paths.
local text = "hello, world"

vars.set("$greeting", {
    text = text,
    upper = string.upper(text),
    length = string.len(text),
})
