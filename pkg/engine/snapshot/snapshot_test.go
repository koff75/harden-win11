package snapshot

import (
	"testing"
)

func TestDiff_RegistryAdded(t *testing.T) {
	before := &Snapshot{Registry: []RegEntry{}}
	after := &Snapshot{Registry: []RegEntry{
		{Path: `HKLM:\Foo`, Name: "Bar", Exists: true, Value: 1.0},
	}}
	d := Diff(before, after)
	if len(d) != 1 || d[0].Change != "added" || d[0].Key != `HKLM:\Foo\Bar` {
		t.Errorf("unexpected diff: %+v", d)
	}
}

func TestDiff_RegistryModified(t *testing.T) {
	before := &Snapshot{Registry: []RegEntry{
		{Path: `HKLM:\Foo`, Name: "Bar", Exists: true, Value: 0.0},
	}}
	after := &Snapshot{Registry: []RegEntry{
		{Path: `HKLM:\Foo`, Name: "Bar", Exists: true, Value: 1.0},
	}}
	d := Diff(before, after)
	if len(d) != 1 || d[0].Change != "modified" {
		t.Errorf("unexpected diff: %+v", d)
	}
}

func TestDiff_RegistryRemoved(t *testing.T) {
	before := &Snapshot{Registry: []RegEntry{
		{Path: `HKLM:\Foo`, Name: "Bar", Exists: true, Value: 0.0},
	}}
	after := &Snapshot{Registry: []RegEntry{}}
	d := Diff(before, after)
	if len(d) != 1 || d[0].Change != "removed" {
		t.Errorf("unexpected diff: %+v", d)
	}
}

func TestDiff_NoChange(t *testing.T) {
	s := &Snapshot{Registry: []RegEntry{
		{Path: `HKLM:\Foo`, Name: "Bar", Exists: true, Value: 1.0},
	}}
	d := Diff(s, s)
	if len(d) != 0 {
		t.Errorf("expected empty diff, got %+v", d)
	}
}

func TestDiff_Service(t *testing.T) {
	before := &Snapshot{Services: []ServiceEntry{
		{Name: "WinDefend", StartType: "Automatic", Status: "Running"},
	}}
	after := &Snapshot{Services: []ServiceEntry{
		{Name: "WinDefend", StartType: "Disabled", Status: "Stopped"},
	}}
	d := Diff(before, after)
	if len(d) != 1 || d[0].Kind != "service" {
		t.Errorf("unexpected diff: %+v", d)
	}
}
