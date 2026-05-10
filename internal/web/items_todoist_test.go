package web

import "testing"

func TestTodoistTaskIDFromSourceRef(t *testing.T) {
	cases := map[string]string{
		"task:123":            "123",
		"6XWMrGx9jxV6hHH3/123": "123",
		"work:3:list:123":     "123",
		"123":                 "123",
	}
	for input, want := range cases {
		if got := todoistTaskIDFromSourceRef(input); got != want {
			t.Fatalf("todoistTaskIDFromSourceRef(%q) = %q, want %q", input, got, want)
		}
	}
}
