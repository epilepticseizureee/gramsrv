package rpc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/iamxvbaba/td/tg"
	"go.uber.org/zap/zaptest"

	"telesrv/internal/clientaddr"
	"telesrv/internal/domain"
	"telesrv/internal/geoip"
)

type staticGeoIPResolver struct {
	ip       string
	lang     string
	location geoip.Location
}

func (r *staticGeoIPResolver) Lookup(ip, lang string) (geoip.Location, bool) {
	r.ip = ip
	r.lang = lang
	return r.location, r.location.String() != ""
}

func TestAuthzFromCtxPersistsRemoteIP(t *testing.T) {
	ctx := clientaddr.WithRemoteAddr(context.Background(), "[2001:db8::10]:2398")
	got := (&Router{}).authzFromCtx(ctx)
	if got.IP != "2001:db8::10" {
		t.Fatalf("authorization IP = %q, want %q", got.IP, "2001:db8::10")
	}
}

func TestPersistClientInfoRefreshesAuthorizationIP(t *testing.T) {
	authKeyID := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	auth := &captureAuthService{
		authKeyClientInfos: map[[8]byte]domain.AuthKeyClientInfo{},
		authorizations:     []domain.Authorization{{AuthKeyID: authKeyID}},
	}
	r := New(Config{}, Deps{Auth: auth}, zaptest.NewLogger(t), fixedClock{now: time.Unix(1700000000, 0)})
	ctx := WithAuthKeyID(clientaddr.WithRemoteAddr(context.Background(), "8.8.8.8:2398"), authKeyID)

	r.persistAuthKeyClientInfo(ctx, clientSessionInfo{})

	if got := auth.authorizations[0].IP; got != "8.8.8.8" {
		t.Fatalf("refreshed authorization IP = %q, want %q", got, "8.8.8.8")
	}
}

func TestAuthorizationUsesGeoIPCountryAndCity(t *testing.T) {
	resolver := &staticGeoIPResolver{
		location: geoip.Location{Country: "Россия", City: "Екатеринбург"},
	}
	r := New(Config{}, Deps{GeoIP: resolver}, zaptest.NewLogger(t), fixedClock{now: time.Unix(1700000000, 0)})
	ctx := WithClientInfo(context.Background(), ClientInfo{LangCode: "ru"})

	got := r.tgAuthorization(ctx, domain.Authorization{IP: "8.8.8.8"}, [8]byte{}, 1700000000)
	if got.Country != "Россия" || got.Region != "Екатеринбург" {
		t.Fatalf("authorization location = %q, %q", got.Country, got.Region)
	}
	if resolver.ip != "8.8.8.8" || resolver.lang != "ru" {
		t.Fatalf("resolver request = ip %q lang %q", resolver.ip, resolver.lang)
	}
}

func TestSignInNotificationUsesGeoIPLocation(t *testing.T) {
	var authKeyID [8]byte
	authKeyID[0] = 9
	resolver := &staticGeoIPResolver{
		location: geoip.Location{Country: "Россия", City: "Екатеринбург"},
	}
	r := New(Config{}, Deps{GeoIP: resolver}, zaptest.NewLogger(t), fixedClock{now: time.Date(2026, 7, 22, 22, 11, 6, 0, time.UTC)})
	ctx := WithClientInfo(
		clientaddr.WithRemoteAddr(context.Background(), "8.8.8.8:2398"),
		ClientInfo{DeviceModel: "reynard", AppVersion: "1.0", LangCode: "ru"},
	)

	got := r.tgSignInServiceNotification(ctx, domain.User{
		ID:        1000000001,
		FirstName: "Reynard",
	}, authKeyID)
	update := got.Updates[0]
	message := update.(*tg.UpdateServiceNotification).Message
	if !strings.Contains(message, "Россия, Екатеринбург") {
		t.Fatalf("notification %q does not contain resolved location", message)
	}
}
