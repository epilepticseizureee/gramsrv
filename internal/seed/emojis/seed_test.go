package emojis

import "testing"

func TestDefaultEmojiSeedIsEmbedded(t *testing.T) {
	if _, err := FS.ReadFile("default_emojis_seed.json"); err != nil {
		t.Fatalf("default_emojis_seed.json not embedded: %v", err)
	}
}
