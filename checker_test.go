package main

import "testing"

func TestBuildId(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{
			name: "example.com",
			id:   "example_com",
		},
		{
			name: "Some WeIrD 12345//- stuff.whatever",
			id:   "some_weird_12345_stuff_whatever",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := buildId(test.name)
			if id != test.id {
				t.Errorf("Id has been built wrong: %s", id)
			}
		})
	}
}
