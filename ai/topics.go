// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

// Topic is a Go concept category shown in the mission popup.
type Topic struct {
	Slug     string
	Title    string
	Concepts string // short comma-separated display string
	Guidance string // topic-wide rules injected into every order for this topic
	Beats    []Beat // the syllabus, easiest first
}

// Beat is one idiom inside a topic. The harness picks the beat; the signal only
// writes the mission around it. Requires and Forbids are engine construct rules
// checked against the player's region, so a beat cannot be passed without it.
type Beat struct {
	Slug     string
	Brief    string
	Requires []string
	Forbids  []string
}

// FindTopic returns the topic with the given slug.
func FindTopic(slug string) (Topic, bool) {
	for _, t := range StdlibTopics {
		if t.Slug == slug {
			return t, true
		}
	}
	return Topic{}, false
}

// OpenTopic builds a one-off topic from the astronaut's own words. It carries a
// single beat with no construct rules: the wish itself is the syllabus.
func OpenTopic(wish string) Topic {
	return Topic{
		Slug:     "open",
		Title:    "Open channel",
		Concepts: wish,
		Guidance: "The astronaut chose this topic in their own words. Stay within standard-library Go.",
		Beats:    []Beat{{Slug: "wish", Brief: wish}},
	}
}

const deterministic = "Output must be deterministic: sort results or use fixed counts before printing."

// StdlibTopics is the catalog of Go topics available for forged missions.
var StdlibTopics = []Topic{
	{
		Slug: "variables", Title: "Variables & Types", Concepts: "var, const, basic types, zero values",
		Beats: []Beat{
			{Slug: "short-decl", Brief: "declare one value with := and print it", Requires: []string{"short-var"}},
			{Slug: "var-zero", Brief: "declare variables with var and no initializer, print their zero values", Requires: []string{"var"}},
			{Slug: "const-threshold", Brief: "use a const threshold in a comparison", Requires: []string{"const"}},
			{Slug: "conversion", Brief: "convert between int and float64 explicitly before computing", Requires: []string{"conversion"}},
			{Slug: "multi-assign", Brief: "swap or assign several values in one statement", Requires: []string{"multi-assign"}},
			{Slug: "rune-byte", Brief: "inspect a string as bytes and as runes and print both counts", Requires: []string{"call:utf8.RuneCountInString"}},
			{Slug: "iota", Brief: "define an enumeration with iota and print a value by name", Requires: []string{"iota"}},
		},
	},
	{
		Slug: "functions", Title: "Functions", Concepts: "signatures, multiple returns, variadic",
		Guidance: "Markers sit at package level; the player writes the function and the scaffold footer calls it, or the player writes func main as well.",
		Beats: []Beat{
			{Slug: "single-return", Brief: "write a function with one return value", Requires: []string{"func"}},
			{Slug: "value-error", Brief: "return a value and an error, and handle the error at the call site", Requires: []string{"func", "multi-return"}},
			{Slug: "named-returns", Brief: "use named result parameters with a bare return", Requires: []string{"named-returns"}},
			{Slug: "variadic", Brief: "write a variadic function (...T) and call it with a spread slice", Requires: []string{"variadic"}},
			{Slug: "func-value", Brief: "pass a function as a value to another function", Requires: []string{"func-param"}},
			{Slug: "closure", Brief: "return a closure that keeps state between calls", Requires: []string{"closure"}},
			{Slug: "recursion", Brief: "solve the task with a recursive function", Requires: []string{"recursion"}},
			{Slug: "defer-order", Brief: "use several defers and rely on their LIFO order", Requires: []string{"defer"}},
		},
	},
	{
		Slug: "control-flow", Title: "Control Flow", Concepts: "if, switch, for, range",
		Beats: []Beat{
			{Slug: "if-else", Brief: "branch with if / else if / else", Requires: []string{"if"}},
			{Slug: "if-init", Brief: "use an if statement with an init clause", Requires: []string{"if-init"}},
			{Slug: "switch-expr", Brief: "switch on a value with several cases and a default", Requires: []string{"switch"}},
			{Slug: "switch-true", Brief: "use a tagless switch as a cleaner if-else chain", Requires: []string{"switch-tagless"}},
			{Slug: "for-classic", Brief: "loop with a classic three-clause for", Requires: []string{"for-classic"}},
			{Slug: "for-while", Brief: "loop with a condition-only for (the Go while)", Requires: []string{"for-cond"}},
			{Slug: "range-int", Brief: "loop over an integer with range n", Requires: []string{"range"}},
			{Slug: "range-string", Brief: "range over a string and handle runes, not bytes", Requires: []string{"range"}},
			{Slug: "labeled-break", Brief: "break out of a nested loop with a label", Requires: []string{"label"}},
		},
	},
	{
		Slug: "slices", Title: "Slices", Concepts: "append, copy, slicing, capacity",
		Guidance: "Print intermediate len/cap states when they make memory behaviour visible.",
		Beats: []Beat{
			{Slug: "append", Brief: "grow a slice with append", Requires: []string{"call:append"}},
			{Slug: "copy", Brief: "copy into a separate slice and show the source is unchanged", Requires: []string{"call:copy"}},
			{Slug: "subslice", Brief: "take sub-slices with [low:high] and show they share the backing array", Requires: []string{"slice-expr"}},
			{Slug: "three-index", Brief: "limit capacity with a three-index slice so append does not clobber the source", Requires: []string{"slice-3"}},
			{Slug: "make-cap", Brief: "preallocate with make([]T, 0, n) and print len and cap", Requires: []string{"call:make", "call:cap"}},
			{Slug: "filter-inplace", Brief: "filter a slice in place reusing its backing array (s[:0])", Requires: []string{"call:append", "slice-expr"}},
			{Slug: "pkg-slices", Brief: "use the slices package (slices.Sort, slices.Index, slices.Contains)", Requires: []string{"slices.*"}},
			{Slug: "grid", Brief: "build and walk a 2D slice", Requires: []string{"nested-slice"}},
		},
	},
	{
		Slug: "maps", Title: "Maps", Concepts: "make, delete, range, nil maps",
		Beats: []Beat{
			{Slug: "literal", Brief: "initialise a map with a literal and read from it", Requires: []string{"map"}},
			{Slug: "comma-ok", Brief: "tell a missing key from a zero value with comma-ok", Requires: []string{"comma-ok"}},
			{Slug: "delete", Brief: "delete keys from a map", Requires: []string{"call:delete"}},
			{Slug: "frequency", Brief: "count frequencies with m[k]++", Requires: []string{"map", "inc"}},
			{Slug: "group", Brief: "group values into a map of slices", Requires: []string{"map", "call:append"}},
			{Slug: "sorted-keys", Brief: "print a map in sorted key order", Requires: []string{"range", "sort"}},
			{Slug: "set", Brief: "use map[T]struct{} as a set", Requires: []string{"empty-struct"}},
			{Slug: "pkg-maps", Brief: "use the maps package (maps.Keys with slices.Sorted, maps.Clone)", Requires: []string{"maps.*"}},
		},
	},
	{
		Slug: "structs", Title: "Structs & Methods", Concepts: "struct literals, value/pointer receivers",
		Guidance: "Markers sit at package level; the player declares the types and methods.",
		Beats: []Beat{
			{Slug: "literal", Brief: "declare a struct type and build it with a keyed literal", Requires: []string{"struct"}},
			{Slug: "value-method", Brief: "add a value-receiver method that reads fields", Requires: []string{"method"}},
			{Slug: "pointer-method", Brief: "add a pointer-receiver method that mutates state", Requires: []string{"pointer-method"}},
			{Slug: "embedding", Brief: "embed one struct in another and call a promoted method", Requires: []string{"embed"}},
			{Slug: "constructor", Brief: "write a NewX constructor that validates input and returns *X", Requires: []string{"func", "pointer"}},
			{Slug: "stringer", Brief: "implement String() string so fmt prints the value", Requires: []string{"method"}},
			{Slug: "compare", Brief: "compare two struct values with ==", Requires: []string{"struct", "eq"}},
		},
	},
	{
		Slug: "interfaces", Title: "Interfaces", Concepts: "implicit impl, empty interface, type assertion",
		Guidance: "Markers sit at package level.",
		Beats: []Beat{
			{Slug: "two-impls", Brief: "define an interface and two concrete types that satisfy it", Requires: []string{"interface", "method"}},
			{Slug: "comma-ok-assert", Brief: "recover a concrete type with a comma-ok type assertion", Requires: []string{"type-assert"}},
			{Slug: "type-switch", Brief: "dispatch over three or more types with a type switch", Requires: []string{"type-switch"}},
			{Slug: "composition", Brief: "compose a larger interface from smaller ones", Requires: []string{"interface-embed"}},
			{Slug: "io-writer", Brief: "accept an io.Writer and write to it", Requires: []string{"io.Writer"}},
			{Slug: "nil-trap", Brief: "show why an interface holding a typed nil pointer is not nil", Requires: []string{"interface", "nil"}},
		},
	},
	{
		Slug: "errors", Title: "Errors", Concepts: "error type, fmt.Errorf, errors.Is/As",
		Beats: []Beat{
			{Slug: "return-check", Brief: "return an error and check it at the call site", Requires: []string{"if-err"}},
			{Slug: "sentinel", Brief: "declare a sentinel error with errors.New and compare with errors.Is", Requires: []string{"call:errors.New", "call:errors.Is"}},
			{Slug: "custom-type", Brief: "define a custom error type and extract it with errors.As", Requires: []string{"call:errors.As", "method"}},
			{Slug: "wrap", Brief: "wrap an error with %w and keep the chain inspectable", Requires: []string{"call:fmt.Errorf"}},
			{Slug: "join", Brief: "collect several failures with errors.Join", Requires: []string{"call:errors.Join"}},
			{Slug: "recover", Brief: "turn a panic into an error with defer and recover", Requires: []string{"call:recover", "defer"}},
		},
	},
	{
		Slug: "goroutines", Title: "Goroutines", Concepts: "go keyword, sync.WaitGroup",
		Guidance: deterministic,
		Beats: []Beat{
			{Slug: "waitgroup", Brief: "start goroutines and wait for them with sync.WaitGroup", Requires: []string{"sync.WaitGroup"}},
			{Slug: "wg-go", Brief: "use WaitGroup.Go to launch and track workers", Requires: []string{"call:.Go"}},
			{Slug: "mutex", Brief: "protect a shared counter with sync.Mutex", Requires: []string{"go", "sync.Mutex"}},
			{Slug: "rwmutex", Brief: "use sync.RWMutex for read-heavy shared state", Requires: []string{"go", "sync.RWMutex"}},
			{Slug: "once", Brief: "initialise a subsystem exactly once with sync.Once", Requires: []string{"sync.Once"}},
			{Slug: "atomic", Brief: "count across goroutines with sync/atomic", Requires: []string{"go", "atomic.*"}},
			{Slug: "collect", Brief: "fan out work and collect results into a slice by index", Requires: []string{"go"}},
		},
	},
	{
		Slug: "channels", Title: "Channels", Concepts: "make, send/recv, buffered, close",
		Guidance: deterministic,
		Beats: []Beat{
			{Slug: "unbuffered", Brief: "hand values between goroutines over an unbuffered channel", Requires: []string{"chan", "go"}},
			{Slug: "buffered", Brief: "use a buffered channel as a queue", Requires: []string{"chan-buffered"}},
			{Slug: "close-range", Brief: "close a channel and drain it with range", Requires: []string{"call:close", "range"}},
			{Slug: "done", Brief: "signal shutdown with a done channel", Requires: []string{"chan", "call:close"}},
			{Slug: "directional", Brief: "type function parameters as chan<- T and <-chan T", Requires: []string{"chan-dir"}},
			{Slug: "pipeline", Brief: "chain two stages into a pipeline", Requires: []string{"chan", "go"}},
			{Slug: "fan-in", Brief: "merge two channels into one", Requires: []string{"chan", "go"}},
		},
	},
	{
		Slug: "select", Title: "Select", Concepts: "select statement, default, timeout pattern",
		Guidance: deterministic + " Never let output depend on which ready case select picks at random.",
		Beats: []Beat{
			{Slug: "two-chans", Brief: "wait on two channels with select", Requires: []string{"select"}},
			{Slug: "default", Brief: "probe a channel without blocking using select with default", Requires: []string{"select-default"}},
			{Slug: "timeout", Brief: "bound a wait with time.After", Requires: []string{"select", "call:time.After"}},
			{Slug: "cancel", Brief: "stop a worker loop when a done channel closes", Requires: []string{"select", "for"}},
			{Slug: "nil-case", Brief: "disable a select case by setting its channel to nil", Requires: []string{"select", "nil"}},
		},
	},
	{
		Slug: "context", Title: "Context", Concepts: "WithCancel, WithTimeout, WithValue",
		Guidance: deterministic,
		Beats: []Beat{
			{Slug: "cancel", Brief: "cancel work with context.WithCancel and ctx.Done()", Requires: []string{"call:context.WithCancel"}},
			{Slug: "timeout", Brief: "cut off a slow subsystem with context.WithTimeout", Requires: []string{"call:context.WithTimeout"}},
			{Slug: "cause", Brief: "cancel with a cause and read it with context.Cause", Requires: []string{"call:context.Cause"}},
			{Slug: "value", Brief: "pass request-scoped metadata with WithValue and an unexported key type", Requires: []string{"call:context.WithValue"}},
			{Slug: "propagate", Brief: "propagate cancellation into several goroutines", Requires: []string{"go", ".Done"}},
		},
	},
	{
		Slug: "io", Title: "io & Readers", Concepts: "io.Reader, io.Writer, io.Copy",
		Beats: []Beat{
			{Slug: "copy", Brief: "move data with io.Copy", Requires: []string{"call:io.Copy"}},
			{Slug: "readall", Brief: "consume a reader with io.ReadAll", Requires: []string{"call:io.ReadAll"}},
			{Slug: "limit", Brief: "cap bytes read with io.LimitReader", Requires: []string{"call:io.LimitReader"}},
			{Slug: "scanner", Brief: "read line by line with bufio.Scanner", Requires: []string{"call:bufio.NewScanner"}},
			{Slug: "buffer", Brief: "use bytes.Buffer as both reader and writer", Requires: []string{"bytes.Buffer"}},
			{Slug: "multiwriter", Brief: "broadcast to two sinks with io.MultiWriter", Requires: []string{"call:io.MultiWriter"}},
			{Slug: "custom-reader", Brief: "implement io.Reader on your own type", Requires: []string{"method"}},
		},
	},
	{
		Slug: "http", Title: "net/http", Concepts: "httptest, http.Get, handlers",
		Guidance: "Never reach the internet. The scaffold starts a local server with httptest.NewServer and exposes its URL in a variable named url; " +
			"the player talks to that url or writes the handler. Always close response bodies.",
		Beats: []Beat{
			{Slug: "get-status", Brief: "GET the url and print the status code", Requires: []string{"call:http.Get"}},
			{Slug: "read-body", Brief: "read and print the response body", Requires: []string{"call:io.ReadAll", "defer"}},
			{Slug: "headers", Brief: "inspect a response header", Requires: []string{".Header"}},
			{Slug: "custom-request", Brief: "build a request with http.NewRequest, add a header, send it with a client", Requires: []string{"call:http.NewRequest"}},
			{Slug: "handler", Brief: "write the http.HandlerFunc the scaffold serves", Requires: []string{"http.ResponseWriter"}},
			{Slug: "mux", Brief: "route two paths with http.NewServeMux and method patterns", Requires: []string{"call:http.NewServeMux"}},
		},
	},
	{
		Slug: "json", Title: "encoding/json", Concepts: "Marshal, Unmarshal, struct tags",
		Beats: []Beat{
			{Slug: "marshal", Brief: "marshal a struct to JSON", Requires: []string{"call:json.Marshal"}},
			{Slug: "unmarshal", Brief: "unmarshal JSON into a struct", Requires: []string{"call:json.Unmarshal"}},
			{Slug: "tags", Brief: "control field names with tags, omitempty and \"-\"", Requires: []string{"struct-tag"}},
			{Slug: "nested", Brief: "decode nested objects and arrays", Requires: []string{"call:json.Unmarshal", "struct"}},
			{Slug: "decoder", Brief: "stream-decode with json.NewDecoder", Requires: []string{"call:json.NewDecoder"}},
			{Slug: "raw", Brief: "delay parsing part of a document with json.RawMessage", Requires: []string{"json.RawMessage"}},
			{Slug: "custom", Brief: "implement MarshalJSON on a type", Requires: []string{"method"}},
		},
	},
	{
		Slug: "files", Title: "os & Files", Concepts: "os.Open, os.Create, bufio.Scanner",
		Guidance: "Work only under os.TempDir() or a directory from os.MkdirTemp. Never use absolute paths.",
		Beats: []Beat{
			{Slug: "write-read", Brief: "write a file with os.WriteFile and read it back", Requires: []string{"call:os.WriteFile", "call:os.ReadFile"}},
			{Slug: "create-open", Brief: "create, write, close, then open and scan a file", Requires: []string{"call:os.Create", "call:os.Open"}},
			{Slug: "append", Brief: "append to a file with os.OpenFile and O_APPEND", Requires: []string{"call:os.OpenFile"}},
			{Slug: "exists", Brief: "check existence with os.Stat and errors.Is(err, fs.ErrNotExist)", Requires: []string{"call:os.Stat", "call:errors.Is"}},
			{Slug: "walk", Brief: "list a directory tree with filepath.WalkDir", Requires: []string{"call:filepath.WalkDir"}},
			{Slug: "root", Brief: "confine file access to one directory with os.OpenRoot", Requires: []string{"call:os.OpenRoot"}},
		},
	},
	{
		Slug: "generics", Title: "Generics", Concepts: "type parameters, constraints, generic types",
		Guidance: "Markers sit at package level.",
		Beats: []Beat{
			{Slug: "func", Brief: "write a generic function over a slice of T", Requires: []string{"generics"}},
			{Slug: "constraint", Brief: "constrain T with cmp.Ordered or a custom ~int | ~float64 union", Requires: []string{"generics", "constraint"}},
			{Slug: "type", Brief: "declare a generic type such as Stack[T] with methods", Requires: []string{"generic-type", "method"}},
			{Slug: "comparable", Brief: "use the comparable constraint for a set or dedupe", Requires: []string{"generics"}},
			{Slug: "two-params", Brief: "write Map[T, U any] that transforms a slice", Requires: []string{"generics"}},
		},
	},
	{
		Slug: "iterators", Title: "Iterators", Concepts: "range over func, iter.Seq, slices.Collect",
		Guidance: "Range-over-func: a function func(yield func(T) bool) can be ranged over; iter.Seq[T] names that type; " +
			"stop when yield returns false. slices.Values, slices.Collect, maps.Keys and slices.Sorted are iterator helpers.",
		Beats: []Beat{
			{Slug: "seq", Brief: "write a function returning iter.Seq[T] and range over it", Requires: []string{"iter.Seq"}},
			{Slug: "early-stop", Brief: "stop an iterator early with break and honour yield's false", Requires: []string{"yield-check"}},
			{Slug: "seq2", Brief: "yield key/value pairs with iter.Seq2", Requires: []string{"iter.Seq2"}},
			{Slug: "collect", Brief: "chain slices.Values, a filter iterator, and slices.Collect", Requires: []string{"call:slices.Collect"}},
		},
	},
	{
		Slug: "strings", Title: "strings & strconv", Concepts: "Builder, Fields, Split, Atoi, Format",
		Beats: []Beat{
			{Slug: "fields", Brief: "split telemetry with strings.Fields or strings.Split", Requires: []string{"strings.*"}},
			{Slug: "builder", Brief: "assemble output with strings.Builder", Requires: []string{"strings.Builder"}},
			{Slug: "atoi", Brief: "parse numbers with strconv.Atoi and handle the error", Requires: []string{"call:strconv.Atoi"}},
			{Slug: "cut", Brief: "split key=value pairs with strings.Cut", Requires: []string{"call:strings.Cut"}},
			{Slug: "format", Brief: "format floats with strconv.FormatFloat or fmt verbs and padding", Requires: []string{"fmt.*"}},
		},
	},
	{
		Slug: "time", Title: "time", Concepts: "Duration, Date, Format, Truncate",
		Guidance: "Never call time.Now or sleep for output: build fixed instants with time.Date in time.UTC so output is deterministic.",
		Beats: []Beat{
			{Slug: "duration", Brief: "do arithmetic with time.Duration and print it", Requires: []string{"time.Duration"}},
			{Slug: "date-math", Brief: "add and subtract times built with time.Date", Requires: []string{"call:time.Date"}},
			{Slug: "format", Brief: "format and parse timestamps with a layout", Requires: []string{"call:time.Parse"}},
			{Slug: "truncate", Brief: "bucket events by truncating times", Requires: []string{".Truncate"}},
		},
	},
	{
		Slug: "slog", Title: "log/slog", Concepts: "structured logging, attrs, handlers",
		Guidance: "Log to os.Stdout with slog.NewTextHandler and a ReplaceAttr that drops slog.TimeKey so output is deterministic.",
		Beats: []Beat{
			{Slug: "text-handler", Brief: "log key/value pairs with a text handler", Requires: []string{"call:slog.NewTextHandler"}},
			{Slug: "levels", Brief: "filter records below a level with HandlerOptions.Level", Requires: []string{"slog.HandlerOptions"}},
			{Slug: "with", Brief: "derive a logger with fixed attrs using With", Requires: []string{".With"}},
			{Slug: "group", Brief: "nest attributes with slog.Group", Requires: []string{"call:slog.Group"}},
		},
	},
}
