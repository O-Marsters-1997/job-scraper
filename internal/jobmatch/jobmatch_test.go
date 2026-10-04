package jobmatch_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/jobmatch"
)

func TestTitle(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"lowercases", "Senior Backend Engineer", "senior backend engineer"},
		{"strips m/f/d tag and remote suffix", "Senior Backend Engineer (m/f/d) - Remote, UK", "senior backend engineer"},
		{"strips m/w/d tag", "Backend Engineer (m/w/d)", "backend engineer"},
		{"strips f/m/x tag", "Backend Engineer (f/m/x)", "backend engineer"},
		{"strips spaced and bracketed gender tag", "Backend Engineer [M / W / D]", "backend engineer"},
		{"strips all genders tag", "Backend Engineer (all genders)", "backend engineer"},
		{"strips country suffix", "Senior Fullstack Engineer - UK", "senior fullstack engineer"},
		{"strips en dash city suffix", "Data Engineer – London", "data engineer"},
		{"strips pipe arrangement suffix", "Data Engineer | Hybrid", "data engineer"},
		{"strips stacked suffixes", "Data Engineer - London - Hybrid", "data engineer"},
		{"strips bracketed arrangement suffix", "Data Engineer (Remote)", "data engineer"},
		{"strips bracketed city and arrangement", "Data Engineer (London, Hybrid)", "data engineer"},
		{"strips remote first suffix", "Platform Engineer - Remote First", "platform engineer"},
		{"keeps specialism suffix", "Software Engineer - Payments", "software engineer payments"},
		{"keeps bracketed specialism", "Software Engineer (Python)", "software engineer python"},
		{"keeps a title that is only a location word", "Remote", "remote"},
		{"collapses punctuation and whitespace", "  Full-Stack   Engineer, Node.js ", "full stack engineer node js"},
		{"blank stays blank", "   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jobmatch.Title(tt.raw); got != tt.want {
				t.Errorf("Title(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestTitleKeys(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		same bool
	}{
		{"tagged vs untagged", "Senior Backend Engineer (m/f/d) - Remote, UK", "Senior Backend Engineer", true},
		{"country suffix vs none", "Senior Fullstack Engineer - UK", "Senior Fullstack Engineer", true},
		{"senior vs staff", "Senior Backend Engineer", "Staff Backend Engineer", false},
		{"senior vs unprefixed", "Senior Backend Engineer", "Backend Engineer", false},
		{"staff vs unprefixed", "Staff Backend Engineer", "Backend Engineer", false},
		{"backend vs frontend", "Senior Backend Engineer", "Senior Frontend Engineer", false},
		{"senior full stack vs full stack", "Senior Full Stack Engineer", "Full Stack Engineer", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := jobmatch.Title(tt.a), jobmatch.Title(tt.b)
			if (a == b) != tt.same {
				t.Errorf("Title(%q) = %q, Title(%q) = %q, want equal = %v", tt.a, a, tt.b, b, tt.same)
			}
		})
	}
}

func TestLocation(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"city and region", "London, London", "london"},
		{"city and country code", "London, GB", "london"},
		{"city, region and country", "London, England, United Kingdom", "london"},
		{"trims and lowercases", "  Manchester  ", "manchester"},
		{"drops bracketed arrangement", "London (Hybrid)", "london"},
		{"blank", "  ", ""},
		{"remote", "Remote", "remote"},
		{"remote with country", "Remote, UK", "remote"},
		{"country then remote", "UK (Remote)", "remote"},
		{"anywhere", "Anywhere", "remote"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jobmatch.Location(tt.raw); got != tt.want {
				t.Errorf("Location(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestLocationsCompatible(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"same city", "london", "london", true},
		{"city vs remote", "london", "remote", true},
		{"remote vs city", "remote", "london", true},
		{"city vs blank", "london", "", true},
		{"blank vs city", "", "london", true},
		{"both blank", "", "", true},
		{"different cities", "london", "manchester", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jobmatch.LocationsCompatible(tt.a, tt.b); got != tt.want {
				t.Errorf("LocationsCompatible(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestLocationsCompatibleOnRawLondonForms(t *testing.T) {
	london := jobmatch.Location("London, England, United Kingdom")
	tests := []struct {
		name  string
		other string
		want  bool
	}{
		{"london vs london gb", "London, GB", true},
		{"london vs remote", "Remote", true},
		{"london vs blank", "", true},
		{"london vs manchester", "Manchester, England, United Kingdom", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			other := jobmatch.Location(tt.other)
			if got := jobmatch.LocationsCompatible(london, other); got != tt.want {
				t.Errorf("LocationsCompatible(%q, %q) = %v, want %v", london, other, got, tt.want)
			}
		})
	}
}
