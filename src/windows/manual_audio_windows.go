//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

type manualPCM struct {
	format   WAVEFORMATEX
	pcm      []byte
	duration time.Duration
}

func manualReadPCM(path string) (manualPCM, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return manualPCM{}, err
	}
	if len(b) < 44 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return manualPCM{}, fmt.Errorf("invalid WAV: %s", path)
	}
	var data, fmtData []byte
	for off := 12; off+8 <= len(b); {
		n := int(binary.LittleEndian.Uint32(b[off+4 : off+8]))
		start := off + 8
		end := start + n
		if n < 0 || end > len(b) || end < start {
			break
		}
		switch string(b[off : off+4]) {
		case "fmt ":
			fmtData = b[start:end]
		case "data":
			data = b[start:end]
		}
		off = end
		if off%2 != 0 {
			off++
		}
	}
	if len(fmtData) < 16 || len(data) == 0 {
		return manualPCM{}, fmt.Errorf("WAV missing fmt/data: %s", path)
	}
	f := WAVEFORMATEX{
		FormatTag:      binary.LittleEndian.Uint16(fmtData[0:2]),
		Channels:       binary.LittleEndian.Uint16(fmtData[2:4]),
		SamplesPerSec:  binary.LittleEndian.Uint32(fmtData[4:8]),
		AvgBytesPerSec: binary.LittleEndian.Uint32(fmtData[8:12]),
		BlockAlign:     binary.LittleEndian.Uint16(fmtData[12:14]),
		BitsPerSample:  binary.LittleEndian.Uint16(fmtData[14:16]),
	}
	if f.FormatTag != WAVE_FORMAT_PCM || f.BitsPerSample != 16 || f.AvgBytesPerSec == 0 || f.BlockAlign == 0 {
		return manualPCM{}, fmt.Errorf("unsupported Piper WAV data")
	}
	frames := len(data) / int(f.BlockAlign)
	if frames == 0 {
		return manualPCM{}, fmt.Errorf("empty WAV data")
	}
	duration := time.Duration(int64(frames) * int64(time.Second) / int64(f.SamplesPerSec))
	return manualPCM{format: f, pcm: data[:frames*int(f.BlockAlign)], duration: duration}, nil
}
func manualWAVDuration(path string) (time.Duration, error) {
	wav, err := manualReadPCM(path)
	if err != nil {
		return 0, err
	}
	return wav.duration, nil
}

// Uses the same selected Windows audio device as the existing calculator,
// but independently controlled so narration cannot block operational Piper.
// Slice at a PCM *sample frame*, not at a guessed section boundary.
func manualPlayWAVFrom(path string, deviceIndex, volume int, offset time.Duration, cancel <-chan struct{}, progress func(time.Duration)) error {
	wav, err := manualReadPCM(path)
	if err != nil {
		return err
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= wav.duration {
		progress(wav.duration)
		return nil
	}
	f := wav.format
	bytesOffset := int(int64(offset) * int64(f.AvgBytesPerSec) / int64(time.Second))
	bytesOffset = bytesOffset / int(f.BlockAlign) * int(f.BlockAlign)
	if bytesOffset < 0 {
		bytesOffset = 0
	}
	if bytesOffset >= len(wav.pcm) {
		progress(wav.duration)
		return nil
	}
	// Do not mutate cache WAV; volume 0 is a silent local track, not global mute.
	pcm := append([]byte(nil), wav.pcm[bytesOffset:]...)
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	if volume != 100 {
		for i := 0; i+1 < len(pcm); i += 2 {
			sample := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
			scaled := int32(sample) * int32(volume) / 100
			binary.LittleEndian.PutUint16(pcm[i:i+2], uint16(int16(scaled)))
		}
	}
	var hwo uintptr
	device := uintptr(^uint32(0))
	if deviceIndex >= 0 {
		device = uintptr(deviceIndex)
	}
	code, _, _ := procWaveOutOpen.Call(uintptr(unsafe.Pointer(&hwo)), device, uintptr(unsafe.Pointer(&f)), 0, 0, CALLBACK_NULL)
	if code != 0 || hwo == 0 {
		return fmt.Errorf("narrator waveOutOpen failed: %d", code)
	}
	defer procWaveOutClose.Call(hwo)
	hdr := WAVEHDR{Data: &pcm[0], BufferLength: uint32(len(pcm))}
	code, _, _ = procWaveOutPrepare.Call(hwo, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	if code != 0 {
		return fmt.Errorf("narrator waveOutPrepareHeader failed: %d", code)
	}
	defer procWaveOutUnprepare.Call(hwo, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	select {
	case <-cancel:
		return nil
	default:
	}
	code, _, _ = procWaveOutWrite.Call(hwo, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	if code != 0 {
		return fmt.Errorf("narrator waveOutWrite failed: %d", code)
	}
	actualOffset := time.Duration(int64(bytesOffset) * int64(time.Second) / int64(f.AvgBytesPerSec))
	playedAt := time.Now()
	deadline := time.Now().Add(wav.duration - actualOffset + 20*time.Second)
	for hdr.Flags&WHDR_DONE == 0 {
		select {
		case <-cancel:
			procWaveOutReset.Call(hwo)
			return nil
		default:
		}
		// Query the Windows audio driver's sample position; use the clock only
		// on devices that do not support TIME_SAMPLES. This follows audible
		// PCM playback rather than making the slider a cosmetic animation.
		elapsed := time.Since(playedAt) + actualOffset
		var pos struct {
			Kind uint32
			Data [8]byte
		}
		pos.Kind = 2 // TIME_SAMPLES
		if got, _, _ := procWaveOutGetPosition.Call(hwo, uintptr(unsafe.Pointer(&pos)), unsafe.Sizeof(pos)); got == 0 && pos.Kind == 2 {
			frames := binary.LittleEndian.Uint32(pos.Data[:4])
			elapsed = actualOffset + time.Duration(int64(frames)*int64(time.Second)/int64(f.SamplesPerSec))
		}
		if elapsed > wav.duration {
			elapsed = wav.duration
		}
		if progress != nil {
			progress(elapsed)
		}
		if time.Now().After(deadline) {
			procWaveOutReset.Call(hwo)
			return fmt.Errorf("narration audio playback timeout")
		}
		time.Sleep(15 * time.Millisecond)
	}
	if progress != nil {
		progress(wav.duration)
	}
	runtime.KeepAlive(pcm)
	runtime.KeepAlive(hdr)
	runtime.KeepAlive(f)
	return nil
}

// Explicitly keeps the Windows syscall package available to guard the
// platform-specific audio signatures at build time.
var _ syscall.Handle
