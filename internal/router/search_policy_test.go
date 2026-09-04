package router

import (
	"testing"
)

func TestAlwaysSearchAllCategoriesDefault(t *testing.T) {
	if !AlwaysSearchAllCategories() {
		t.Fatal("default (env unset) must keep the deployed always-search behavior")
	}
}

func TestSearchOnlyCategoriesOptIn(t *testing.T) {
	t.Setenv("SEARCH_ONLY_CATEGORIES", "true")
	if AlwaysSearchAllCategories() {
		t.Fatal("SEARCH_ONLY_CATEGORIES=true must disable always-search")
	}
	if categoryUsesWebSearch(CategoryWebSearch) != true ||
		categoryUsesWebSearch(CategoryFactual) != true ||
		categoryUsesWebSearch(CategorySimple) != false ||
		categoryUsesWebSearch(CategoryCreative) != false ||
		categoryUsesWebSearch(CategoryComplex) != false {
		t.Fatal("categoryUsesWebSearch must cover web_search+factual only")
	}
}

// Route() with the saver on: a simple question must NOT carry a search
// budget; a factual one must.
func TestRouteSearchPolicyByCategory(t *testing.T) {
	t.Setenv("SEARCH_ONLY_CATEGORIES", "true")

	simple := Route("que horas são?")
	if simple.WebSearch {
		t.Errorf("simple question should skip search under SEARCH_ONLY_CATEGORIES, got webSearch=true (category=%s)", simple.Category)
	}

	factual := Route("quem foi Santos Dumont?")
	if !factual.WebSearch {
		t.Errorf("factual question should keep search, got webSearch=false (category=%s)", factual.Category)
	}
}

// Default policy: search everywhere, the deployed behavior.
func TestRouteSearchPolicyDefaultAlwaysOn(t *testing.T) {
	d := Route("que horas são?")
	if !d.WebSearch {
		t.Error("default policy: every routing must include web search")
	}
}
