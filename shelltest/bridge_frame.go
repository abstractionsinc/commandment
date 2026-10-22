package shelltest

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"net"
)

const (
	bridgeRequestFrameLimit          = 16 << 20
	bridgeResponseFrameLimit         = 8 << 20
	bridgeAckFrameLimit              = 1
	bridgeStdinLimit                 = 8 << 20
	bridgeCompletionResponseLimit    = 8 << 20
	bridgeRequestElementLimit        = 65_536
	bridgeStringLimit                = 1 << 20
	bridgeRequestElementLimitSubject = "application bridge request element count exceeds limit"
)

type bridgeProcessRequest struct {
	args             []string
	environ          []string
	workingDirectory string
	stdin            []byte
}

func writeBridgeFrame(connection net.Conn, payload []byte, limit int) error {
	length, ok := bridgeUint32Length(len(payload), limit)
	if !ok {
		failure := failureError{
			subject: "application bridge frame exceeds limit",
		}
		return failure
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], length)
	if err := writeAll(connection, header[:]); err != nil {
		failure := failureError{
			subject: "write application bridge frame header",
			cause:   err,
		}
		return failure
	}
	if err := writeAll(connection, payload); err != nil {
		failure := failureError{
			subject: "write application bridge frame payload",
			cause:   err,
		}
		return failure
	}
	return nil
}

func readBridgeFrame(connection net.Conn, limit int) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(connection, header[:]); err != nil {
		failure := failureError{
			subject: "read application bridge frame header",
			cause:   err,
		}
		return nil, failure
	}
	length := binary.BigEndian.Uint32(header[:])
	limit32, ok := bridgeUint32Length(limit, math.MaxUint32)
	if !ok || length > limit32 {
		failure := failureError{
			subject: "application bridge frame exceeds limit",
		}
		return nil, failure
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(connection, payload); err != nil {
		failure := failureError{
			subject: "read application bridge frame payload",
			cause:   err,
		}
		return nil, failure
	}
	return payload, nil
}

func encodeBridgeProcessRequest(request bridgeProcessRequest) ([]byte, error) {
	if len(request.args) > bridgeRequestElementLimit || len(request.environ) > bridgeRequestElementLimit || len(request.stdin) > bridgeStdinLimit {
		failure := failureError{
			subject: "application bridge request exceeds limit",
		}
		return nil, failure
	}
	writer := boundedWriter{
		limit: bridgeRequestFrameLimit,
	}
	if err := writeAll(&writer, []byte{1}); err != nil {
		failure := failureError{
			subject: "encode application bridge request version",
			cause:   err,
		}
		return nil, failure
	}
	if err := writeBridgeStrings(&writer, request.args); err != nil {
		return nil, err
	}
	if err := writeBridgeStrings(&writer, request.environ); err != nil {
		return nil, err
	}
	if err := writeBridgeString(&writer, request.workingDirectory); err != nil {
		return nil, err
	}
	if err := writeBridgeBytes(&writer, request.stdin, bridgeStdinLimit); err != nil {
		return nil, err
	}
	encoded, err := writer.bytes()
	if err != nil {
		failure := failureError{
			subject: "application bridge request exceeds frame limit",
			cause:   err,
		}
		return nil, failure
	}
	return encoded, nil
}

func decodeBridgeProcessRequest(payload []byte) (bridgeProcessRequest, error) {
	reader := bytes.NewReader(payload)
	version, err := reader.ReadByte()
	if err != nil || version != 1 {
		failure := failureError{
			subject: "invalid application bridge request version",
			cause:   err,
		}
		return bridgeProcessRequest{}, failure
	}
	args, err := readBridgeStrings(reader)
	if err != nil {
		return bridgeProcessRequest{}, err
	}
	environ, err := readBridgeStrings(reader)
	if err != nil {
		return bridgeProcessRequest{}, err
	}
	workingDirectory, err := readBridgeString(reader)
	if err != nil {
		return bridgeProcessRequest{}, err
	}
	stdin, err := readBridgeBytes(reader, bridgeStdinLimit)
	if err != nil {
		return bridgeProcessRequest{}, err
	}
	if reader.Len() != 0 {
		failure := failureError{
			subject: "trailing application bridge request data",
		}
		return bridgeProcessRequest{}, failure
	}
	request := bridgeProcessRequest{
		args:             args,
		environ:          environ,
		workingDirectory: workingDirectory,
		stdin:            stdin,
	}
	return request, nil
}

func writeBridgeStrings(writer io.Writer, values []string) error {
	if len(values) > bridgeRequestElementLimit {
		failure := failureError{
			subject: bridgeRequestElementLimitSubject,
		}
		return failure
	}
	count, ok := bridgeUint32Length(len(values), bridgeRequestElementLimit)
	if !ok {
		failure := failureError{
			subject: bridgeRequestElementLimitSubject,
		}
		return failure
	}
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], count)
	if err := writeAll(writer, encoded[:]); err != nil {
		failure := failureError{
			subject: "write application bridge collection count",
			cause:   err,
		}
		return failure
	}
	for _, value := range values {
		if err := writeBridgeString(writer, value); err != nil {
			return err
		}
	}
	return nil
}

func writeBridgeString(writer io.Writer, value string) error {
	return writeBridgeBytes(writer, []byte(value), bridgeStringLimit)
}

func writeBridgeBytes(writer io.Writer, value []byte, limit int) error {
	length, ok := bridgeUint32Length(len(value), limit)
	if !ok {
		failure := failureError{
			subject: "application bridge value exceeds limit",
		}
		return failure
	}
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], length)
	if err := writeAll(writer, encoded[:]); err != nil {
		failure := failureError{
			subject: "write application bridge value length",
			cause:   err,
		}
		return failure
	}
	if err := writeAll(writer, value); err != nil {
		failure := failureError{
			subject: "write application bridge value",
			cause:   err,
		}
		return failure
	}
	return nil
}

func readBridgeStrings(reader *bytes.Reader) ([]string, error) {
	count, err := readBridgeUint32(reader)
	if err != nil {
		return nil, err
	}
	if count > bridgeRequestElementLimit {
		failure := failureError{
			subject: bridgeRequestElementLimitSubject,
		}
		return nil, failure
	}
	// #nosec G115 -- the element-limit check above proves count fits in int on supported targets.
	countInt := int(count)
	const encodedLengthBytes = 4
	if countInt > reader.Len()/encodedLengthBytes {
		failure := failureError{
			subject: "invalid application bridge collection length",
		}
		return nil, failure
	}
	values := make([]string, countInt)
	for index := range values {
		values[index], err = readBridgeString(reader)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func readBridgeString(reader *bytes.Reader) (string, error) {
	value, err := readBridgeBytes(reader, bridgeStringLimit)
	return string(value), err
}

func readBridgeBytes(reader *bytes.Reader, limit int) ([]byte, error) {
	length, err := readBridgeUint32(reader)
	if err != nil {
		return nil, err
	}
	limit32, limitOK := bridgeUint32Length(limit, math.MaxUint32)
	remaining32, remainingOK := bridgeUint32Length(reader.Len(), math.MaxUint32)
	if !limitOK || !remainingOK || length > limit32 || length > remaining32 {
		failure := failureError{
			subject: "invalid application bridge value length",
		}
		return nil, failure
	}
	value := make([]byte, int(length))
	if _, err := io.ReadFull(reader, value); err != nil {
		failure := failureError{
			subject: "read application bridge value",
			cause:   err,
		}
		return nil, failure
	}
	return value, nil
}

func readBridgeUint32(reader *bytes.Reader) (uint32, error) {
	var encoded [4]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		failure := failureError{
			subject: "read application bridge integer",
			cause:   err,
		}
		return 0, failure
	}
	return binary.BigEndian.Uint32(encoded[:]), nil
}

func bridgeUint32Length(length, limit int) (uint32, bool) {
	if length < 0 || limit < 0 || length > limit || uint64(length) > math.MaxUint32 {
		return 0, false
	}
	// #nosec G115 -- the range check above proves the conversion is bounded.
	return uint32(length), true
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) != 0 {
		written, err := writer.Write(data)
		if err != nil {
			failure := failureError{
				subject: "write complete byte sequence",
				cause:   err,
			}
			return failure
		}
		if written <= 0 || written > len(data) {
			failure := failureError{
				subject: "write complete byte sequence",
				cause:   io.ErrShortWrite,
			}
			return failure
		}
		data = data[written:]
	}
	return nil
}

type boundedWriter struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (writer *boundedWriter) Write(data []byte) (int, error) {
	if writer.exceeded || len(data) > writer.limit-writer.buffer.Len() {
		writer.exceeded = true
		failure := failureError{
			subject: "bounded writer limit exceeded",
		}
		return 0, failure
	}
	written, _ := writer.buffer.Write(data)
	return written, nil
}

func (writer *boundedWriter) bytes() ([]byte, error) {
	if writer.exceeded {
		failure := failureError{
			subject: "shell completion response exceeds limit",
		}
		return nil, failure
	}
	return writer.buffer.Bytes(), nil
}
