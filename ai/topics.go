// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

// StdlibTopics is the catalog of Go standard library topics available for AI-generated missions.
var StdlibTopics = []Topic{
	{
		Slug:     "variables",
		Title:    "Variables & Types",
		Concepts: "var, const, basic types, zero values",
		Guidance: "Build progression with chain step: early — one variable, one type, simple print. " +
			"Later — multiple vars of different types (int, float64, bool, string, byte, rune, uint, complex128), mix var and :=, use const for fixed thresholds, explore zero values, type conversions, multiple assignment, blank identifier _. " +
			"Each mission must use a type or declaration style not used in the previous one.",
	},
	{
		Slug:     "functions",
		Title:    "Functions",
		Concepts: "signatures, multiple returns, variadic",
		Guidance: "Rotate one shape per mission — never repeat the same signature twice in a chain: " +
			"single return → multiple returns (value+error) → named returns → variadic (...T) → function as value → closure → recursive → defer inside function → init-style setup function. " +
			"Early: simple calculation. Later: error-returning, higher-order, and deferred functions.",
	},
	{
		Slug:     "control-flow",
		Title:    "Control Flow",
		Concepts: "if, switch, for, range",
		Guidance: "Strict rotation — one construct per mission: if/else → if with init statement → switch (expression) → switch (no condition, like if-else chain) → switch with fallthrough → for (classic) → for (as while) → range over slice → range over string (runes) → range over map → labeled break/continue. " +
			"Early: single condition. Later: nested logic, multiple cases, fallthrough.",
	},
	{
		Slug:     "slices",
		Title:    "Slices",
		Concepts: "append, copy, slicing, capacity",
		Guidance: "Rotate: append single → append multiple/spread → copy (show src unchanged) → sub-slicing [low:high] → three-index slice [a:b:c] to limit capacity → make([]T, len, cap) and show cap → nil slice vs empty slice ([]T{}) distinction → filter with append → reverse in place → 2D slice. " +
			"Print intermediate len/cap states to make memory behaviour visible.",
	},
	{
		Slug:     "maps",
		Title:    "Maps",
		Concepts: "make, delete, range, nil maps",
		Guidance: "Rotate: map literal initialisation {k: v} → make + set + get → comma-ok existence check → delete a key → frequency counting → grouping (map of slices) → iterate in sorted key order → map as set pattern (map[T]struct{}) → map as lookup table. " +
			"Later missions combine multiple operations.",
	},
	{
		Slug:     "structs",
		Title:    "Structs & Methods",
		Concepts: "struct literals, value/pointer receivers",
		Guidance: "Rotate: plain struct literal → value receiver method → pointer receiver (mutates state) → embedding one struct in another (show promoted methods) → anonymous struct → builder-style (method returns *T) → struct with json/yaml tags → struct implementing an interface → comparing structs with ==. " +
			"Early: one struct, one method. Later: composed structs with multiple methods.",
	},
	{
		Slug:     "interfaces",
		Title:    "Interfaces",
		Concepts: "implicit impl, empty interface, type assertion",
		Guidance: "Rotate: define interface + two concrete types → call via interface → comma-ok type assertion → type switch over 3+ types → any (empty interface) as parameter → interface composition → fmt.Stringer (implement String() string) → use io.Reader or io.Writer as interface example → nil interface pitfall (typed nil vs untyped nil). " +
			"Early: simple interface + two impls. Later: type switches, composition, standard library interfaces.",
	},
	{
		Slug:     "errors",
		Title:    "Errors",
		Concepts: "error type, fmt.Errorf, errors.Is/As",
		Guidance: "Rotate: return plain error → fmt.Errorf with context (no wrap) → sentinel error with var ErrX = errors.New(...) → custom error type (struct implementing error) → wrapping with %w → errors.Is for sentinel → errors.As to unwrap to concrete type → errors.Join (Go 1.20+) to combine multiple errors → panic + recover. " +
			"Early: simple return+check. Later: wrapped errors with full context chain.",
	},
	{
		Slug:     "goroutines",
		Title:    "Goroutines",
		Concepts: "go keyword, sync.WaitGroup",
		Guidance: "Rotate: single goroutine + WaitGroup → multiple goroutines fan-out → sync.Mutex protecting shared counter → sync.RWMutex for read-heavy state → sync.Once for one-time init → sync/atomic for lock-free counter → goroutine with closure capture → fan-out then collect results into slice. " +
			"Output must be deterministic: sort results or use fixed counts before printing.",
	},
	{
		Slug:     "channels",
		Title:    "Channels",
		Concepts: "make, send/recv, buffered, close",
		Guidance: "Rotate: unbuffered send/recv → buffered channel as message queue → close + range to drain → done channel for shutdown signal → direction-typed channels (chan<- T sender, <-chan T receiver) → pipeline (goroutine A feeds B via typed channels) → fan-in (merge two <-chan T into one) → nil channel (blocks forever, use in select to disable a case). " +
			"Output must be deterministic.",
	},
	{
		Slug:     "select",
		Title:    "Select",
		Concepts: "select statement, default, timeout pattern",
		Guidance: "Rotate: select between two channels → select with default (non-blocking probe) → timeout with time.After → cancellation via done channel → fan-in two sources with select → priority select (try one channel before falling to select) → disable a case with nil channel.",
	},
	{
		Slug:     "context",
		Title:    "Context",
		Concepts: "WithCancel, WithTimeout, WithValue",
		Guidance: "Rotate: WithCancel + cancel() + ctx.Done() → WithTimeout simulating slow subsystem → WithDeadline for absolute cutoff → WithValue passing request-scoped metadata → chained contexts (timeout inside cancel) → propagating cancellation into goroutines → context.Background() vs context.TODO() distinction.",
	},
	{
		Slug:     "io",
		Title:    "io & Readers",
		Concepts: "io.Reader, io.Writer, io.Copy",
		Guidance: "Rotate: strings.NewReader as data feed → io.Copy between reader and writer → io.ReadAll to consume a reader → io.LimitReader to cap bytes read → bufio.Scanner line-by-line → bytes.Buffer as both reader and writer → io.MultiWriter broadcast to two sinks → io.TeeReader (read + log simultaneously) → io.Pipe for synchronous reader/writer pair.",
	},
	{
		Slug:     "http",
		Title:    "net/http",
		Concepts: "http.Get, http.HandleFunc, ResponseWriter",
		Guidance: "Always use \"http://example.com\" as the target URL — no other URLs allowed. " +
			"Rotate: GET + print status code → GET + io.ReadAll body + print first N bytes → GET + inspect response headers → http.NewRequest with custom header + http.Client.Do → http.Client with Timeout set → check resp.StatusCode for errors → always defer resp.Body.Close().",
	},
	{
		Slug:     "json",
		Title:    "encoding/json",
		Concepts: "Marshal, Unmarshal, struct tags",
		Guidance: "Rotate: Marshal struct to JSON string → Unmarshal JSON into struct → struct tags (json:\"name\", omitempty, json:\"-\") → nested struct serialisation → Marshal map[string]any → Unmarshal into map → json.Decoder from strings.NewReader (streaming decode) → json.Encoder to bytes.Buffer (streaming encode) → json.RawMessage to delay parsing → custom MarshalJSON / UnmarshalJSON method. " +
			"Early: flat struct. Later: nested, omitempty, streaming, custom marshaler.",
	},
	{
		Slug:     "files",
		Title:    "os & Files",
		Concepts: "os.Open, os.Create, bufio.Scanner",
		Guidance: "Rotate: os.WriteFile + os.ReadFile → os.Create + Write + os.Open + Read → bufio.Scanner line-by-line → append with os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644) → os.MkdirTemp + file inside it → check existence with os.Stat + errors.Is(err, os.ErrNotExist) → os.Remove after use → filepath.Join for cross-platform paths → filepath.Walk to list directory contents → os.Rename to move/rename a file. " +
			"Always write to os.TempDir() paths — never hardcoded absolute paths.",
	},
}
