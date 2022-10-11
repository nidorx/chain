package pubsub

import (
	"bytes"
	"compress/lzw"
	"io"
)

// compressPayload takes an opaque input buffer, compresses it and wraps it in a compress message that is encoded.
func compressPayload(payload []byte) ([]byte, error) {
	// ? metrics compression time, rate
	var buffer bytes.Buffer
	//writer := gzip.NewWriter(&buffer)
	writer := lzw.NewWriter(&buffer, lzw.LSB, 8)
	if _, err := writer.Write(payload); err != nil {
		return nil, err
	}

	// Ensure we flush everything out
	if err := writer.Close(); err != nil {
		return nil, err
	}

	// Create a compressed message
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(messageTypeCompress))
	buf.Write(buffer.Bytes())
	return buf.Bytes(), nil
}

// decompressPayload is used to unpack an encoded message and return its payload uncompressed
func decompressPayload(encoded []byte) ([]byte, error) {
	r := bytes.NewReader(encoded[1:])
	// Create a un compressor
	//reader, err := gzip.NewReader(r)
	//if err != nil {
	//	return nil, err
	//}
	reader := lzw.NewReader(r, lzw.LSB, 8)
	defer reader.Close()

	// Read all the data
	var b bytes.Buffer
	if _, err := io.Copy(&b, reader); err != nil {
		return nil, err
	}

	// Return the uncompressed bytes
	return b.Bytes(), nil
}
