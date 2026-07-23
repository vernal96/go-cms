package site

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/vernal96/go-cms/kernel"
)

type ID int64

var ErrNotFound = errors.New("site not found")

type Site struct {
	ID          ID
	ProfileCode kernel.ProfileCode
	Domain      string
	Locale      string
	Settings    map[string]any
}

type Repository interface {
	List(context.Context) ([]Site, error)
	UpdateSettings(context.Context, ID, map[string]any) error
}

type Runtime struct {
	site           Site
	profileRuntime *kernel.ProfileRuntime
}

func NewRuntime(
	item Site,
	profileRuntime *kernel.ProfileRuntime,
) (*Runtime, error) {
	if item.ID <= 0 {
		return nil, errors.New("invalid site id")
	}

	if item.ProfileCode == "" {
		return nil, errors.New("site profile code is empty")
	}

	domain, err := NormalizeDomain(item.Domain)
	if err != nil {
		return nil, err
	}

	item.Domain = domain

	if item.Locale == "" {
		return nil, errors.New("site locale is empty")
	}

	if profileRuntime == nil {
		return nil, errors.New("profile runtime is nil")
	}

	if item.ProfileCode != profileRuntime.Profile().Code {
		return nil, fmt.Errorf(
			"site profile %q does not match runtime profile %q",
			item.ProfileCode,
			profileRuntime.Profile().Code,
		)
	}

	paramSchema := profileRuntime.ParamSchema()
	if paramSchema == nil {
		return nil, errors.New("profile param schema is nil")
	}

	settings, err := paramSchema.Validate(item.Settings)
	if err != nil {
		return nil, fmt.Errorf("validate site settings: %w", err)
	}
	item.Settings = cloneSettings(settings)

	return &Runtime{
		site:           item,
		profileRuntime: profileRuntime,
	}, nil
}

func (r *Runtime) Site() Site {
	result := r.site
	result.Settings = cloneSettings(result.Settings)
	return result
}

func (r *Runtime) Profile() *kernel.ProfileRuntime {
	return r.profileRuntime
}

type ProfileResolver interface {
	ProfileRuntime(
		kernel.ProfileCode,
	) (*kernel.ProfileRuntime, bool)
}

type runtimeSnapshot struct {
	byDomain map[string]*Runtime
	byID     map[ID]*Runtime
}

type Catalog struct {
	repository Repository
	profiles   ProfileResolver

	snapshot   atomic.Pointer[runtimeSnapshot]
	mutationMu sync.Mutex
}

func NewCatalog(
	repository Repository,
	profiles ProfileResolver,
) (*Catalog, error) {
	if repository == nil {
		return nil, errors.New("site repository is nil")
	}

	if profiles == nil {
		return nil, errors.New("profile resolver is nil")
	}

	catalog := &Catalog{
		repository: repository,
		profiles:   profiles,
	}

	catalog.snapshot.Store(&runtimeSnapshot{
		byDomain: make(map[string]*Runtime),
		byID:     make(map[ID]*Runtime),
	})

	return catalog, nil
}

func (c *Catalog) RuntimeByDomain(
	domain string,
) (*Runtime, bool) {
	domain, err := NormalizeDomain(domain)
	if err != nil {
		return nil, false
	}

	snapshot := c.snapshot.Load()
	if snapshot == nil {
		return nil, false
	}

	runtime, exists := snapshot.byDomain[domain]
	return runtime, exists
}

func (c *Catalog) RuntimeByID(
	id ID,
) (*Runtime, bool) {
	if id <= 0 {
		return nil, false
	}

	snapshot := c.snapshot.Load()
	if snapshot == nil {
		return nil, false
	}

	runtime, exists := snapshot.byID[id]
	return runtime, exists
}

func (c *Catalog) Reload(ctx context.Context) error {
	if ctx == nil {
		return errors.New("site reload context is nil")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	c.mutationMu.Lock()
	defer c.mutationMu.Unlock()

	sites, err := c.repository.List(ctx)
	if err != nil {
		return fmt.Errorf("list sites: %w", err)
	}

	next := &runtimeSnapshot{
		byDomain: make(map[string]*Runtime, len(sites)),
		byID:     make(map[ID]*Runtime, len(sites)),
	}

	for index, item := range sites {
		profileRuntime, exists := c.profiles.ProfileRuntime(
			item.ProfileCode,
		)
		if !exists {
			return fmt.Errorf(
				"site at index %d references unknown profile %q",
				index,
				item.ProfileCode,
			)
		}

		runtime, err := NewRuntime(item, profileRuntime)
		if err != nil {
			return fmt.Errorf(
				"build site runtime at index %d with id %d: %w",
				index,
				item.ID,
				err,
			)
		}

		domain := runtime.site.Domain

		if _, exists := next.byDomain[domain]; exists {
			return fmt.Errorf(
				"duplicate normalized site domain %q",
				domain,
			)
		}
		if _, exists := next.byID[item.ID]; exists {
			return fmt.Errorf(
				"duplicate site id %d",
				item.ID,
			)
		}

		next.byDomain[domain] = runtime
		next.byID[item.ID] = runtime
	}

	c.snapshot.Store(next)
	return nil
}

func (c *Catalog) UpdateSettings(
	ctx context.Context,
	id ID,
	values map[string]any,
) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("site settings update context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, errors.New("invalid site id")
	}

	c.mutationMu.Lock()
	defer c.mutationMu.Unlock()

	currentSnapshot := c.snapshot.Load()
	if currentSnapshot == nil {
		return nil, errors.New("site runtime snapshot is nil")
	}

	current, exists := currentSnapshot.byID[id]
	if !exists {
		return nil, ErrNotFound
	}

	item := current.Site()
	item.Settings = cloneSettings(values)

	nextRuntime, err := NewRuntime(item, current.Profile())
	if err != nil {
		return nil, fmt.Errorf(
			"build updated site runtime %d: %w",
			id,
			err,
		)
	}

	normalized := nextRuntime.Site().Settings
	if err := c.repository.UpdateSettings(ctx, id, normalized); err != nil {
		return nil, fmt.Errorf("update site settings: %w", err)
	}

	nextSnapshot := &runtimeSnapshot{
		byDomain: make(
			map[string]*Runtime,
			len(currentSnapshot.byDomain),
		),
		byID: make(
			map[ID]*Runtime,
			len(currentSnapshot.byID),
		),
	}
	for domain, runtime := range currentSnapshot.byDomain {
		nextSnapshot.byDomain[domain] = runtime
	}
	for currentID, runtime := range currentSnapshot.byID {
		nextSnapshot.byID[currentID] = runtime
	}

	nextSnapshot.byDomain[nextRuntime.site.Domain] = nextRuntime
	nextSnapshot.byID[id] = nextRuntime
	c.snapshot.Store(nextSnapshot)

	return nextRuntime, nil
}

func NormalizeDomain(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("site domain is empty")
	}

	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}

	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	value = strings.TrimRight(value, ".")
	value = strings.ToLower(strings.TrimSpace(value))

	if value == "" {
		return "", errors.New("site domain is empty")
	}

	if net.ParseIP(value) == nil &&
		strings.ContainsAny(value, " /\\@:#") {
		return "", fmt.Errorf(
			"invalid site domain %q",
			value,
		)
	}

	return value, nil
}

func cloneSettings(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}

	result := make(map[string]any, len(source))

	for key, value := range source {
		result[key] = cloneSettingValue(value)
	}

	return result
}

func cloneSettingValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneSettings(typed)

	case []any:
		result := make([]any, len(typed))

		for index, item := range typed {
			result[index] = cloneSettingValue(item)
		}

		return result

	case []string:
		return append([]string(nil), typed...)

	default:
		return typed
	}
}
