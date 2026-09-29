package cvedit

import "testing"

func TestHashPrompts_ChangesWhenEitherPromptChanges(t *testing.T) {
	base := hashPrompts("voice", "rules")
	if hashPrompts("voice", "rules") != base {
		t.Error("hash is not stable")
	}
	if hashPrompts("voice!", "rules") == base {
		t.Error("hash unchanged after voice change")
	}
	if hashPrompts("voice", "rules!") == base {
		t.Error("hash unchanged after rules change")
	}
	if hashPrompts("voicer", "ules") == base {
		t.Error("hash ignores the voice/rules boundary")
	}
}

func TestPromptVersion_IsHashOfEmbeddedFiles(t *testing.T) {
	if PromptVersion == "" || PromptVersion != hashPrompts(voice, rules) {
		t.Errorf("PromptVersion = %q", PromptVersion)
	}
}
