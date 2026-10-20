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
		return failureError{subject: "application bridge frame exceeds limit"}
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], length)
	if err := writeAll(connection, header[:]); err != nil {
		return failureError{
			subject: "write application bridge frame header",
			cause:   err,
		}
	}
	if err := writeAll(connection, payload); err != nil {
		return failureError{
			subject: "write application bridge frame payload",
			cause:   err,
		}
	}
	return nil
}

func readBridgeFrame(connection net.Conn, limit int) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(connection, header[:]); err != nil {
		return nil, failureError{
			subject: "read application bridge frame header",
			cause:   err,
		}
	}
	length := binary.BigEndian.Uint32(header[:])
	limit32, ok := bridgeUint32Length(limit, math.MaxUint32)
	if !ok || length > limit32 {
		return nil, failureError{subject: "application bridge frame exceeds limit"}
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(connection, payload); err != nil {
		return nil, failureError{
			subject: "read application bridge frame payload",
			cause:   err,
		}
	}
	return payload, nil
}

func encodeBridgeProcessRequest(request bridgeProcessRequest) ([]byte, error) {
	if len(request.args) > bridgeRequestElementLimit || len(request.environ) > bridgeRequestElementLimit || len(request.stdin) > bridgeStdinLimit {
		return nil, failureError{subject: "application bridge request exceeds limit"}
	}
	writer := boundedWriter{limit: bridgeRequestFrameLimit}
	if err := writeAll(&writer, []byte{1}); err != nil {
		return nil, failureError{
			subject: "encode application bridge request version",
			cause:   err,
		}
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
		return nil, failureError{
			subject: "application bridge request exceeds frame limit",
			cause:   err,
		}
	}
	return encoded, nil
}

func decodeBridgeProcessRequest(payload []byte) (bridgeProcessRequest, error) {
	reader := bytes.NewReader(payload)
	version, err := reader.ReadByte()
	if err != nil || version != 1 {
		return bridgeProcessRequest{}, failureError{
			subject: "invalid application bridge request version",
			cause:   err,
		}
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
		return bridgeProcessRequest{}, failureError{subject: "trailing application bridge request data"}
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
		return failureError{subject: bridgeRequestElementLimitSubject}
	}
	count, ok := bridgeUint32Length(len(values), bridgeRequestElementLimit)
	if !ok {
		return failureError{subject: bridgeRequestElementLimitSubject}
	}
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], count)
	if err := writeAll(writer, encoded[:]); err != nil {
		return failureError{
			subject: "write application bridge collection count",
			cause:   err,
		}
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
		return failureError{subject: "application bridge value exceeds limit"}
	}
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], length)
	if err := writeAll(writer, encoded[:]); err != nil {
		return failureError{
			subject: "write application bridge value length",
			cause:   err,
		}
	}
	if err := writeAll(writer, value); err != nil {
		return failureError{
			subject: "write application bridge value",
			cause:   err,
		}
	}
	return nil
}

func readBridgeStrings(reader *bytes.Reader) ([]string, error) {
	count, err := readBridgeUint32(reader)
	if err != nil {
		return nil, err
	}
	if count > bridgeRequestElementLimit {
		return nil, failureError{subject: bridgeRequestElementLimitSubject}
	}
	// #nosec G115 -- the element-limit check above proves count fits in int on supported targets.
	countInt := int(count)
	const encodedLengthBytes = 4
	if countInt > reader.Len()/encodedLengthBytes {
		return nil, failureError{subject: "invalid application bridge collection length"}
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
		return nil, failureError{subject: "invalid application bridge value length"}
	}
	value := make([]byte, int(length))
	if _, err := io.ReadFull(reader, value); err != nil {
		return nil, failureError{
			subject: "read application bridge value",
			cause:   err,
		}
	}
	return value, nil
}

func readBridgeUint32(reader *bytes.Reader) (uint32, error) {
	var encoded [4]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		return 0, failureError{
			subject: "read application bridge integer",
			cause:   err,
		}
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
			return failureError{
				subject: "write complete byte sequence",
				cause:   err,
			}
		}
		if written <= 0 || written > len(data) {
			return failureError{
				subject: "write complete byte sequence",
				cause:   io.ErrShortWrite,
			}
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
		return 0, failureError{subject: "bounded writer limit exceeded"}
	}
	written, _ := writer.buffer.Write(data)
	return written, nil
}

func (writer *boundedWriter) bytes() ([]byte, error) {
	if writer.exceeded {
		return nil, failureError{subject: "shell completion response exceeds limit"}
	}
	return writer.buffer.Bytes(), nil
}
