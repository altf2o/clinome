package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// decodeWAV parses a PCM/float WAV file and returns interleaved stereo
// float32 samples resampled to targetRate.
func decodeWAV(data []byte, targetRate int) ([]float32, error) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, errors.New("not a RIFF/WAVE file")
	}

	var (
		format, channels, bits uint16
		rate                   uint32
		pcm                    []byte
		haveFmt                bool
	)
	for off := 12; off+8 <= len(data); {
		id := string(data[off : off+4])
		size := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		body := off + 8
		end := body + size
		if end > len(data) {
			end = len(data)
		}
		switch id {
		case "fmt ":
			if end-body < 16 {
				return nil, errors.New("short fmt chunk")
			}
			format = binary.LittleEndian.Uint16(data[body:])
			channels = binary.LittleEndian.Uint16(data[body+2:])
			rate = binary.LittleEndian.Uint32(data[body+4:])
			bits = binary.LittleEndian.Uint16(data[body+14:])
			if format == 0xFFFE && end-body >= 26 { // WAVE_FORMAT_EXTENSIBLE
				format = binary.LittleEndian.Uint16(data[body+24:])
			}
			haveFmt = true
		case "data":
			pcm = data[body:end]
		}
		off = body + size + size%2 // chunks are word aligned
	}
	if !haveFmt || pcm == nil {
		return nil, errors.New("missing fmt or data chunk")
	}
	if channels == 0 || channels > 32 {
		return nil, fmt.Errorf("unsupported channel count %d", channels)
	}
	// Bound the rate so resampling can't blow up a tiny file into gigabytes.
	if rate < 1000 || rate > 384000 {
		return nil, fmt.Errorf("unsupported sample rate %d", rate)
	}

	var sample func(b []byte) float32
	switch {
	case format == 1 && bits == 16:
		sample = func(b []byte) float32 { return float32(int16(binary.LittleEndian.Uint16(b))) / 32768 }
	case format == 1 && bits == 24:
		sample = func(b []byte) float32 {
			v := int32(b[0])<<8 | int32(b[1])<<16 | int32(b[2])<<24
			return float32(v) / 2147483648
		}
	case format == 1 && bits == 32:
		sample = func(b []byte) float32 { return float32(int32(binary.LittleEndian.Uint32(b))) / 2147483648 }
	case format == 3 && bits == 32:
		sample = func(b []byte) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b)) }
	default:
		return nil, fmt.Errorf("unsupported WAV format %d / %d bits", format, bits)
	}

	bps := int(bits / 8)
	frameSize := bps * int(channels)
	frames := len(pcm) / frameSize
	if frames == 0 {
		return nil, errors.New("empty data chunk")
	}
	out := make([]float32, frames*2)
	for i := 0; i < frames; i++ {
		f := pcm[i*frameSize:]
		l := sample(f)
		r := l
		if channels > 1 {
			r = sample(f[bps:])
		}
		out[2*i], out[2*i+1] = l, r
	}

	if int(rate) != targetRate {
		out = resample(out, int(rate), targetRate)
	}
	return out, nil
}

// resample does linear interpolation on interleaved stereo data.
func resample(in []float32, from, to int) []float32 {
	inFrames := len(in) / 2
	outFrames := int(int64(inFrames) * int64(to) / int64(from))
	out := make([]float32, outFrames*2)
	step := float64(from) / float64(to)
	for i := 0; i < outFrames; i++ {
		pos := float64(i) * step
		j := int(pos)
		frac := float32(pos - float64(j))
		k := j + 1
		if k >= inFrames {
			k = inFrames - 1
		}
		for c := 0; c < 2; c++ {
			out[2*i+c] = in[2*j+c]*(1-frac) + in[2*k+c]*frac
		}
	}
	return out
}
