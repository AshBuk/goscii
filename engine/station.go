// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

// System is one station subsystem a mission can repair.
type System struct {
	Slug  string
	Name  string
	Brief string // what a mission set here is about
}

// Systems is the station layout. Forged missions are set in one of these.
var Systems = []System{
	{"atmos", "Atmosphere", "oxygen scrubbers and air mix"},
	{"hull", "Hull & airlocks", "hull integrity and airlock cycling"},
	{"nav", "Navigation", "orbital mechanics and course plotting"},
	{"comms", "Comms", "signal processing and transmissions"},
	{"water", "Reclamation", "water and resource recycling"},
	{"power", "Power grid", "reactor output and power distribution"},
	{"dock", "Docking bay", "bay operations and cargo handling"},
	{"drive", "Propulsion", "thrusters and burn planning"},
	{"sensors", "Sensor array", "telemetry and scanning"},
	{"thermal", "Thermal", "heat exchange and radiators"},
	{"beacon", "Emergency", "beacons and alarms"},
	{"crew", "Crew tracking", "life signs and crew manifest"},
	{"medbay", "Medbay", "medical stores and life support"},
	{"gyro", "Attitude", "gyroscopes and orientation"},
}

// FindSystem returns the system with the given slug.
func FindSystem(slug string) (System, bool) {
	for _, s := range Systems {
		if s.Slug == slug {
			return s, true
		}
	}
	return System{}, false
}

const baseIntegrity = 20

// Station is the persistent state of the wreck.
type Station struct {
	Integrity map[string]int `json:"integrity,omitempty"`
}

// Level returns a system's integrity in percent.
func (s Station) Level(slug string) int {
	if v, ok := s.Integrity[slug]; ok {
		return v
	}
	return baseIntegrity
}

// Repair raises a system's integrity, capped at 100. It returns the new level.
func (s *Station) Repair(slug string, amount int) int {
	if s.Integrity == nil {
		s.Integrity = make(map[string]int)
	}
	v := min(100, s.Level(slug)+amount)
	s.Integrity[slug] = v
	return v
}

// Overall is the mean integrity across all systems.
func (s Station) Overall() int {
	total := 0
	for _, sys := range Systems {
		total += s.Level(sys.Slug)
	}
	return total / len(Systems)
}

// Weakest returns the system with the lowest integrity, first in layout order on ties.
func (s Station) Weakest() System {
	best := Systems[0]
	for _, sys := range Systems[1:] {
		if s.Level(sys.Slug) < s.Level(best.Slug) {
			best = sys
		}
	}
	return best
}
