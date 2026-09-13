package lesson

import "testing"

func TestSessionAdvancesInOrderAndCompletes(t *testing.T) {
	steps := []LessonStep{
		{StepID: "intro"},
		{StepID: "rules"},
	}

	session, err := NewSession("welcome", steps)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	if got := session.Current().StepID; got != "intro" {
		t.Fatalf("Current().StepID = %q, want intro", got)
	}

	got, completed := session.Advance()
	if completed {
		t.Fatal("Advance() completed the session too early")
	}
	if got.StepID != "rules" {
		t.Fatalf("Advance().StepID = %q, want rules", got.StepID)
	}

	_, completed = session.Advance()
	if !completed {
		t.Fatal("Advance() did not complete the session")
	}
}

func TestNewSessionRejectsEmptySteps(t *testing.T) {
	if _, err := NewSession("welcome", nil); err == nil {
		t.Fatal("NewSession() accepted an empty lesson")
	}
}
