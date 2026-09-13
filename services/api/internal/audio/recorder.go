package audio

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gordonklaus/portaudio"
)

const (
	SampleRate      = 16000
	Channels        = 1
	FramesPerBuffer = 1024
)

type Recorder struct {
	stream *portaudio.Stream
	buffer []int16
}

func NewRecorder() (*Recorder, error) {
	err := portaudio.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize portaudio: %w", err)
	}

	return &Recorder{
		buffer: make([]int16, FramesPerBuffer),
	}, nil
}

func (ar *Recorder) Close() error {
	if ar.stream != nil {
		ar.stream.Close()
	}
	return portaudio.Terminate()
}

// RecordAudio records audio until Enter is pressed or duration expires
func (ar *Recorder) RecordAudio(filename string, maxDuration time.Duration) error {
	stream, err := portaudio.OpenDefaultStream(Channels, 0, float64(SampleRate), FramesPerBuffer, ar.buffer)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	ar.stream = stream
	defer stream.Close()

	err = stream.Start()
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}
	defer stream.Stop()

	fmt.Println("Recording... Press Ctrl+C to stop")

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write WAV header
	err = writeWAVHeader(file, SampleRate, Channels, 16)
	if err != nil {
		return err
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup timeout
	timeout := time.After(maxDuration)

	done := make(chan bool)
	var recordedSamples int

	go func() {
		for {
			err := stream.Read()
			if err != nil {
				fmt.Printf("Error reading stream: %v\n", err)
				done <- true
				return
			}

			err = binary.Write(file, binary.LittleEndian, ar.buffer)
			if err != nil {
				fmt.Printf("Error writing to file: %v\n", err)
				done <- true
				return
			}
			recordedSamples += len(ar.buffer)
		}
	}()

	select {
	case <-sigChan:
		fmt.Println("\nRecording stopped by user")
	case <-timeout:
		fmt.Println("\nRecording stopped (max duration reached)")
	case <-done:
		fmt.Println("\nRecording stopped (error)")
	}

	// Update WAV header with actual size
	file.Seek(0, 0)
	err = writeWAVHeader(file, SampleRate, Channels, 16)
	if err != nil {
		return err
	}

	duration := float64(recordedSamples) / float64(SampleRate)
	fmt.Printf("Recorded %.2f seconds\n", duration)

	return nil
}

func writeWAVHeader(file *os.File, sampleRate, channels, bitsPerSample int) error {
	// RIFF header
	file.WriteString("RIFF")
	binary.Write(file, binary.LittleEndian, int32(0)) // File size - 8 (will update later)
	file.WriteString("WAVE")

	// fmt chunk
	file.WriteString("fmt ")
	binary.Write(file, binary.LittleEndian, int32(16)) // Chunk size
	binary.Write(file, binary.LittleEndian, int16(1))  // Audio format (PCM)
	binary.Write(file, binary.LittleEndian, int16(channels))
	binary.Write(file, binary.LittleEndian, int32(sampleRate))
	binary.Write(file, binary.LittleEndian, int32(sampleRate*channels*bitsPerSample/8)) // Byte rate
	binary.Write(file, binary.LittleEndian, int16(channels*bitsPerSample/8))            // Block align
	binary.Write(file, binary.LittleEndian, int16(bitsPerSample))

	// data chunk
	file.WriteString("data")

	// Get current file size to calculate data size
	pos, _ := file.Seek(0, 1)
	fileInfo, _ := file.Stat()
	dataSize := fileInfo.Size() - pos - 4

	binary.Write(file, binary.LittleEndian, int32(dataSize))

	return nil
}
