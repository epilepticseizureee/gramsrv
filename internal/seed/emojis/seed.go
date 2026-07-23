package emojis

import "embed"

// FS contains the bundled custom emoji document catalog and blobs.
//
//go:embed default_emojis_seed.json documents/*
var FS embed.FS
