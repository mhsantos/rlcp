package storage

import (
	"sync"
	"testing"
	"time"
)

func collectFromChannel(ch chan []byte, done <-chan struct{}) []byte {
	var result []byte
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return result
			}
			result = append(result, data...)
		case <-done:
			// drain remaining
			for {
				select {
				case data, ok := <-ch:
					if !ok {
						return result
					}
					result = append(result, data...)
				default:
					return result
				}
			}
		}
	}
}

func TestProcessOutputAndRegisterListener(t *testing.T) {
	job := NewJob()

	ch := make(chan []byte, 16) // buffered to avoid blocking
	go job.RegisterListener(ch)

	// Give RegisterListener time to register
	time.Sleep(20 * time.Millisecond)

	input := []byte("hello world")
	if err := job.ProcessOutput(input); err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}

	select {
	case got := <-ch:
		if string(got) != string(input) {
			t.Errorf("listener received %q, want %q", got, input)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for listener output")
	}

	job.SetStatus(Completed)

	// Channel should be closed after terminal status
	timer := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // success: channel closed
			}
			// drain any remaining data
		case <-timer:
			t.Fatal("timed out waiting for channel close")
		}
	}
}

func TestLateListenerGetsHistory(t *testing.T) {
	job := NewJob()

	// Write output before any listener is registered
	input1 := []byte("first output")
	input2 := []byte("second output")
	if err := job.ProcessOutput(input1); err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}
	if err := job.ProcessOutput(input2); err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}

	// Now register a listener -- it should receive all prior output
	ch := make(chan []byte, 16)
	go job.RegisterListener(ch)

	var received []byte
	timeout := time.After(2 * time.Second)
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				goto done
			}
			received = append(received, data...)
			if len(received) >= len(input1)+len(input2) {
				goto done
			}
		case <-timeout:
			t.Fatalf("timed out waiting for historical output, got %q so far", received)
		}
	}
done:
	expected := string(input1) + string(input2)
	if string(received) != expected {
		t.Errorf("late listener received %q, want %q", received, expected)
	}

	job.SetStatus(Completed)
}

func TestMultipleConcurrentListeners(t *testing.T) {
	job := NewJob()

	const numListeners = 3
	channels := make([]chan []byte, numListeners)
	for i := range channels {
		channels[i] = make(chan []byte, 16)
		go job.RegisterListener(channels[i])
	}

	// Give listeners time to register
	time.Sleep(30 * time.Millisecond)

	input := []byte("broadcast message")
	if err := job.ProcessOutput(input); err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}

	for i, ch := range channels {
		select {
		case got := <-ch:
			if string(got) != string(input) {
				t.Errorf("listener %d received %q, want %q", i, got, input)
			}
		case <-time.After(time.Second):
			t.Fatalf("listener %d timed out", i)
		}
	}

	job.SetStatus(Completed)
}

func TestSliceAliasingProtection(t *testing.T) {
	job := NewJob()

	ch := make(chan []byte, 16)
	go job.RegisterListener(ch)
	time.Sleep(20 * time.Millisecond)

	if err := job.ProcessOutput([]byte("first")); err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}
	if err := job.ProcessOutput([]byte("second")); err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}

	got1 := <-ch
	got2 := <-ch

	if string(got1) != "first" {
		t.Errorf("first message = %q, want %q", got1, "first")
	}
	if string(got2) != "second" {
		t.Errorf("second message = %q, want %q", got2, "second")
	}

	job.SetStatus(Completed)
}

func TestSetStatusClosesListeners(t *testing.T) {
	job := NewJob()

	ch := make(chan []byte, 16)
	go job.RegisterListener(ch)
	time.Sleep(20 * time.Millisecond)

	job.SetStatus(Completed)

	// Channel should be closed
	timer := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				goto checkDone
			}
		case <-timer:
			t.Fatal("channel not closed after Completed status")
		}
	}
checkDone:
	// Done channel should also be closed
	select {
	case <-job.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("done channel not closed after Completed status")
	}
}

func TestGetSetStatusThreadSafety(t *testing.T) {
	job := NewJob()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			job.GetStatus()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			job.SetStatus(Running)
		}
	}()

	wg.Wait()
	job.SetStatus(Completed)
}

func TestDoubleSetTerminalStatus(t *testing.T) {
	job := NewJob()

	ch := make(chan []byte, 16)
	go job.RegisterListener(ch)
	time.Sleep(20 * time.Millisecond)

	// Should not panic when called twice with terminal status
	job.SetStatus(Completed)
	job.SetStatus(Stopped)
}
