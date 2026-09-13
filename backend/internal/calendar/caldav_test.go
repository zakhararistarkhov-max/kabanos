package calendar

import "testing"

// A Yandex-style multistatus: getetag is NOT wrapped in double quotes, which is
// exactly what broke go-webdav's strict parser. Our lenient parser must cope.
const sampleReport = `<?xml version="1.0" encoding="UTF-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>/calendars/user@yandex.ru/events-1/abc%40kabanos.ics</d:href>
    <d:propstat>
      <d:prop>
        <d:getetag>1694635200-1</d:getetag>
        <cal:calendar-data>BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Yandex//Calendar//EN
BEGIN:VEVENT
UID:abc@kabanos
SUMMARY:Тренировка Лужники
DTSTART:20260914T204300Z
DTEND:20260914T214300Z
END:VEVENT
END:VCALENDAR
</cal:calendar-data>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`

func TestParseCalendarReportLenientETag(t *testing.T) {
	events, err := parseCalendarReport([]byte(sampleReport))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	e := events[0]
	if e.UID != "abc@kabanos" {
		t.Errorf("UID = %q", e.UID)
	}
	if e.Summary != "Тренировка Лужники" {
		t.Errorf("Summary = %q", e.Summary)
	}
	if e.ETag != "1694635200-1" {
		t.Errorf("ETag = %q, want unquoted value", e.ETag)
	}
	if e.Start.IsZero() || e.End == nil {
		t.Errorf("start/end not parsed: start=%v end=%v", e.Start, e.End)
	}
}

func TestCleanETag(t *testing.T) {
	cases := map[string]string{
		`"abc"`:    "abc",
		`abc`:      "abc",
		`W/"abc"`:  "abc",
		` "abc" `:  "abc",
		`12345-67`: "12345-67",
	}
	for in, want := range cases {
		if got := cleanETag(in); got != want {
			t.Errorf("cleanETag(%q) = %q, want %q", in, got, want)
		}
	}
}
