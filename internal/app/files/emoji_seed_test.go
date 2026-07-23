package files

import (
	"context"
	"fmt"
	"testing"

	"telesrv/internal/domain"
)

func TestSeedEmojisImportsDefaultDocuments(t *testing.T) {
	media := newFakeMediaStore()
	blobs, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalFS: %v", err)
	}
	svc := NewService(media, blobs, 2)

	stats, err := svc.SeedEmojis(context.Background())
	if err != nil {
		t.Fatalf("SeedEmojis: %v", err)
	}
	if stats.Skipped || stats.Documents != 1 || stats.Blobs < 1 {
		t.Fatalf("SeedEmojis stats = %+v, want one document import", stats)
	}

	const documentID int64 = 5449449325434266744
	doc, ok, err := media.GetDocument(context.Background(), documentID)
	if err != nil || !ok {
		t.Fatalf("GetDocument(%d) = ok %v err %v", documentID, ok, err)
	}
	if doc.MimeType != "application/x-tgsticker" || doc.Size != 3666 || doc.DCID != 2 {
		t.Fatalf("document = dc %d mime %q size %d", doc.DCID, doc.MimeType, doc.Size)
	}
	if doc.AccessHash != -2674406205984148845 {
		t.Fatalf("document access_hash = %d, want -2674406205984148845", doc.AccessHash)
	}
	if !docHasEmojiAttribute(doc) {
		t.Fatalf("document attributes = %+v, want custom emoji", doc.Attributes)
	}
	if _, ok, err := media.GetFileBlob(context.Background(), fmt.Sprintf("doc:%d", documentID)); err != nil || !ok {
		t.Fatalf("main blob ok=%v err=%v, want present", ok, err)
	}
}

func docHasEmojiAttribute(doc domain.Document) bool {
	for _, attr := range doc.Attributes {
		if attr.Kind == domain.DocAttrCustomEmoji {
			return true
		}
	}
	return false
}
