package main

import (
	"reflect"
	"testing"
)

func TestSetDifference(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{"disjoint", []string{"x", "y"}, []string{"z"}, []string{"x", "y"}},
		{"full overlap", []string{"x", "y"}, []string{"x", "y"}, nil},
		{"partial overlap", []string{"x", "y", "z"}, []string{"y"}, []string{"x", "z"}},
		{"empty a", nil, []string{"x"}, nil},
		{"empty b", []string{"x", "y"}, nil, []string{"x", "y"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setDifference(tt.a, tt.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("setDifference(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestClosedBoardURLs_EmptyURLsGuard(t *testing.T) {
	openURLs := []string{"https://boards.example.com/acme/1", "https://boards.example.com/acme/2"}

	if got := closedBoardURLs(openURLs, nil); got != nil {
		t.Errorf("closedBoardURLs with empty urls = %v, want nil (refuse mass-closure)", got)
	}

	want := []string{"https://boards.example.com/acme/2"}
	got := closedBoardURLs(openURLs, []string{"https://boards.example.com/acme/1"})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("closedBoardURLs = %v, want %v", got, want)
	}
}
