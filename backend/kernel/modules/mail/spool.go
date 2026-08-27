package mail

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/messageid"
)

const SpoolFilesystemAlias filesystem.Alias = "spool"
const spoolPrefix = "mail-spool/"

var errSpoolCleanupLimit = errors.New("mail spool cleanup scan limit reached")

type AttachmentSpool struct{ disk filesystem.Disk }

func NewAttachmentSpool(disk filesystem.Disk) (*AttachmentSpool, error) {
	if disk == nil {
		return nil, errors.New("mail attachment spool disk is nil")
	}
	if disk.Visibility() != filesystem.VisibilityPrivate {
		return nil, errors.New("mail attachment spool must be private")
	}
	if _, ok := disk.(filesystem.PrefixWalker); !ok {
		return nil, errors.New("mail attachment spool does not support bounded cleanup")
	}
	return &AttachmentSpool{disk: disk}, nil
}

func (s *AttachmentSpool) Put(ctx context.Context, input TransientAttachment, maxSize int64) (Attachment, error) {
	if s == nil || s.disk == nil {
		return Attachment{}, errors.New("mail transient attachments are disabled")
	}
	filename := strings.TrimSpace(input.Filename)
	if filename == "" || filepath.Base(filename) != filename || filename == "." {
		return Attachment{}, fmt.Errorf("%w: transient attachment filename is invalid", ErrInvalid)
	}
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(input.MIMEType))
	if err != nil || mediaType == "" {
		return Attachment{}, fmt.Errorf("%w: transient attachment MIME type is invalid", ErrInvalid)
	}
	if input.Body == nil || input.Size < 0 || maxSize < 1 || input.Size > maxSize {
		return Attachment{}, fmt.Errorf("%w: transient attachment size is invalid", ErrInvalid)
	}
	id, err := messageid.New()
	if err != nil {
		return Attachment{}, err
	}
	key := spoolPrefix + strconv.FormatInt(time.Now().UTC().Unix(), 10) + "-" + string(id)
	hash := sha256.New()
	counter := &countingReader{reader: io.TeeReader(io.LimitReader(input.Body, input.Size+1), hash)}
	if err := s.disk.PutNew(ctx, key, counter, mediaType); err != nil {
		return Attachment{}, fmt.Errorf("write transient mail attachment: %w", err)
	}
	if counter.count != input.Size {
		_ = s.disk.Delete(context.WithoutCancel(ctx), key)
		return Attachment{}, fmt.Errorf("%w: transient attachment size does not match body", ErrInvalid)
	}
	return newStoredAttachment(AttachmentTransient, nil, key, filename, mediaType, input.Size, hex.EncodeToString(hash.Sum(nil))), nil
}

func (s *AttachmentSpool) Open(ctx context.Context, attachment Attachment) (io.ReadCloser, error) {
	if s == nil || s.disk == nil || attachment.Source != AttachmentTransient || !validSpoolKey(attachment.spoolKey) {
		return nil, errors.New("mail transient attachment reference is invalid")
	}
	return s.disk.Open(ctx, attachment.spoolKey)
}

func (s *AttachmentSpool) Delete(ctx context.Context, attachment Attachment) error {
	if attachment.Source != AttachmentTransient {
		return nil
	}
	if s == nil || s.disk == nil || !validSpoolKey(attachment.spoolKey) {
		return errors.New("mail transient attachment reference is invalid")
	}
	return s.disk.Delete(ctx, attachment.spoolKey)
}

func (s *AttachmentSpool) Cleanup(ctx context.Context, olderThan time.Time, limit int, active func(context.Context, []string) (map[string]struct{}, error)) (int, error) {
	if s == nil || s.disk == nil || limit < 1 || active == nil {
		return 0, errors.New("mail spool cleanup request is invalid")
	}
	walker := s.disk.(filesystem.PrefixWalker)
	candidates := make([]string, 0, limit)
	err := walker.WalkPrefix(ctx, spoolPrefix, func(key string) error {
		if len(candidates) >= limit {
			return errSpoolCleanupLimit
		}
		if !validSpoolKey(key) {
			return nil
		}
		createdAt, ok := spoolCreatedAt(key)
		if ok && createdAt.Before(olderThan) {
			candidates = append(candidates, key)
		}
		return nil
	})
	if errors.Is(err, errSpoolCleanupLimit) {
		err = nil
	}
	if err != nil || len(candidates) == 0 {
		return 0, err
	}
	protected, err := active(ctx, candidates)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, key := range candidates {
		if _, exists := protected[key]; exists {
			continue
		}
		if err := s.disk.Delete(ctx, key); err != nil && !errors.Is(err, filesystem.ErrNotFound) {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.count += int64(n)
	return n, err
}

func validSpoolKey(key string) bool {
	_, ok := spoolCreatedAt(key)
	return ok
}

func spoolCreatedAt(key string) (time.Time, bool) {
	if !strings.HasPrefix(key, spoolPrefix) || filepath.Clean(key) != key || strings.Contains(strings.TrimPrefix(key, spoolPrefix), "/") {
		return time.Time{}, false
	}
	raw := strings.TrimPrefix(key, spoolPrefix)
	seconds, opaqueID, ok := strings.Cut(raw, "-")
	if !ok {
		return time.Time{}, false
	}
	value, err := strconv.ParseInt(seconds, 10, 64)
	if err != nil || value <= 0 || messageid.ID(opaqueID).Validate() != nil {
		return time.Time{}, false
	}
	return time.Unix(value, 0).UTC(), true
}
