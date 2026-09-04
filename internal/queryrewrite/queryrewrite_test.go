package queryrewrite

import (
	"testing"
	"time"
)

var now = time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

func TestRewriteSenatorPlate(t *testing.T) {
	got := Rewrite("Acabei de ver a placa 04 de senador, de qual senador ela é?", now)
	if len(got) == 0 {
		t.Fatal("expected at least one query")
	}
	primary := got[0]
	for _, want := range []string{"placa", "04", "senado federal"} {
		if !contains(primary, want) {
			t.Errorf("primary query %q missing %q", primary, want)
		}
	}
	if contains(primary, "acabei") || contains(primary, "ver a") {
		t.Errorf("primary query kept noise: %q", primary)
	}
}

func TestRewriteDollarKeepsRecency(t *testing.T) {
	got := Rewrite("Qual a cotação do dólar hoje?", now)
	if !contains(got[0], "2026") {
		t.Errorf("temporal query missing year: %q", got[0])
	}
	if !contains(got[0], "cotação dólar") {
		t.Errorf("domain hint missing: %q", got[0])
	}
}

func TestRewriteEmpty(t *testing.T) {
	got := Rewrite("", now)
	if len(got) != 1 || got[0] == "" {
		t.Errorf("empty input should return fallback, got %v", got)
	}
}

func TestRewriteSimple(t *testing.T) {
	got := Rewrite("Quem é o presidente do Brasil?", now)
	if len(got) == 0 || got[0] == "" {
		t.Fatalf("expected query, got %v", got)
	}
	if contains(got[0], "quem") {
		t.Errorf("interrogative kept in primary: %q", got[0])
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
