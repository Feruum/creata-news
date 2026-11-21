package roomid

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "normalizes map room id to uppercase", input: " c1.2.223k ", expected: "C1.2.223K"},
		{name: "Cyrillic campus prefix", input: " С1.3.366L ", expected: "C1.3.366L"},
		{name: "lowercase Cyrillic campus prefix", input: "с1.3.366l", expected: "C1.3.366L"},
		{name: "canonical Korkem room id", input: "K.201", expected: "K.201"},
		{name: "lowercase Korkem room id", input: " k.201 ", expected: "K.201"},
		{name: "Cyrillic Korkem room id", input: "К.201", expected: "K.201"},
		{name: "lowercase Cyrillic Korkem room id", input: "к.201", expected: "K.201"},
		{name: "Korkem cafe", input: "k.cafe", expected: "K.CAFE"},
		{name: "Atrium", input: " atrium ", expected: "ATRIUM"},
		{name: "Coworking", input: "coworking", expected: "COWORKING"},
		{name: "Gymnasium", input: "fullgym", expected: "FULLGYM"},
		{name: "Cafeteria", input: "dininghall", expected: "DININGHALL"},
		{name: "preserves named room id casing", input: "  Media_Studio  ", expected: "Media_Studio"},
		{name: "empty string", input: "  ", expected: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := Normalize(testCase.input)
			if result != testCase.expected {
				t.Fatalf("Normalize(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

func TestNormalizeListKorkem(t *testing.T) {
	input := []string{"k.201", "К.201", "K.201", "k.cafe", "K.CAFE", "Media_Studio", "media_studio"}
	expected := []string{"K.201", "K.CAFE", "Media_Studio", "media_studio"}

	result := NormalizeList(input)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("NormalizeList() = %v, want %v", result, expected)
	}
}

func TestNormalizeList(t *testing.T) {
	input := []string{" c1.1.221p ", "C1.1.221P", "", "  ", "Media_Studio", " Media_Studio "}
	expected := []string{"C1.1.221P", "Media_Studio"}

	result := NormalizeList(input)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("NormalizeList() = %v, want %v", result, expected)
	}
}
