package mocks

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/maestro/pkg/api"
	"github.com/openshift-online/maestro/pkg/dao"
)

var _ dao.EventDao = &eventDaoMock{}

type eventDaoMock struct {
	events api.EventList
}

func NewEventDao() *eventDaoMock {
	return &eventDaoMock{}
}

func (d *eventDaoMock) Get(ctx context.Context, id string) (*api.Event, error) {
	for _, event := range d.events {
		if event.ID == id {
			return event, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *eventDaoMock) Create(ctx context.Context, event *api.Event) (*api.Event, error) {
	d.events = append(d.events, event)
	return event, nil
}

func (d *eventDaoMock) Replace(ctx context.Context, event *api.Event) (*api.Event, error) {
	for i, e := range d.events {
		if e.ID == event.ID {
			d.events[i] = event
			return event, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *eventDaoMock) Delete(ctx context.Context, id string) error {
	newEvents := api.EventList{}
	for _, e := range d.events {
		if e.ID == id {
			// deleting this one
			// do not include in the new list
		} else {
			newEvents = append(newEvents, e)
		}
	}
	d.events = newEvents
	return nil
}

func (d *eventDaoMock) FindByIDs(ctx context.Context, ids []string) (api.EventList, error) {
	filteredEvents := api.EventList{}
	for _, id := range ids {
		for _, e := range d.events {
			if e.ID == id {
				filteredEvents = append(filteredEvents, e)
			}
		}
	}
	return filteredEvents, nil
}

func (d *eventDaoMock) All(ctx context.Context) (api.EventList, error) {
	return d.events, nil
}

func (d *eventDaoMock) DeleteAllReconciledEvents(ctx context.Context) error {
	newEvents := api.EventList{}
	for _, e := range d.events {
		if e.ReconciledDate != nil {
			// deleting this one
			// do not include in the new list
		} else {
			newEvents = append(newEvents, e)
		}
	}
	d.events = newEvents
	return nil
}

func (d *eventDaoMock) FindAllUnreconciledEvents(ctx context.Context) (api.EventList, error) {
	filteredEvents := api.EventList{}
	for _, e := range d.events {
		if e.ReconciledDate != nil {
			continue
		}
		filteredEvents = append(filteredEvents, e)
	}

	return filteredEvents, nil
}
