package geoip

import (
	"os"
	"testing"
)

func TestLocationString(t *testing.T) {
	tests := []struct {
		location Location
		want     string
	}{
		{location: Location{Country: "Russia", City: "Yekaterinburg"}, want: "Russia, Yekaterinburg"},
		{location: Location{Country: "Russia"}, want: "Russia"},
		{location: Location{}, want: ""},
	}
	for _, tt := range tests {
		if got := tt.location.String(); got != tt.want {
			t.Fatalf("Location.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestLocalizedNameFallsBackToEnglish(t *testing.T) {
	names := map[string]string{"en": "Germany", "ru": "Германия"}
	if got := localizedName(names, "ru-RU"); got != "Германия" {
		t.Fatalf("localized Russian name = %q", got)
	}
	if got := localizedName(names, "unsupported"); got != "Germany" {
		t.Fatalf("fallback name = %q", got)
	}
}

func TestDatabaseLookupWhenConfigured(t *testing.T) {
	path := os.Getenv("TELESRV_TEST_GEOIP_CITY_DB")
	if path == "" {
		t.Skip("TELESRV_TEST_GEOIP_CITY_DB is not configured")
	}
	db, err := OpenCity(path)
	if err != nil {
		t.Fatalf("open City database: %v", err)
	}
	defer func() { _ = db.Close() }()

	location, ok := db.Lookup("8.8.8.8", "en")
	if !ok || location.Country == "" {
		t.Fatalf("lookup 8.8.8.8 = %+v, %v; want a country", location, ok)
	}
	if _, ok := db.Lookup("127.0.0.1", "en"); ok {
		t.Fatal("private/loopback address unexpectedly resolved")
	}
}
