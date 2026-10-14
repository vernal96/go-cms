package infrastructure

import (
	"reflect"
	"testing"

	connectorredis "github.com/vernal96/go-cms-kernel/connectors/redis"
	viewsredis "github.com/vernal96/go-cms-kernel/modules/core/resourceviews/adapters/redis"
)

func TestResourceViewsUseDedicatedRedisBinding(t *testing.T) {
	definition := (Config{
		RedisAddress: "cache:6379", RedisViewsAddress: "views-cache:6379",
	}).Definition()
	if len(definition.Caches) != 1 {
		t.Fatalf("cache factories = %d, want 1", len(definition.Caches))
	}
	cacheFactory, ok := definition.Caches[0].(connectorredis.Factory)
	if !ok || !reflect.DeepEqual(cacheFactory.Config.Addrs, []string{"cache:6379"}) {
		t.Fatalf("ordinary cache factory = %#v, want cache:6379", definition.Caches[0])
	}
	if len(definition.ResourceViews) != 1 || definition.ResourceViews[0].Code != "views" {
		t.Fatalf("resource view bindings = %#v, want one views binding", definition.ResourceViews)
	}
	viewFactory, ok := definition.ResourceViews[0].Factory.(viewsredis.Factory)
	if !ok {
		t.Fatalf("resource view factory = %T, want Redis accumulator", definition.ResourceViews[0].Factory)
	}
	if !reflect.DeepEqual(viewFactory.Options.Addrs, []string{"views-cache:6379"}) || viewFactory.Prefix != "cms:resource-views" {
		t.Fatalf("resource view Redis = %#v, want dedicated address and stable project prefix", viewFactory)
	}
}
