package format

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	baseTimestamp time.Time
)

func init() {
	baseTimestamp = time.Now()
}

// SimpleTextFormatter formats logs into text
type SimpleTextFormatter struct {
	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed.
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool

	// DisableLevelTruncation disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool

	// QuoteEmptyFields will wrap empty fields in quotes if true.
	QuoteEmptyFields bool

	// DisableIndent disables appending a tab in the prefix.
	DisableIndent bool
}

// Format renders a single log entry
func (f *SimpleTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	if !f.DisableIndent {
		b.WriteString("\n\t")
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = time.RFC3339
	}
	f.print(b, entry, keys, timestampFormat)
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *SimpleTextFormatter) print(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string) {
	levelText := strings.ToUpper(entry.Level.String())
	if !f.DisableLevelTruncation {
		levelText = levelText[0:4]
	}

	if f.DisableTimestamp {
		_, _ = fmt.Fprintf(b, "%s %-44s ", levelText, entry.Message)
	} else if !f.FullTimestamp {
		_, _ = fmt.Fprintf(b, "%s[%04d] %-44s ", levelText, int(entry.Time.Sub(baseTimestamp)/time.Second), entry.Message)
	} else {
		_, _ = fmt.Fprintf(b, "%s[%s] %-44s ", levelText, entry.Time.Format(timestampFormat), entry.Message)
	}
	for _, k := range keys {
		v := entry.Data[k]
		_, _ = fmt.Fprintf(b, " %s=", k)
		f.appendValue(b, v)
	}
}

func (f *SimpleTextFormatter) needsQuoting(text string) bool {
	if f.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

func (f *SimpleTextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
}

func (f *SimpleTextFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}
