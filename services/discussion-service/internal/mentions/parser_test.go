package mentions

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"empty", "", nil},
		{"none", "hello world", nil},
		{"single", "hey @alice take a look", []string{"alice"}},
		{"start of string", "@alice ping", []string{"alice"}},
		{"multiple unique", "@alice and @bob please review", []string{"alice", "bob"}},
		{"dedupe case insensitive", "@Alice @alice @ALICE", []string{"Alice"}},
		{"ignore email", "send mail to foo@bar.com please", nil},
		{"trailing punctuation", "hi @carol, thanks", []string{"carol"}},
		{"dot in username", "@dr.who check", []string{"dr.who"}},
		{"hyphen and underscore", "@jane-doe @jane_doe", []string{"jane-doe", "jane_doe"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}
