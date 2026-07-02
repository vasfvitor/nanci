package app

import (
	"context"
	"fmt"
	"time"
)

type EventView struct {
	ID                     string
	Type                   string
	EventAt                *time.Time
	ReplacementChaveAcesso string
	Description            string
	RawXMLPath             string
}

// ListEventsForDocument returns the events associated with a document.
func (s *DocumentService) ListEventsForDocument(ctx context.Context, documentID string) ([]EventView, error) {
	events, err := s.DocumentRepo.ListEventsByDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar eventos do documento: %w", err)
	}

	views := make([]EventView, len(events))
	for i, e := range events {
		views[i] = EventView{
			ID:                     e.ID,
			Type:                   string(e.Type),
			EventAt:                e.EventAt,
			ReplacementChaveAcesso: e.ReplacementChaveAcesso,
			Description:            e.Description,
			RawXMLPath:             e.RawXMLPath,
		}
	}
	return views, nil
}
