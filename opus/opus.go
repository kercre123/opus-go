package opus

import (
	"encoding/binary"
	"fmt"

	"github.com/kercre123/opus-go/ogg"
	opus "github.com/pion/opus"
)

// FrameSizes is a list of valid frame sizes for Opus encoding
var FrameSizes = []float32{60, 40, 20, 10, 5, 2.5}

// OggStream is an encoder that takes audio samples and encodes them
// using the Opus audio codec into an Ogg container, using the given
// settings.
type OggStream struct {
	SampleRate uint
	Channels   uint
	Bitrate    uint
	FrameSize  float32
	Complexity uint
	stream     ogg.Stream
	encbuf     []byte
	decoder    opus.Decoder
}

func (s *OggStream) Decode(oggBytes []byte) ([]byte, error) {
	toReturn := make([]byte, 0)

	err := s.stream.SubmitDecodeBytes(oggBytes)
	if err != nil {
		return nil, err
	}

	for {
		buf, err := s.stream.DecodeBytesOut()
		if err != nil {
			return nil, err
		} else if buf == nil {
			break
		}
		bytesOut := make([]byte, 2048)
		_, _, err = s.decoder.Decode(buf, bytesOut)
		if err != nil {
			return nil, err
		}

		toReturn = append(toReturn, bytesOut...)
	}

	return toReturn, nil
}

// Flush attempts to flush any possible completed pages from the Ogg container
// if data was submitted that didn't generate new page output.
func (s *OggStream) Flush() []byte {
	return s.stream.Flush()
}

func bytesToSamples(buf []byte) []int16 {
	samples := make([]int16, len(buf)/2)
	for i := 0; i < len(buf)/2; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(buf[i*2:]))
	}
	return samples
}

func samplesToBytes(buf []int16) []byte {
	output := make([]byte, len(buf)*2)
	for i := 0; i < len(buf); i++ {
		binary.LittleEndian.PutUint16(output[2*i:2*(i+1)], uint16(buf[i]))
	}
	return output
}

func (s *OggStream) getFrameSamples(samples uint) (int, error) {
	nSamples := func(fs float32) uint {
		return uint(fs * float32(s.Channels*s.SampleRate/1000))
	}
	ideal := nSamples(s.FrameSize)
	if ideal <= samples {
		return int(ideal), nil
	}

	// loop over valid frame sizes until we find one small enough to encode this
	for _, val := range FrameSizes {
		// if this is bigger than our frame size we already know it won't work
		if val >= s.FrameSize {
			continue
		}
		i := nSamples(val)
		if i <= samples {
			return int(i), nil
		}
	}
	return 0, fmt.Errorf("Could not find valid frame size for %d samples @ %d/%d sample rate/channels", samples, s.Channels, s.SampleRate)
}

func getWriteError(err error, n int, desc string) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("%s: %d", desc, n)
}
