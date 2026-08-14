package service

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
)

const (
	maxMP3ID3TagSize       = 16 << 20
	maxEmbeddedCoverSize   = 12 << 20
	maxEmbeddedCoverPixels = 24_000_000
)

var errMP3CoverNotFound = errors.New("MP3 embedded cover not found")

// extractEmbeddedMP3Cover reads the APIC frame in an ID3v2.3/v2.4 tag and
// normalizes the embedded image to JPEG. It deliberately accepts only a
// bounded tag and image, because the source MP3 is user-managed input.
func extractEmbeddedMP3Cover(inputPath, outputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 10)
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("read ID3 header: %w", err)
	}
	if string(header[:3]) != "ID3" || (header[3] != 3 && header[3] != 4) {
		return errMP3CoverNotFound
	}
	tagSize, ok := synchsafeInteger(header[6:10])
	if !ok || tagSize == 0 || tagSize > maxMP3ID3TagSize {
		return errMP3CoverNotFound
	}
	tag := make([]byte, tagSize)
	if _, err := io.ReadFull(file, tag); err != nil {
		return fmt.Errorf("read ID3 tag: %w", err)
	}

	position := 0
	if header[5]&0x40 != 0 {
		position, ok = skipID3ExtendedHeader(tag, header[3])
		if !ok {
			return errMP3CoverNotFound
		}
	}

	var fallback []byte
	for position+10 <= len(tag) {
		frameHeader := tag[position : position+10]
		if bytes.Equal(frameHeader[:4], []byte{0, 0, 0, 0}) {
			break
		}
		if !validID3FrameID(frameHeader[:4]) {
			break
		}
		frameSize, valid := id3FrameSize(frameHeader[4:8], header[3])
		position += 10
		if !valid || frameSize > len(tag)-position {
			break
		}
		if string(frameHeader[:4]) == "APIC" {
			imageData, pictureType, found := parseAPICFrame(tag[position : position+frameSize])
			if found {
				if pictureType == 3 {
					return writeJPEGCover(imageData, outputPath)
				}
				if fallback == nil {
					fallback = imageData
				}
			}
		}
		position += frameSize
	}
	if fallback == nil {
		return errMP3CoverNotFound
	}
	return writeJPEGCover(fallback, outputPath)
}

func synchsafeInteger(value []byte) (int, bool) {
	if len(value) != 4 || value[0]&0x80 != 0 || value[1]&0x80 != 0 || value[2]&0x80 != 0 || value[3]&0x80 != 0 {
		return 0, false
	}
	return int(value[0])<<21 | int(value[1])<<14 | int(value[2])<<7 | int(value[3]), true
}

func skipID3ExtendedHeader(tag []byte, version byte) (int, bool) {
	if len(tag) < 4 {
		return 0, false
	}
	if version == 3 {
		size := int(binary.BigEndian.Uint32(tag[:4]))
		position := 4 + size
		return position, position <= len(tag)
	}
	size, ok := synchsafeInteger(tag[:4])
	return size, ok && size >= 4 && size <= len(tag)
}

func validID3FrameID(value []byte) bool {
	if len(value) != 4 {
		return false
	}
	for _, character := range value {
		if !(character >= 'A' && character <= 'Z') && !(character >= '0' && character <= '9') {
			return false
		}
	}
	return true
}

func id3FrameSize(value []byte, version byte) (int, bool) {
	if version == 4 {
		return synchsafeInteger(value)
	}
	if len(value) != 4 {
		return 0, false
	}
	size := binary.BigEndian.Uint32(value)
	if size > uint32(maxMP3ID3TagSize) {
		return 0, false
	}
	return int(size), true
}

func parseAPICFrame(frame []byte) ([]byte, byte, bool) {
	if len(frame) < 4 {
		return nil, 0, false
	}
	encoding := frame[0]
	mimeEnd := bytes.IndexByte(frame[1:], 0)
	if mimeEnd < 0 {
		return nil, 0, false
	}
	position := 1 + mimeEnd + 1
	if position >= len(frame) {
		return nil, 0, false
	}
	pictureType := frame[position]
	position++
	descriptionLength, ok := id3TextTerminatorLength(frame[position:], encoding)
	if !ok {
		return nil, 0, false
	}
	position += descriptionLength
	if position >= len(frame) || len(frame)-position > maxEmbeddedCoverSize {
		return nil, 0, false
	}
	return frame[position:], pictureType, true
}

func id3TextTerminatorLength(value []byte, encoding byte) (int, bool) {
	switch encoding {
	case 0, 3:
		index := bytes.IndexByte(value, 0)
		if index < 0 {
			return 0, false
		}
		return index + 1, true
	case 1, 2:
		for index := 0; index+1 < len(value); index += 2 {
			if value[index] == 0 && value[index+1] == 0 {
				return index + 2, true
			}
		}
	}
	return 0, false
}

func writeJPEGCover(data []byte, outputPath string) error {
	if len(data) == 0 || len(data) > maxEmbeddedCoverSize {
		return errMP3CoverNotFound
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > maxEmbeddedCoverPixels/config.Height {
		return errMP3CoverNotFound
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return errMP3CoverNotFound
	}
	output, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(output, decoded, &jpeg.Options{Quality: 88}); err != nil {
		output.Close()
		_ = os.Remove(outputPath)
		return err
	}
	if err := output.Close(); err != nil {
		_ = os.Remove(outputPath)
		return err
	}
	return nil
}
