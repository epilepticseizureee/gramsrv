package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"

	"telesrv/internal/domain"
	"telesrv/internal/seed/emojis"
)

// EmojiSeedStats reports bundled custom emoji documents imported into media storage.
type EmojiSeedStats struct {
	Documents int
	Blobs     int
	Skipped   bool
}

// SeedEmojis imports bundled custom emoji documents into telesrv media storage.
func (s *Service) SeedEmojis(ctx context.Context) (EmojiSeedStats, error) {
	var stats EmojiSeedStats
	catalog, raw, err := loadEmojiSeedCatalog()
	if err != nil {
		return stats, err
	}
	if len(catalog.Documents) == 0 {
		stats.Skipped = true
		return stats, nil
	}

	stateHash, err := s.seedEmojisStateHash(raw)
	if err != nil {
		return stats, err
	}
	index := catalog.index()
	ready, err := s.seedDocumentJSONsReady(ctx, catalog.seedDocuments(), index)
	if err != nil {
		return stats, err
	}
	matched, err := s.seedStateMatches(ctx, seedEmojisStateKey, stateHash)
	if err != nil {
		return stats, err
	}
	if matched && ready {
		stats.Skipped = true
		stats.Documents = len(catalog.Documents)
		return stats, nil
	}

	var seedStats SeedStats
	seen := make(map[int64]bool, len(catalog.Documents))
	for _, doc := range catalog.Documents {
		if doc.ID == 0 || seen[doc.ID] {
			continue
		}
		seen[doc.ID] = true
		if _, err := s.seedEmojiDocument(ctx, doc, &seedStats); err != nil {
			return stats, fmt.Errorf("seed emoji document %d: %w", doc.ID, err)
		}
	}
	if err := s.putSeedState(ctx, seedEmojisStateKey, stateHash); err != nil {
		return stats, err
	}
	stats.Documents = seedStats.Documents
	stats.Blobs = seedStats.Blobs
	return stats, nil
}

func loadEmojiSeedCatalog() (emojiSeedCatalog, []byte, error) {
	raw, err := emojis.FS.ReadFile("default_emojis_seed.json")
	if err != nil {
		return emojiSeedCatalog{}, nil, err
	}
	var catalog emojiSeedCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return emojiSeedCatalog{}, nil, err
	}
	return catalog, raw, nil
}

func (s *Service) seedEmojisStateHash(raw []byte) (string, error) {
	return seedStateHash(func(h hash.Hash) error {
		writeSeedStateHeader(h, seedEmojisStateVersion, s.dc)
		_, _ = h.Write(raw)
		return nil
	})
}

func (s *Service) seedEmojiDocument(ctx context.Context, in emojiSeedDocumentJSON, stats *SeedStats) (domain.Document, error) {
	if in.ID == 0 {
		return domain.Document{}, nil
	}
	existing, existingFound, err := s.media.GetDocument(ctx, in.ID)
	if err != nil {
		return domain.Document{}, err
	}
	ref, _ := hex.DecodeString(in.FileReference)
	doc := domain.Document{
		ID:            in.ID,
		AccessHash:    in.AccessHash,
		FileReference: ref,
		Date:          parseSeedDate(in.Date),
		MimeType:      in.MimeType,
		Size:          in.Size,
		DCID:          s.dc,
		Attributes:    seedDocumentAttributes(in.Attributes),
	}

	if in.Path != "" {
		data, sum, err := readEmojiSeedBlob(in.Path, in.SHA256)
		if err != nil {
			return domain.Document{}, err
		}
		objectKey, err := s.blobs.Put(ctx, data)
		if err != nil {
			return domain.Document{}, err
		}
		if err := s.media.PutFileBlob(ctx, domain.FileBlob{
			LocationKey: fmt.Sprintf("doc:%d", doc.ID),
			Backend:     domain.MediaBackend(s.blobs.Name()),
			ObjectKey:   objectKey,
			Size:        int64(len(data)),
			SHA256:      sum,
			MimeType:    in.MimeType,
		}); err != nil {
			return domain.Document{}, err
		}
		s.prewarmSmallBlob(objectKey, data)
		stats.Blobs++
	}

	thumbs := make([]domain.PhotoSize, 0, len(in.Thumbs))
	for _, thumb := range in.Thumbs {
		ps, downloadable := seedPhotoSize(thumb.seedThumbJSON())
		if ps.Kind == "" {
			continue
		}
		if downloadable {
			if thumb.Path == "" {
				continue
			}
			data, sum, err := readEmojiSeedBlob(thumb.Path, thumb.SHA256)
			if err != nil {
				return domain.Document{}, err
			}
			ps.Size = len(data)
			ps = seedInlineCachedDocumentThumb(ps, data)
			if prior, ok := seedDocumentThumbByType(existing.Thumbs, ps.Type); existingFound && ok && seedPhotoSizeBetter(prior, ps) {
				ps = prior
			} else {
				objectKey, err := s.blobs.Put(ctx, data)
				if err != nil {
					return domain.Document{}, err
				}
				if err := s.media.PutFileBlob(ctx, domain.FileBlob{
					LocationKey: fmt.Sprintf("doc:%d:%s", doc.ID, ps.Type),
					Backend:     domain.MediaBackend(s.blobs.Name()),
					ObjectKey:   objectKey,
					Size:        int64(len(data)),
					SHA256:      sum,
					MimeType:    seedThumbMimeType(data),
				}); err != nil {
					return domain.Document{}, err
				}
				s.prewarmSmallBlob(objectKey, data)
				stats.Blobs++
			}
		}
		thumbs = append(thumbs, ps)
	}

	doc.Thumbs = thumbs
	if existingFound {
		doc.Thumbs = mergeSeedDocumentThumbs(existing.Thumbs, doc.Thumbs)
	}
	if err := s.ensureSeedCachedThumbBlobs(ctx, doc, stats); err != nil {
		return domain.Document{}, err
	}
	if err := s.ensureTGStickerPreviewThumb(ctx, &doc, stats); err != nil {
		return domain.Document{}, err
	}
	if err := s.media.PutDocument(ctx, doc); err != nil {
		return domain.Document{}, err
	}
	stats.Documents++
	return doc, nil
}

func readEmojiSeedBlob(path, wantSHA string) ([]byte, []byte, error) {
	data, err := emojis.FS.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if wantSHA != "" && got != wantSHA {
		return nil, nil, fmt.Errorf("%s sha256 = %s, want %s", path, got, wantSHA)
	}
	return data, append([]byte(nil), sum[:]...), nil
}

type emojiSeedCatalog struct {
	Source    string                  `json:"source"`
	Documents []emojiSeedDocumentJSON `json:"documents"`
}

func (c emojiSeedCatalog) seedDocuments() []seedDocumentJSON {
	docs := make([]seedDocumentJSON, 0, len(c.Documents))
	for _, doc := range c.Documents {
		docs = append(docs, doc.seedDocumentJSON())
	}
	return docs
}

func (c emojiSeedCatalog) index() seedDirIndex {
	index := seedDirIndex{
		main:  make(map[int64]string, len(c.Documents)),
		thumb: make(map[int64]map[string]string, len(c.Documents)),
	}
	for _, doc := range c.Documents {
		if doc.ID == 0 {
			continue
		}
		if doc.Path != "" {
			index.main[doc.ID] = doc.Path
		}
		for _, thumb := range doc.Thumbs {
			if thumb.Path == "" || thumb.SizeType == "" {
				continue
			}
			if index.thumb[doc.ID] == nil {
				index.thumb[doc.ID] = map[string]string{}
			}
			index.thumb[doc.ID][thumb.SizeType] = thumb.Path
		}
	}
	return index
}

type emojiSeedDocumentJSON struct {
	ID            int64                `json:"id"`
	AccessHash    int64                `json:"access_hash"`
	FileReference string               `json:"file_reference"`
	Date          string               `json:"date"`
	MimeType      string               `json:"mime_type"`
	Size          int64                `json:"size"`
	DCID          int                  `json:"dc_id"`
	Path          string               `json:"path"`
	SHA256        string               `json:"sha256"`
	Attributes    []seedAttrJSON       `json:"attributes"`
	Thumbs        []emojiSeedThumbJSON `json:"thumbs"`
}

func (d emojiSeedDocumentJSON) seedDocumentJSON() seedDocumentJSON {
	thumbs := make([]seedThumbJSON, 0, len(d.Thumbs))
	for _, thumb := range d.Thumbs {
		thumbs = append(thumbs, thumb.seedThumbJSON())
	}
	return seedDocumentJSON{
		ID:            d.ID,
		AccessHash:    d.AccessHash,
		FileReference: d.FileReference,
		Date:          d.Date,
		MimeType:      d.MimeType,
		Size:          d.Size,
		DCID:          d.DCID,
		Attributes:    d.Attributes,
		Thumbs:        thumbs,
	}
}

type emojiSeedThumbJSON struct {
	Type     string `json:"_"`
	SizeType string `json:"type"`
	W        int    `json:"w"`
	H        int    `json:"h"`
	Size     int    `json:"size"`
	Bytes    string `json:"bytes"`
	Sizes    []int  `json:"sizes"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
}

func (t emojiSeedThumbJSON) seedThumbJSON() seedThumbJSON {
	return seedThumbJSON{
		Type:     t.Type,
		SizeType: t.SizeType,
		W:        t.W,
		H:        t.H,
		Size:     t.Size,
		Bytes:    t.Bytes,
		Sizes:    t.Sizes,
	}
}
