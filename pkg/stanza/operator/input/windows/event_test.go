// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows
// +build windows

package windows

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/text/encoding/unicode"

	"github.com/stretchr/testify/require"
)

func TestEventCloseWhenAlreadyClosed(t *testing.T) {
	event := NewEvent(0)
	err := event.Close()
	require.NoError(t, err)
}

func TestEventCloseSyscallFailure(t *testing.T) {
	event := NewEvent(5)
	closeProc = SimpleMockProc(0, 0, ErrorNotSupported)
	err := event.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to close event handle")
}

func TestEventCloseSuccess(t *testing.T) {
	event := NewEvent(5)
	closeProc = SimpleMockProc(1, 0, ErrorSuccess)
	err := event.Close()
	require.NoError(t, err)
	require.Equal(t, uintptr(0), event.handle)
}

func TestEventRenderSimple(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "xmlSample.xml"))
	dataUTF16, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(data)
	require.NoError(t, err)
	event := NewEvent(5)
	buf := NewBuffer()
	renderProc = renderMockProc(dataUTF16)
	_, err = event.RenderSimple(buf)
	require.NoError(t, err)
}

// same test as above, but with an initial buffer smaller than the event data
func TestEventRenderSimpleSmallBuffer(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "xmlSample.xml"))
	dataUTF16, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(data)
	require.NoError(t, err)
	event := NewEvent(5)
	buf := Buffer{buffer: make([]byte, len(dataUTF16)/2)}
	renderProc = renderMockProc(dataUTF16)
	_, err = event.RenderSimple(buf)
	require.NoError(t, err)
}

func TestEventRenderRaw(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "xmlSample.xml"))
	dataUTF16, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(data)
	require.NoError(t, err)
	event := NewEvent(5)
	buf := NewBuffer()
	renderProc = renderMockProc(dataUTF16)
	_, err = event.RenderRaw(buf)
	require.NoError(t, err)
}

// same test as above, but with an initial buffer smaller than the event data
func TestEventRenderRawSmallBuffer(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "xmlSample.xml"))
	dataUTF16, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(data)
	require.NoError(t, err)
	event := NewEvent(5)
	buf := Buffer{buffer: make([]byte, len(dataUTF16)/2)}
	renderProc = renderMockProc(dataUTF16)
	_, err = event.RenderRaw(buf)
	require.NoError(t, err)
}

func renderMockProc(data []byte) MockProc {
	return MockProc{
		call: func(args ...uintptr) (uintptr, uintptr, error) {
			bufferSize, buffer, bufferUsed := (uint32)(args[3]), unsafe.Pointer(args[4]), unsafe.Pointer(args[5])
			bufferSlice := unsafe.Slice((*byte)(buffer), bufferSize)
			*(*uint32)(bufferUsed) = uint32(len(data))
			if int(bufferSize) < len(data) {
				return 0, 0, ErrorInsufficientBuffer
			}
			copy(bufferSlice, data)
			return 0, 0, nil
		},
	}
}
