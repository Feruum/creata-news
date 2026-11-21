package achievements

import "testing"

func TestAutomaticAwardState(t *testing.T) {
	tests := []struct {
		name      string
		old       awardState
		qualifies bool
		want      awardState
	}{
		{"qualification grants automatic", awardState{}, true, awardState{Automatic: true}},
		{"manual persists when qualification lost", awardState{Manual: true, Automatic: true}, false, awardState{Manual: true}},
		{"revocation prevents automatic restoration", awardState{Suppressed: true}, true, awardState{Suppressed: true}},
		{"automatic removed on lost qualification", awardState{Automatic: true}, false, awardState{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := automaticState(tt.old, tt.qualifies); got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVisibility(t *testing.T) {
	for _, state := range []awardState{{}, {Manual: true, Suppressed: true}, {Automatic: true, Suppressed: true}} {
		if state.visible() {
			t.Fatalf("unexpected visible: %+v", state)
		}
	}
	if !(awardState{Manual: true}).visible() || !(awardState{Automatic: true}).visible() {
		t.Fatal("award should be visible")
	}
}
