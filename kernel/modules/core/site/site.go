package site

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type ID int64

var (
	ErrNotFound = errors.New("site not found")
	ErrConflict = errors.New("site conflict")

	readPermission = permission.MustCode(
		"core",
		"site",
		permission.Read,
	)
	updatePermission = permission.MustCode(
		"core",
		"site",
		permission.Update,
	)
)

type Site struct {
	ID          ID
	ProfileCode kernel.ProfileCode
	Domain      string
	Locale      string
	Settings    map[string]any
	IsPublic    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   *security.UserID
	UpdatedBy   *security.UserID
}

type Repository interface {
	List(context.Context) ([]Site, error)
	Update(
		context.Context,
		*security.UserID,
		Site,
	) (Site, error)
}

type Access interface {
	security.Authorizer
	IsGuestSubject(context.Context, security.Actor) (bool, error)
}

type UpdateInput struct {
	ID       ID
	Domain   string
	Locale   string
	Settings map[string]any
	IsPublic bool
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
	result.CreatedBy = cloneUserID(result.CreatedBy)
	result.UpdatedBy = cloneUserID(result.UpdatedBy)
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
	access     Access

	snapshot   atomic.Pointer[runtimeSnapshot]
	mutationMu sync.Mutex
}

func NewCatalog(
	repository Repository,
	profiles ProfileResolver,
	access Access,
) (*Catalog, error) {
	if repository == nil {
		return nil, errors.New("site repository is nil")
	}

	if profiles == nil {
		return nil, errors.New("profile resolver is nil")
	}
	if access == nil {
		return nil, errors.New("site access service is nil")
	}

	catalog := &Catalog{
		repository: repository,
		profiles:   profiles,
		access:     access,
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

func (c *Catalog) ResolveByDomain(
	ctx context.Context,
	actor security.Actor,
	domain string,
) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("site resolve context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runtime, exists := c.RuntimeByDomain(domain)
	if !exists {
		return nil, ErrNotFound
	}
	if err := c.access.Check(ctx, actor, readPermission); err != nil {
		return nil, err
	}
	guest, err := c.access.IsGuestSubject(ctx, actor)
	if err != nil {
		return nil, err
	}
	if guest && !runtime.site.IsPublic {
		return nil, security.ErrForbidden
	}
	return runtime, nil
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

func (c *Catalog) Update(
	ctx context.Context,
	actor security.Actor,
	input UpdateInput,
) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("site settings update context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.access.Check(ctx, actor, updatePermission); err != nil {
		return nil, err
	}
	if input.ID <= 0 {
		return nil, errors.New("invalid site id")
	}

	c.mutationMu.Lock()
	defer c.mutationMu.Unlock()

	currentSnapshot := c.snapshot.Load()
	if currentSnapshot == nil {
		return nil, errors.New("site runtime snapshot is nil")
	}

	current, exists := currentSnapshot.byID[input.ID]
	if !exists {
		return nil, ErrNotFound
	}

	item := current.Site()
	item.Domain = input.Domain
	item.Locale = strings.TrimSpace(input.Locale)
	item.Settings = cloneSettings(input.Settings)
	item.IsPublic = input.IsPublic

	nextRuntime, err := NewRuntime(item, current.Profile())
	if err != nil {
		return nil, fmt.Errorf(
			"build updated site runtime %d: %w",
			input.ID,
			err,
		)
	}
	if existing, exists := currentSnapshot.byDomain[nextRuntime.site.Domain]; exists && existing.site.ID != input.ID {
		return nil, ErrConflict
	}

	stored, err := c.repository.Update(
		ctx,
		actor.AuditUserID(),
		nextRuntime.Site(),
	)
	if err != nil {
		return nil, fmt.Errorf("update site: %w", err)
	}
	nextRuntime.site.CreatedAt = stored.CreatedAt
	nextRuntime.site.UpdatedAt = stored.UpdatedAt
	nextRuntime.site.CreatedBy = cloneUserID(stored.CreatedBy)
	nextRuntime.site.UpdatedBy = cloneUserID(stored.UpdatedBy)

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

	delete(nextSnapshot.byDomain, current.site.Domain)
	nextSnapshot.byDomain[nextRuntime.site.Domain] = nextRuntime
	nextSnapshot.byID[input.ID] = nextRuntime
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

func cloneUserID(value *security.UserID) *security.UserID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
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
