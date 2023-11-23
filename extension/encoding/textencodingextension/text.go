// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package textencodingextension // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/textencodingextension"

import (
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/textutils"
)

type textLogCodec struct {
	enc *textutils.Encoding
}

func (r *textLogCodec) UnmarshalLogs(buf []byte) (plog.Logs, error) {
	p := plog.NewLogs()
	decoded, err := r.enc.Decode(buf)
	if err != nil {
		return p, err
	}

	l := p.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	l.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	l.Body().SetStr(string(decoded))
	return p, nil
}

func (r *textLogCodec) MarshalLogs(ld plog.Logs) ([]byte, error) {
	encoder := r.enc.Encoding.NewEncoder()
	delimiter := '\n'
	builder := strings.Builder{}
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLogs := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLogs.ScopeLogs().Len(); j++ {
			scopeLogs := resourceLogs.ScopeLogs().At(j)
			for k := 0; k < scopeLogs.LogRecords().Len(); k++ {
				logRecord := scopeLogs.LogRecords().At(k)
				if builder.Len() > 0 {
					builder.WriteRune(delimiter)
				}
				builder.WriteString(logRecord.Body().AsString())
			}
		}
	}
	output := make([]byte, builder.Len())
	_, _, err := encoder.Transform(output, []byte(builder.String()), false)
	return output, err
}
