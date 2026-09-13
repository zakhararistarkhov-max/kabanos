package calendar

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
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

// davClient builds a go-webdav CalDAV client (used only for discovery, which
// does not read per-item ETags).
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

// --- raw CalDAV (Yandex returns non-RFC ETags that break go-webdav's strict
// unquoting, so pull/push/delete are done over plain HTTP with lenient ETag
// handling; iCalendar bodies are still parsed/built with go-ical). ---

const httpTimeout = 30 * time.Second

func httpDo(ctx context.Context, login, password, method, url, contentType, depth string, body []byte) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(login, password)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if depth != "" {
		req.Header.Set("Depth", depth)
	}
	client := &http.Client{Timeout: httpTimeout}
	return client.Do(req)
}

func fullURL(pathOrURL string) string {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		return pathOrURL
	}
	if !strings.HasPrefix(pathOrURL, "/") {
		pathOrURL = "/" + pathOrURL
	}
	return yandexEndpoint + pathOrURL
}

func statusErr(status int, body []byte) error {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return ErrInvalidCreds
	}
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	return fmt.Errorf("caldav: unexpected status %d: %s", status, snippet)
}

func cleanETag(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "W/")
	return strings.Trim(s, `"`)
}

// multistatus mirrors the CalDAV REPORT response. Fields match by local name so
// the DAV:/CalDAV namespaces don't matter, and ETag stays a plain string.
type msMultistatus struct {
	XMLName   xml.Name     `xml:"multistatus"`
	Responses []msResponse `xml:"response"`
}

type msResponse struct {
	Href     string       `xml:"href"`
	Propstat []msPropstat `xml:"propstat"`
}

type msPropstat struct {
	Status string `xml:"status"`
	Prop   struct {
		ETag         string `xml:"getetag"`
		CalendarData string `xml:"calendar-data"`
	} `xml:"prop"`
}

// pullEvents fetches VEVENTs in [from, to) from the given calendar collection.
func pullEvents(ctx context.Context, login, password, calURL string, from, to time.Time) ([]remoteEvent, error) {
	const layout = "20060102T150405Z"
	body := []byte(`<?xml version="1.0" encoding="utf-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop><d:getetag/><c:calendar-data/></d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT">
        <c:time-range start="` + from.UTC().Format(layout) + `" end="` + to.UTC().Format(layout) + `"/>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`)

	resp, err := httpDo(ctx, login, password, "REPORT", fullURL(calURL), "application/xml; charset=utf-8", "1", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		return nil, statusErr(resp.StatusCode, data)
	}
	return parseCalendarReport(data)
}

// parseCalendarReport decodes a CalDAV multistatus REPORT body into events. It
// tolerates non-RFC ETags (Yandex) and skips unparseable objects instead of
// failing the whole sync.
func parseCalendarReport(data []byte) ([]remoteEvent, error) {
	var ms msMultistatus
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("caldav: parse multistatus: %w", err)
	}

	out := []remoteEvent{}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if !strings.Contains(ps.Status, "200") || strings.TrimSpace(ps.Prop.CalendarData) == "" {
				continue
			}
			cal, err := ical.NewDecoder(strings.NewReader(ps.Prop.CalendarData)).Decode()
			if err != nil {
				continue // skip an unparseable object rather than failing the whole sync
			}
			for _, ev := range cal.Events() {
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
					UID: uid, Href: r.Href, ETag: cleanETag(ps.Prop.ETag), Summary: summary,
					Notes: notes, Start: start.UTC(), End: end, AllDay: allDay,
				})
			}
		}
	}
	return out, nil
}

// pushEvent creates or updates a VEVENT for a local item. Returns the UID, the
// resource href and the new ETag.
func pushEvent(ctx context.Context, login, password, calURL string, it pushItem) (uid, href, etag string, err error) {
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

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(cal); err != nil {
		return "", "", "", err
	}

	resp, err := httpDo(ctx, login, password, http.MethodPut, fullURL(href), "text/calendar; charset=utf-8", "", buf.Bytes())
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", "", "", statusErr(resp.StatusCode, body)
	}
	etag = cleanETag(resp.Header.Get("ETag"))
	return uid, href, etag, nil
}

// removeEvent deletes a remote resource by href.
func removeEvent(ctx context.Context, login, password, href string) error {
	resp, err := httpDo(ctx, login, password, http.MethodDelete, fullURL(href), "", "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
		return nil // 404 = already gone
	default:
		return statusErr(resp.StatusCode, body)
	}
}

func joinHref(base, name string) string {
	if strings.HasSuffix(base, "/") {
		return base + name
	}
	return base + "/" + name
}
