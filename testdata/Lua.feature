Feature: Lua steps manage vars
  Lua scripts run with a "vars" module bound to the same variable store used by
  github.com/godogx/vars gherkin steps: a script can read variables set in
  gherkin, and gherkin steps can assert on variables a script produced.
  Variable names always include the vars prefix (default "$").

  # Lua's assert(condition, message) fails the script (and so the step and
  # the scenario) with that message when condition is false, or any other
  # falsy value (false or nil). The same is true for any other Lua runtime
  # error, e.g. a typo like calling a nil function.
  # For a worked example of a failure and how it surfaces, see
  # TestSteps_run_failedAssertionFailsStep in features_test.go.

  Scenario: Set a variable and read it back
    When I run lua script
      """
      -- vars.set(name, value) creates or overwrites a variable.
      vars.set("$foo", "bar")
      """
    Then variable $foo equals to "bar"

  Scenario: Read a variable set by an earlier gherkin step
    Given variable $foo is set to "bar"
    When I run lua script
      """
      -- vars.get(name) returns (value, found), like a Go map lookup.
      -- assert(condition, message) is Lua's standard way to bail out: if
      -- $foo were missing, this would raise an error and fail the step with
      -- the given message, instead of continuing with a nil value.
      local foo, found = vars.get("$foo")
      assert(found, "$foo should be set by the previous step")

      vars.set("$foo_upper", string.upper(foo))
      """
    Then variable $foo_upper equals to "BAR"

  Scenario: Arithmetic and string formatting
    When I run lua script
      """
      local price = 9.99
      local qty = 3

      -- string.format works like Go's fmt.Sprintf, handy for rounding money math.
      vars.set("$total", string.format("%.2f", price * qty))
      """
    Then variable $total equals to "29.97"

  Scenario: Format a fixed point in time into an ISO-8601 timestamp
    When I run lua script
      """
      -- os.date/os.time are the standard Lua time functions. The "!" prefix
      -- in the format string selects UTC, which is what most APIs expect.
      vars.set("$created_at", os.date("!%Y-%m-%dT%H:%M:%SZ", 1700000000))
      """
    Then variable $created_at equals to "2023-11-14T22:13:20Z"

  Scenario: Derive a future timestamp from a variable
    Given variable $created_at_unix is set to 1700000000
    When I run lua script
      """
      local created = vars.get("$created_at_unix")
      local oneHour = 3600

      -- Variables round-trip through Lua as plain numbers, so normal
      -- arithmetic works on them directly.
      vars.set("$expires_at_unix", created + oneHour)
      vars.set("$expires_at", os.date("!%Y-%m-%dT%H:%M:%SZ", created + oneHour))
      """
    Then variable $expires_at_unix equals to 1700003600
    And variable $expires_at equals to "2023-11-14T23:13:20Z"

  Scenario: Build a JSON object and assert individual fields with JSON paths
    When I run lua script
      """
      -- A Lua table with string keys becomes a JSON object. A table that
      -- looks like a 1..n sequence (like "tags" below) becomes a JSON array.
      vars.set("$order", {
          id = 42,
          total = 29.97,
          tags = {"sale", "priority"},
      })
      """
    Then variable $order matches JSON paths
      | $.id      | 42           |
      | $.total   | 29.97        |
      | $.tags[0] | "sale"       |
      | $.tags[1] | "priority"   |

  Scenario: Read a JSON variable set in gherkin, transform it, write it back
    Given variable $order is set to
      """json5
      {"id": 42, "price": 9.99, "qty": 3}
      """
    When I run lua script
      """
      -- vars.get returns nested JSON objects/arrays as Lua tables too, so
      -- they can be read and mutated with normal Lua table syntax.
      local order = vars.get("$order")
      order.total = order.price * order.qty

      vars.set("$order", order)
      """
    Then variable $order matches JSON paths
      | $.id    | 42    |
      | $.total | 29.97 |

  Scenario: Guard a computed value with assert before trusting it
    Given variable $order is set to
      """json5
      {"price": 9.99, "qty": 3}
      """
    When I run lua script
      """
      local order = vars.get("$order")
      local total = order.price * order.qty

      -- A passing assert has no visible effect, it only matters when the
      -- condition turns out false, e.g. try changing ">" to "<" here and
      -- rerun: the step then fails with the message below instead of
      -- silently storing a bogus $total.
      assert(total > 0, "computed total must be positive, got "..tostring(total))

      vars.set("$total", total)
      """
    Then variable $total equals to 29.97

  Scenario: Export computed context as a godog attachment
    When I run lua script
      """
      local order = {id = 42, total = 29.97}

      -- godog.attach(name, body, mediaType) records a piece of test context
      -- next to this step's result, the same mechanism godogx/httpsteps uses
      -- to expose HTTP request/response details for debugging. A table body
      -- is JSON-encoded automatically (media type defaults to
      -- "application/json"); a string body defaults to "text/plain".
      godog.attach("order", order)
      godog.attach("note", "computed inside the lua script")
      """

  Scenario: A script from file builds a more elaborate result
    When I run lua script from file
      """
      testdata/build_greeting.lua
      """
    Then variable $greeting matches JSON paths
      | $.text   | "hello, world" |
      | $.upper  | "HELLO, WORLD" |
      | $.length | 12             |

  Scenario: Deleting a variable removes it
    When I run lua script
      """
      vars.set("$foo", "bar")
      assert(vars.has("$foo"))

      vars.delete("$foo")
      assert(not vars.has("$foo"), "$foo should be gone after delete")
      """
    Then variable $foo is undefined
