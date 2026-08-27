package mail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/vernal96/go-cms/kernel/job"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

type AttachmentOpener interface {
	Open(context.Context, security.Actor, file.ID) (file.OpenedFile, error)
}

type Worker struct {
	siteID     site.ID
	repository Repository
	files      AttachmentOpener
	transports *TransportRegistry
}

func NewWorker(siteID site.ID, repository Repository, files AttachmentOpener, transports *TransportRegistry) (*Worker, error) {
	if siteID <= 0 || repository == nil || files == nil || transports == nil {
		return nil, errors.New("mail worker dependencies are nil")
	}
	return &Worker{siteID: siteID, repository: repository, files: files, transports: transports}, nil
}

func (w *Worker) Handle(ctx context.Context, item job.Envelope) error {
	if item.Name != SendJobName || item.SchemaVersion != 1 {
		return errors.New("mail send job envelope is invalid")
	}
	if item.ScopeID != fmt.Sprint(w.siteID) {
		return errors.New("mail send job site scope is invalid")
	}
	var payload struct {
		MessageID MessageID `json:"message_id"`
	}
	if err := json.Unmarshal(item.Payload, &payload); err != nil || payload.MessageID <= 0 {
		return errors.New("mail send job payload is invalid")
	}
	message, attempt, claimed, err := w.repository.ClaimMessage(ctx, w.siteID, payload.MessageID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil || !claimed {
		return err
	}
	transport, exists := w.transports.Transport(message.Transport)
	if !exists {
		err = fmt.Errorf("%w: %s", ErrTransportNotFound, message.Transport)
		_ = w.repository.FinishAttempt(ctx, message.ID, attempt.AttemptNumber, DeliveryResult{}, err)
		return err
	}
	attachments := make([]DeliveryAttachment, 0, len(message.Attachments))
	closers := make([]io.Closer, 0, len(message.Attachments))
	defer func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}()
	for _, attachment := range message.Attachments {
		opened, openErr := w.files.Open(ctx, security.System(), attachment.FileID)
		if openErr != nil {
			err = fmt.Errorf("open mail attachment %d: %w", attachment.FileID, openErr)
			_ = w.repository.FinishAttempt(ctx, message.ID, attempt.AttemptNumber, DeliveryResult{Driver: transport.Driver()}, err)
			return err
		}
		closers = append(closers, opened.Body)
		attachments = append(attachments, DeliveryAttachment{Attachment: attachment, Body: opened.Body})
	}
	result, sendErr := transport.Send(ctx, Delivery{Message: message, Attachments: attachments})
	if result.Driver == "" {
		result.Driver = transport.Driver()
	}
	if finishErr := w.repository.FinishAttempt(ctx, message.ID, attempt.AttemptNumber, result, sendErr); finishErr != nil {
		return errors.Join(sendErr, finishErr)
	}
	return sendErr
}
