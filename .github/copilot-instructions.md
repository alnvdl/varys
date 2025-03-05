Never introduce new external dependencies to a file, unless explicitly told to do so. Feel free to use everything from the standard library and any imports that are already present in the files you are working with.

When declaring slices in test code, make sure the curly braces are placed as compactly as possible. For example:
```go
var tests = []struct {
	desc string
}{{
	desc: "...",
}, {
	desc: "...",
}}
```

Use the `test` variable when iterating over subtests in a table test.

The description for each subtest should be named `desc` and it should always start with a number or lowercase letter.

Always end comments with a period.
