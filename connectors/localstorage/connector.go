package localstorage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel/filesystem"
)

type Config struct {
	Code       filesystem.Code
	Visibility filesystem.Visibility
	Root       string
	BaseURL    string
	SigningKey string
}

type Factory struct {
	Config Config
}

func (f Factory) Code() filesystem.Code {
	return f.Config.Code
}

func (f Factory) Open(ctx context.Context) (filesystem.Disk, error) {
	return New(ctx, f.Config)
}

type Connector struct {
	code       filesystem.Code
	visibility filesystem.Visibility
	root       string
	baseURL    *url.URL
	signingKey []byte
}

func New(ctx context.Context, config Config) (*Connector, error) {
	if ctx == nil {
		return nil, errors.New("localstorage context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if config.Code == "" {
		return nil, errors.New("localstorage code is empty")
	}
	if !filesystem.ValidVisibility(config.Visibility) {
		return nil, fmt.Errorf(
			"localstorage %q: %w: %q",
			config.Code,
			filesystem.ErrInvalidVisibility,
			config.Visibility,
		)
	}
	if strings.TrimSpace(config.Root) == "" {
		return nil, errors.New("localstorage root is empty")
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, errors.New("localstorage base URL is empty")
	}
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil || !baseURL.IsAbs() || baseURL.Host == "" {
		return nil, fmt.Errorf("invalid localstorage base URL %q", config.BaseURL)
	}
	if config.Visibility == filesystem.VisibilityPrivate &&
		len(config.SigningKey) < 32 {
		return nil, errors.New(
			"private localstorage signing key must contain at least 32 bytes",
		)
	}

	root, err := filepath.Abs(config.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve localstorage root: %w", err)
	}
	root = filepath.Clean(root)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create localstorage root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("canonicalize localstorage root: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("protect localstorage root: %w", err)
	}

	return &Connector{
		code:       config.Code,
		visibility: config.Visibility,
		root:       root,
		baseURL:    baseURL,
		signingKey: []byte(config.SigningKey),
	}, nil
}

func (c *Connector) Code() filesystem.Code {
	return c.code
}

func (c *Connector) Visibility() filesystem.Visibility {
	return c.visibility
}

func (c *Connector) Ping(ctx context.Context) error {
	if ctx == nil {
		return errors.New("localstorage ping context is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Stat(c.root)
	if err != nil {
		return fmt.Errorf("stat localstorage root: %w", err)
	}
	if !info.IsDir() {
		return errors.New("localstorage root is not a directory")
	}
	return nil
}

func (c *Connector) PutNew(
	ctx context.Context,
	key string,
	source io.Reader,
	_ string,
) error {
	if ctx == nil {
		return errors.New("localstorage put context is nil")
	}
	if source == nil {
		return errors.New("localstorage source is nil")
	}
	target, err := c.resolve(key)
	if err != nil {
		return err
	}
	parent := filepath.Dir(target)
	if err := c.ensureNoSymlinks(parent); err != nil {
		return err
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create localstorage object directory: %w", err)
	}
	if err := c.ensureNoSymlinks(parent); err != nil {
		return err
	}

	temp, err := os.CreateTemp(parent, ".upload-*")
	if err != nil {
		return fmt.Errorf("create localstorage temporary object: %w", err)
	}
	tempName := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}()
	if err := temp.Chmod(0o600); err != nil {
		return fmt.Errorf("protect localstorage temporary object: %w", err)
	}

	if _, err := io.Copy(temp, &contextReader{ctx: ctx, reader: source}); err != nil {
		return fmt.Errorf("write localstorage object: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync localstorage object: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close localstorage object: %w", err)
	}

	// A hard link publishes the completed temporary file atomically and fails
	// instead of replacing an existing object.
	if err := os.Link(tempName, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return filesystem.ErrConflict
		}
		return fmt.Errorf("publish localstorage object: %w", err)
	}
	if err := os.Chmod(target, 0o600); err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("protect localstorage object: %w", err)
	}
	return nil
}

func (c *Connector) Open(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	if ctx == nil {
		return nil, errors.New("localstorage open context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target, err := c.resolve(key)
	if err != nil {
		return nil, err
	}
	if err := c.ensureNoSymlinks(target); err != nil {
		return nil, err
	}
	result, err := os.Open(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, filesystem.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open localstorage object: %w", err)
	}
	return result, nil
}

func (c *Connector) Delete(ctx context.Context, key string) error {
	if ctx == nil {
		return errors.New("localstorage delete context is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	target, err := c.resolve(key)
	if err != nil {
		return err
	}
	if err := c.ensureNoSymlinks(target); err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete localstorage object: %w", err)
	}
	return nil
}

func (c *Connector) URL(
	_ context.Context,
	reference filesystem.Reference,
) (string, error) {
	if c.visibility != filesystem.VisibilityPublic {
		return "", filesystem.ErrInvalidVisibility
	}
	return c.deliveryURL(reference, time.Time{}, "")
}

func (c *Connector) TemporaryURL(
	_ context.Context,
	reference filesystem.Reference,
	expiresAt time.Time,
) (string, error) {
	if c.visibility != filesystem.VisibilityPrivate {
		return "", filesystem.ErrInvalidVisibility
	}
	if !expiresAt.After(time.Now()) {
		return "", errors.New("temporary URL expiration must be in the future")
	}
	signature := c.signature(reference, expiresAt)
	return c.deliveryURL(reference, expiresAt, signature)
}

func (c *Connector) VerifyTemporaryURL(
	reference filesystem.Reference,
	expiresAt time.Time,
	signature string,
) error {
	if c.visibility != filesystem.VisibilityPrivate {
		return filesystem.ErrInvalidVisibility
	}
	if !expiresAt.After(time.Now()) {
		return filesystem.ErrUnauthorized
	}
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return filesystem.ErrUnauthorized
	}
	expected, _ := hex.DecodeString(c.signature(reference, expiresAt))
	if !hmac.Equal(provided, expected) {
		return filesystem.ErrUnauthorized
	}
	return nil
}

func (c *Connector) Close() error {
	return nil
}

func (c *Connector) deliveryURL(
	reference filesystem.Reference,
	expiresAt time.Time,
	signature string,
) (string, error) {
	if reference.ID == "" {
		return "", errors.New("filesystem reference id is empty")
	}
	result := *c.baseURL
	result.Path = strings.TrimRight(result.Path, "/") +
		"/_cms/files/" + url.PathEscape(reference.ID)
	query := result.Query()
	if !expiresAt.IsZero() {
		query.Set("expires", strconv.FormatInt(expiresAt.Unix(), 10))
		query.Set("signature", signature)
	}
	result.RawQuery = query.Encode()
	return result.String(), nil
}

func (c *Connector) signature(
	reference filesystem.Reference,
	expiresAt time.Time,
) string {
	mac := hmac.New(sha256.New, c.signingKey)
	_, _ = io.WriteString(mac, reference.ID)
	_, _ = io.WriteString(mac, "\n")
	_, _ = io.WriteString(mac, string(c.code))
	_, _ = io.WriteString(mac, "\n")
	_, _ = io.WriteString(mac, strconv.FormatInt(expiresAt.Unix(), 10))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Connector) resolve(key string) (string, error) {
	if key == "" || strings.ContainsRune(key, '\x00') {
		return "", errors.New("localstorage object key is invalid")
	}
	key = filepath.FromSlash(key)
	if filepath.IsAbs(key) {
		return "", errors.New("localstorage object key is absolute")
	}
	target := filepath.Clean(filepath.Join(c.root, key))
	relative, err := filepath.Rel(c.root, target)
	if err != nil ||
		relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("localstorage object key escapes root")
	}
	return target, nil
}

func (c *Connector) ensureNoSymlinks(target string) error {
	relative, err := filepath.Rel(c.root, target)
	if err != nil ||
		relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("localstorage object path escapes root")
	}
	current := c.root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect localstorage object path: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("localstorage object path contains a symbolic link")
		}
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(target []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(target)
}

var _ filesystem.Disk = (*Connector)(nil)
var _ filesystem.Factory = Factory{}
var _ filesystem.TemporaryURLVerifier = (*Connector)(nil)
