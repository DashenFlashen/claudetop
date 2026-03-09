package session

import (
	"testing"
	"time"
)

func TestDetectStarting(t *testing.T) {
	now := time.Now()
	status := Detect("some output", now, now.Add(-5*time.Second), StatusStarting)
	if status != StatusStarting {
		t.Errorf("expected Starting for new session, got %v", status)
	}
}

func TestDetectWorking(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-1 * time.Second)
	status := Detect("Running bash...", lastOutput, createdAt, StatusDone)
	if status != StatusWorking {
		t.Errorf("expected Working for recent output, got %v", status)
	}
}

func TestDetectNeedsInput(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-10 * time.Second)
	output := "I've analyzed the code. Would you like me to proceed with the fix?"
	status := Detect(output, lastOutput, createdAt, StatusDone)
	if status != StatusNeedsInput {
		t.Errorf("expected NeedsInput, got %v", status)
	}
}

func TestDetectStuck(t *testing.T) {
	createdAt := time.Now().Add(-10 * time.Minute)
	lastOutput := time.Now().Add(-3 * time.Minute)
	status := Detect("Running tool...", lastOutput, createdAt, StatusWorking)
	if status != StatusStuck {
		t.Errorf("expected Stuck, got %v", status)
	}
}

func TestDetectError(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-5 * time.Second)
	status := Detect("Error: connection refused", lastOutput, createdAt, StatusWorking)
	if status != StatusError {
		t.Errorf("expected Error, got %v", status)
	}
}

func TestDetectDone(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-30 * time.Second)
	output := "I've completed the task. The fix has been applied successfully."
	status := Detect(output, lastOutput, createdAt, StatusDone)
	if status != StatusDone {
		t.Errorf("expected Done for silent non-question output, got %v", status)
	}
}
