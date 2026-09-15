package resource

import (
	"context"
	"fmt"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/security"
)

func (s *Service) validateMediaFields(ctx context.Context, actor security.Actor, values []field.StoredValue) error {
	for _, value := range values {
		if value.ReferenceTarget != field.ReferenceMedia {
			continue
		}
		id, ok := value.Value.(int64)
		if !ok || id <= 0 {
			return fmt.Errorf("%w: invalid Media field %q", ErrInvalidReference, value.Key)
		}
		resolved, err := s.media.Resolve(ctx, actor, media.ID(id))
		if err != nil {
			return fmt.Errorf("resolve Media field %q: %w", value.Key, err)
		}
		if err := ValidateImageMediaFile(ctx, resolved.File, media.Usage{Kind: ImageMediaUsage}); err != nil {
			return err
		}
	}
	return nil
}
