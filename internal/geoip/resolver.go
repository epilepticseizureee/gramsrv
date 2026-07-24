package geoip

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// Location is the user-visible geographic result for an IP address.
type Location struct {
	Country string
	City    string
}

func (l Location) String() string {
	switch {
	case l.Country != "" && l.City != "":
		return l.Country + ", " + l.City
	case l.Country != "":
		return l.Country
	default:
		return l.City
	}
}

// Resolver provides local IP geolocation without network access.
type Resolver interface {
	Lookup(ip, lang string) (Location, bool)
}

// Database is a concurrency-safe GeoLite2/GeoIP2 City database reader.
type Database struct {
	reader *geoip2.Reader
}

func OpenCity(path string) (*Database, error) {
	reader, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	if databaseType := reader.Metadata().DatabaseType; !strings.Contains(databaseType, "City") {
		_ = reader.Close()
		return nil, fmt.Errorf("unsupported MaxMind database type %q: City database required", databaseType)
	}
	return &Database{reader: reader}, nil
}

func (d *Database) Close() error {
	if d == nil || d.reader == nil {
		return nil
	}
	return d.reader.Close()
}

func (d *Database) BuildTime() time.Time {
	if d == nil || d.reader == nil {
		return time.Time{}
	}
	return time.Unix(int64(d.reader.Metadata().BuildEpoch), 0).UTC()
}

func (d *Database) Lookup(ip, lang string) (Location, bool) {
	if d == nil || d.reader == nil {
		return Location{}, false
	}
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return Location{}, false
	}
	addr = addr.Unmap()
	if !isPublicAddress(addr) {
		return Location{}, false
	}
	record, err := d.reader.City(net.IP(addr.AsSlice()))
	if err != nil || record == nil {
		return Location{}, false
	}

	countryNames := record.Country.Names
	if len(countryNames) == 0 {
		countryNames = record.RegisteredCountry.Names
	}
	if len(countryNames) == 0 {
		countryNames = record.RepresentedCountry.Names
	}
	location := Location{
		Country: localizedName(countryNames, lang),
		City:    localizedName(record.City.Names, lang),
	}
	return location, location.Country != "" || location.City != ""
}

func isPublicAddress(addr netip.Addr) bool {
	return addr.IsValid() &&
		addr.IsGlobalUnicast() &&
		!addr.IsPrivate() &&
		!addr.IsLoopback() &&
		!addr.IsLinkLocalUnicast()
}

func localizedName(names map[string]string, lang string) string {
	if len(names) == 0 {
		return ""
	}
	for _, candidate := range languageCandidates(lang) {
		if value := strings.TrimSpace(names[candidate]); value != "" {
			return value
		}
		for key, value := range names {
			if strings.EqualFold(key, candidate) {
				if value = strings.TrimSpace(value); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func languageCandidates(lang string) []string {
	normalized := strings.ReplaceAll(strings.TrimSpace(lang), "_", "-")
	normalized = strings.ToLower(normalized)
	base := normalized
	if i := strings.IndexByte(base, '-'); i >= 0 {
		base = base[:i]
	}
	out := make([]string, 0, 4)
	add := func(value string) {
		if value == "" {
			return
		}
		for _, existing := range out {
			if existing == value {
				return
			}
		}
		out = append(out, value)
	}
	add(normalized)
	add(base)
	switch normalized {
	case "pt-br":
		add("pt-BR")
	case "zh-cn", "zh-hans":
		add("zh-CN")
	case "zh-tw", "zh-hant":
		add("zh-TW")
	}
	add("en")
	return out
}
