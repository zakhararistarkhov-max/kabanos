package calendar

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
)

// yandexEndpoint is the Yandex.Calendar CalDAV entry point.
const yandexEndpoint = "https://caldav.yandex.ru"

// ErrInvalidCreds is returned when the server rejects the login/app password.
var ErrInvalidCreds = errors.New("invalid calendar credentials")

func davClient(login, password string) (*caldav.Client, error) {
	httpClient := webdav.HTTPClientWithBasicAuth(&http.Client{Timeout: 30 * time.Second}, login, password)
	return caldav.NewClient(httpClient, yandexEndpoint)
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	if strings.Contains(s, "401") || strings.Contains(s, "403") || strings.Contains(s, "Unauthorized") {
		return ErrInvalidCreds
	}
	return err
}

// discoverCalendars authenticates and lists the account's calendars.
func discoverCalendars(ctx context.Context, login, password string) ([]RemoteCalendar, error) {
	c, err := davClient(login, password)
	if err != nil {
		return nil, err
	}
	principal, err := c.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, classify(err)
	}
	home, err := c.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, classify(err)
	}
	cals, err := c.FindCalendars(ctx, home)
	if err != nil {
		return nil, classify(err)
	}
	out := make([]RemoteCalendar, 0, len(cals))
	for _, cal := range cals {
		if !supportsEvents(cal.SupportedComponentSet) {
			continue
		}
		out = append(out, RemoteCalendar{URL: cal.Path, Name: cal.Name})
	}
	return out, nil
}

func supportsEvents(set []string) bool {
	if len(set) == 0 {
		return true
	}
	for _, s := range set {
		if strings.EqualFold(s, "VEVENT") {
			return true
		}
	}
	return false
}

// pullEvents fetches VEVENTs in [from, to) from the given calendar collection.
func pullEvents(ctx context.Context, login, password, calURL string, from, to time.Time) ([]remoteEvent, error) {
	c, err := davClient(login, password)
	if err != nil {
		return nil, err
	}
	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name:  "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{{Name: "VEVENT", AllProps: true}},
		},
		CompFilter: caldav.CompFilter{
			Name:  "VCALENDAR",
			Comps: []caldav.CompFilter{{Name: "VEVENT", Start: from, End: to}},
		},
	}
	objs, err := c.QueryCalendar(ctx, calURL, query)
	if err != nil {
		return nil, classify(err)
	}
	out := []remoteEvent{}
	for _, o := range objs {
		if o.Data == nil {
			continue
		}
		for _, ev := range o.Data.Events() {
			uid, _ := ev.Props.Text(ical.PropUID)
			start, err := ev.DateTimeStart(time.UTC)
			if uid == "" || err != nil || start.IsZero() {
				continue
			}
			summary, _ := ev.Props.Text(ical.PropSummary)
			if summary == "" {
				summary = "(без названия)"
			}
			notes, _ := ev.Props.Text(ical.PropDescription)
			allDay := false
			if sp := ev.Props.Get(ical.PropDateTimeStart); sp != nil && sp.ValueType() == ical.ValueDate {
				allDay = true
			}
			var end *time.Time
			if ev.Props.Get(ical.PropDateTimeEnd) != nil {
				if e, err := ev.DateTimeEnd(time.UTC); err == nil && !e.IsZero() {
					end = &e
				}
			}
			out = append(out, remoteEvent{
				UID: uid, Href: o.Path, ETag: o.ETag, Summary: summary, Notes: notes,
				Start: start.UTC(), End: end, AllDay: allDay,
			})
		}
	}
	return out, nil
}

// pushEvent creates or updates a VEVENT for a local item. Returns the UID, the
// resource href and the new ETag.
func pushEvent(ctx context.Context, login, password, calURL string, it pushItem) (uid, href, etag string, err error) {
	c, cerr := davClient(login, password)
	if cerr != nil {
		return "", "", "", cerr
	}

	uid = it.ID.String() + "@kabanos"
	if it.ExternalUID != nil && *it.ExternalUID != "" {
		uid = *it.ExternalUID
	}
	href = joinHref(calURL, uid+".ics")
	if it.ExternalHref != nil && *it.ExternalHref != "" {
		href = *it.ExternalHref
	}

	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, uid)
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	event.Props.SetText(ical.PropSummary, it.Title)
	if it.Notes != "" {
		event.Props.SetText(ical.PropDescription, it.Notes)
	}
	if it.AllDay {
		event.Props.SetDate(ical.PropDateTimeStart, it.ScheduledAt)
		end := it.ScheduledAt.Add(24 * time.Hour)
		if it.EndAt != nil {
			end = *it.EndAt
		}
		event.Props.SetDate(ical.PropDateTimeEnd, end)
	} else {
		event.Props.SetDateTime(ical.PropDateTimeStart, it.ScheduledAt.UTC())
		end := it.ScheduledAt.Add(time.Hour)
		if it.EndAt != nil {
			end = *it.EndAt
		}
		event.Props.SetDateTime(ical.PropDateTimeEnd, end.UTC())
	}

	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//kabanos//GTD//EN")
	cal.Children = append(cal.Children, event.Component)

	obj, perr := c.PutCalendarObject(ctx, href, cal)
	if perr != nil {
		return "", "", "", classify(perr)
	}
	if obj != nil {
		if obj.Path != "" {
			href = obj.Path
		}
		etag = obj.ETag
	}
	return uid, href, etag, nil
}

// removeEvent deletes a remote resource by href.
func removeEvent(ctx context.Context, login, password, href string) error {
	c, err := davClient(login, password)
	if err != nil {
		return err
	}
	if err := c.RemoveAll(ctx, href); err != nil {
		// A 404 means it's already gone — treat as success.
		if strings.Contains(err.Error(), "404") {
			return nil
		}
		return classify(err)
	}
	return nil
}

func joinHref(base, name string) string {
	if strings.HasSuffix(base, "/") {
		return base + name
	}
	return base + "/" + name
}
