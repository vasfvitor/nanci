package app

import (
	"context"
	"fmt"
)

type EventView struct {
	ID                     string
	Type                   string
	EventAt                string
	ReplacementChaveAcesso string
	Description            string
	RawXMLPath             string
}

func (a *App) ListEventsForDocument(ctx context.Context, documentID string) ([]EventView, error) {
	events, err := a.DocumentReader.ListEventsByDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar eventos do documento: %w", err)
	}

	views := make([]EventView, len(events))
	for i, e := range events {
		eventAtStr := ""
		if e.EventAtValid {
			eventAtStr = e.EventAt.Format("2006-01-02 15:04:05")
		}

		views[i] = EventView{
			ID:                     e.ID,
			Type:                   string(e.Type),
			EventAt:                eventAtStr,
			ReplacementChaveAcesso: e.ReplacementChaveAcesso,
			Description:            e.Description,
			RawXMLPath:             e.RawXMLPath,
		}
	}
	return views, nil
}
