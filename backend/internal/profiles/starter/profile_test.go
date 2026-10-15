package starter

import "testing"

func TestProfileDoesNotIncludeHomeInBreadcrumbsByDefault(t *testing.T) {
	if Profile.Settings.IncludeHomeInBreadcrumbs {
		t.Error("starter profile must leave home breadcrumbs disabled by default")
	}
}
