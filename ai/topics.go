// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package ai

// StdlibTopics is the catalog of Go standard library topics available for AI-generated missions.
var StdlibTopics = []Topic{
	{"variables", "Variables & Types", "var, const, basic types, zero values"},
	{"functions", "Functions", "signatures, multiple returns, variadic"},
	{"control-flow", "Control Flow", "if, switch, for, range"},
	{"slices", "Slices", "append, copy, slicing, capacity"},
	{"maps", "Maps", "make, delete, range, nil maps"},
	{"structs", "Structs & Methods", "struct literals, value/pointer receivers"},
	{"interfaces", "Interfaces", "implicit impl, empty interface, type assertion"},
	{"errors", "Errors", "error type, fmt.Errorf, errors.Is/As"},
	{"goroutines", "Goroutines", "go keyword, sync.WaitGroup"},
	{"channels", "Channels", "make, send/recv, buffered, close"},
	{"select", "Select", "select statement, default, timeout pattern"},
	{"context", "Context", "WithCancel, WithTimeout, WithValue"},
	{"io", "io & Readers", "io.Reader, io.Writer, io.Copy"},
	{"http", "net/http", "http.Get, http.HandleFunc, ResponseWriter"},
	{"json", "encoding/json", "Marshal, Unmarshal, struct tags"},
	{"files", "os & Files", "os.Open, os.Create, bufio.Scanner"},
}
