package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"telesrv/internal/domain"
	"telesrv/internal/store"
	"telesrv/internal/store/postgres/sqlcgen"
)

// MediaStore 用 PostgreSQL 实现 store.MediaStore（媒体元数据 + blob 索引）。
type MediaStore struct {
	db sqlcgen.DBTX
	q  *sqlcgen.Queries
}

// NewMediaStore 基于 pgx 连接池（或事务）创建 MediaStore。
func NewMediaStore(db sqlcgen.DBTX) *MediaStore {
	return &MediaStore{db: db, q: sqlcgen.New(db)}
}

// bytesOrEmpty 把 nil []byte 归一为空切片，避免落入 NOT NULL bytea 列时被当作 NULL。
func bytesOrEmpty(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}

var _ store.MediaStore = (*MediaStore)(nil)

// ---- 上传分片 ----

func (s *MediaStore) SaveFilePart(ctx context.Context, part domain.UploadPart) error {
	return s.q.SaveUploadPart(ctx, sqlcgen.SaveUploadPartParams{
		OwnerUserID: part.OwnerUserID,
		FileID:      part.FileID,
		Part:        int32(part.Part),
		TotalParts:  int32(part.TotalParts),
		IsBig:       part.Big,
		Bytes:       part.Bytes,
	})
}

func (s *MediaStore) LoadFileParts(ctx context.Context, ownerUserID, fileID int64) ([]domain.UploadPart, error) {
	rows, err := s.q.ListUploadParts(ctx, sqlcgen.ListUploadPartsParams{OwnerUserID: ownerUserID, FileID: fileID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.UploadPart, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.UploadPart{
			OwnerUserID: ownerUserID,
			FileID:      fileID,
			Part:        int(r.Part),
			TotalParts:  int(r.TotalParts),
			Big:         r.IsBig,
			Bytes:       r.Bytes,
		})
	}
	return out, nil
}

func (s *MediaStore) DeleteFileParts(ctx context.Context, ownerUserID, fileID int64) error {
	return s.q.DeleteUploadParts(ctx, sqlcgen.DeleteUploadPartsParams{OwnerUserID: ownerUserID, FileID: fileID})
}

// ---- blob 索引 ----

func (s *MediaStore) PutFileBlob(ctx context.Context, blob domain.FileBlob) error {
	backend := string(blob.Backend)
	if backend == "" {
		backend = string(domain.MediaBackendLocalFS)
	}
	sha := blob.SHA256
	if sha == nil {
		sha = []byte{} // 列为 NOT NULL；nil []byte 会被 pgx 当作 NULL。
	}
	return s.q.PutFileBlob(ctx, sqlcgen.PutFileBlobParams{
		LocationKey: blob.LocationKey,
		Backend:     backend,
		ObjectKey:   blob.ObjectKey,
		Size:        blob.Size,
		Sha256:      sha,
		MimeType:    blob.MimeType,
	})
}

func (s *MediaStore) GetFileBlob(ctx context.Context, locationKey string) (domain.FileBlob, bool, error) {
	row, err := s.q.GetFileBlob(ctx, locationKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.FileBlob{}, false, nil
		}
		return domain.FileBlob{}, false, err
	}
	return domain.FileBlob{
		LocationKey: row.LocationKey,
		Backend:     domain.MediaBackend(row.Backend),
		ObjectKey:   row.ObjectKey,
		Size:        row.Size,
		SHA256:      row.Sha256,
		MimeType:    row.MimeType,
	}, true, nil
}

// ---- 文档 ----

func (s *MediaStore) PutDocument(ctx context.Context, doc domain.Document) error {
	attrs, err := jsonArrayOrEmpty(doc.Attributes)
	if err != nil {
		return err
	}
	thumbs, err := jsonArrayOrEmpty(doc.Thumbs)
	if err != nil {
		return err
	}
	return s.q.PutDocument(ctx, sqlcgen.PutDocumentParams{
		ID:             doc.ID,
		AccessHash:     doc.AccessHash,
		FileReference:  bytesOrEmpty(doc.FileReference),
		Date:           int32(doc.Date),
		MimeType:       doc.MimeType,
		Size:           doc.Size,
		DcID:           int32(doc.DCID),
		AttributesJson: attrs,
		ThumbsJson:     thumbs,
	})
}

func (s *MediaStore) GetDocument(ctx context.Context, id int64) (domain.Document, bool, error) {
	row, err := s.q.GetDocument(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Document{}, false, nil
		}
		return domain.Document{}, false, err
	}
	doc, err := documentFromRow(row)
	if err != nil {
		return domain.Document{}, false, err
	}
	return doc, true, nil
}

func (s *MediaStore) GetDocuments(ctx context.Context, ids []int64) ([]domain.Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.q.GetDocuments(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Document, 0, len(rows))
	for _, r := range rows {
		doc, err := documentFromRow(sqlcgen.GetDocumentRow(r))
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

func documentFromRow(row sqlcgen.GetDocumentRow) (domain.Document, error) {
	attrs, err := decodeDocumentAttributes(row.AttributesJson)
	if err != nil {
		return domain.Document{}, err
	}
	thumbs, err := decodePhotoSizes(row.ThumbsJson)
	if err != nil {
		return domain.Document{}, err
	}
	return domain.Document{
		ID:            row.ID,
		AccessHash:    row.AccessHash,
		FileReference: row.FileReference,
		Date:          int(row.Date),
		MimeType:      row.MimeType,
		Size:          row.Size,
		DCID:          int(row.DcID),
		Attributes:    attrs,
		Thumbs:        thumbs,
	}, nil
}

// ---- 照片 ----

func (s *MediaStore) PutPhoto(ctx context.Context, photo domain.Photo) error {
	sizes, err := jsonArrayOrEmpty(photo.Sizes)
	if err != nil {
		return err
	}
	return s.q.PutPhoto(ctx, sqlcgen.PutPhotoParams{
		ID:            photo.ID,
		AccessHash:    photo.AccessHash,
		FileReference: bytesOrEmpty(photo.FileReference),
		Date:          int32(photo.Date),
		DcID:          int32(photo.DCID),
		HasStickers:   photo.HasStickers,
		SizesJson:     sizes,
	})
}

func (s *MediaStore) GetPhoto(ctx context.Context, id int64) (domain.Photo, bool, error) {
	row, err := s.q.GetPhoto(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Photo{}, false, nil
		}
		return domain.Photo{}, false, err
	}
	sizes, err := decodePhotoSizes(row.SizesJson)
	if err != nil {
		return domain.Photo{}, false, err
	}
	return domain.Photo{
		ID:            row.ID,
		AccessHash:    row.AccessHash,
		FileReference: row.FileReference,
		Date:          int(row.Date),
		DCID:          int(row.DcID),
		HasStickers:   row.HasStickers,
		Sizes:         sizes,
	}, true, nil
}

// ---- 贴纸集 ----

func (s *MediaStore) PutStickerSet(ctx context.Context, set domain.StickerSet) error {
	thumbs, err := jsonArrayOrEmpty(set.Thumbs)
	if err != nil {
		return err
	}
	docIDs, err := jsonArrayOrEmpty(set.DocumentIDs)
	if err != nil {
		return err
	}
	packs, err := jsonArrayOrEmpty(set.Packs)
	if err != nil {
		return err
	}
	kind := string(set.Kind)
	if kind == "" {
		kind = string(domain.StickerSetKindStickers)
	}
	return s.q.PutStickerSet(ctx, sqlcgen.PutStickerSetParams{
		ID:              set.ID,
		AccessHash:      set.AccessHash,
		ShortName:       set.ShortName,
		Title:           set.Title,
		Count:           int32(set.Count),
		Hash:            int32(set.Hash),
		SetKind:         kind,
		Official:        set.Official,
		Animated:        set.Animated,
		Videos:          set.Videos,
		Emojis:          set.Emojis,
		Masks:           set.Masks,
		Installed:       set.Installed,
		Archived:        set.Archived,
		InstalledDate:   int32(set.InstalledDate),
		ThumbDocumentID: set.ThumbDocumentID,
		ThumbsJson:      thumbs,
		ThumbDcID:       int32(set.ThumbDCID),
		ThumbVersion:    int32(set.ThumbVersion),
		DocumentIdsJson: docIDs,
		PacksJson:       packs,
		SortOrder:       int32(set.SortOrder),
		SystemKey:       set.SystemKey,
	})
}

func (s *MediaStore) GetStickerSetByID(ctx context.Context, id int64) (domain.StickerSet, bool, error) {
	row, err := s.q.GetStickerSetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.StickerSet{}, false, nil
		}
		return domain.StickerSet{}, false, err
	}
	return stickerSetFromRow(row)
}

func (s *MediaStore) GetStickerSetByShortName(ctx context.Context, shortName string) (domain.StickerSet, bool, error) {
	row, err := s.q.GetStickerSetByShortName(ctx, shortName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.StickerSet{}, false, nil
		}
		return domain.StickerSet{}, false, err
	}
	return stickerSetFromRow(sqlcgen.GetStickerSetByIDRow(row))
}

func (s *MediaStore) GetStickerSetBySystemKey(ctx context.Context, systemKey string) (domain.StickerSet, bool, error) {
	row, err := s.q.GetStickerSetBySystemKey(ctx, systemKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.StickerSet{}, false, nil
		}
		return domain.StickerSet{}, false, err
	}
	return stickerSetFromRow(sqlcgen.GetStickerSetByIDRow(row))
}

func (s *MediaStore) ListStickerSets(ctx context.Context, kind domain.StickerSetKind) ([]domain.StickerSet, error) {
	rows, err := s.q.ListStickerSetsByKind(ctx, string(kind))
	if err != nil {
		return nil, err
	}
	out := make([]domain.StickerSet, 0, len(rows))
	for _, r := range rows {
		set, _, err := stickerSetFromRow(sqlcgen.GetStickerSetByIDRow(r))
		if err != nil {
			return nil, err
		}
		out = append(out, set)
	}
	return out, nil
}

func (s *MediaStore) CountStickerSets(ctx context.Context) (int, error) {
	n, err := s.q.CountStickerSets(ctx)
	return int(n), err
}

func stickerSetFromRow(row sqlcgen.GetStickerSetByIDRow) (domain.StickerSet, bool, error) {
	thumbs, err := decodePhotoSizes(row.ThumbsJson)
	if err != nil {
		return domain.StickerSet{}, false, err
	}
	docIDs, err := decodeInt64Slice(row.DocumentIdsJson)
	if err != nil {
		return domain.StickerSet{}, false, err
	}
	packs, err := decodeStickerPacks(row.PacksJson)
	if err != nil {
		return domain.StickerSet{}, false, err
	}
	return domain.StickerSet{
		ID:              row.ID,
		AccessHash:      row.AccessHash,
		ShortName:       row.ShortName,
		Title:           row.Title,
		Count:           int(row.Count),
		Hash:            int(row.Hash),
		Kind:            domain.StickerSetKind(row.SetKind),
		Official:        row.Official,
		Animated:        row.Animated,
		Videos:          row.Videos,
		Emojis:          row.Emojis,
		Masks:           row.Masks,
		Installed:       row.Installed,
		Archived:        row.Archived,
		InstalledDate:   int(row.InstalledDate),
		ThumbDocumentID: row.ThumbDocumentID,
		Thumbs:          thumbs,
		ThumbDCID:       int(row.ThumbDcID),
		ThumbVersion:    int(row.ThumbVersion),
		DocumentIDs:     docIDs,
		Packs:           packs,
		SortOrder:       int(row.SortOrder),
		SystemKey:       row.SystemKey,
	}, true, nil
}

// ---- 可用 reaction ----

func (s *MediaStore) PutAvailableReaction(ctx context.Context, r domain.AvailableReaction) error {
	return s.q.PutAvailableReaction(ctx, sqlcgen.PutAvailableReactionParams{
		Reaction:            r.Reaction,
		Title:               r.Title,
		Inactive:            r.Inactive,
		Premium:             r.Premium,
		StaticIconID:        r.StaticIconID,
		AppearAnimationID:   r.AppearAnimationID,
		SelectAnimationID:   r.SelectAnimationID,
		ActivateAnimationID: r.ActivateAnimationID,
		EffectAnimationID:   r.EffectAnimationID,
		AroundAnimationID:   r.AroundAnimationID,
		CenterIconID:        r.CenterIconID,
		SortOrder:           int32(r.Order),
	})
}

func (s *MediaStore) ListAvailableReactions(ctx context.Context) ([]domain.AvailableReaction, error) {
	rows, err := s.q.ListAvailableReactions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AvailableReaction, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AvailableReaction{
			Reaction:            r.Reaction,
			Title:               r.Title,
			Inactive:            r.Inactive,
			Premium:             r.Premium,
			StaticIconID:        r.StaticIconID,
			AppearAnimationID:   r.AppearAnimationID,
			SelectAnimationID:   r.SelectAnimationID,
			ActivateAnimationID: r.ActivateAnimationID,
			EffectAnimationID:   r.EffectAnimationID,
			AroundAnimationID:   r.AroundAnimationID,
			CenterIconID:        r.CenterIconID,
			Order:               int(r.SortOrder),
		})
	}
	return out, nil
}

func (s *MediaStore) CountAvailableReactions(ctx context.Context) (int, error) {
	n, err := s.q.CountAvailableReactions(ctx)
	return int(n), err
}

// ---- 头像历史 ----

func (s *MediaStore) AddProfilePhoto(ctx context.Context, ownerType domain.PeerType, ownerID, photoID int64, date int) error {
	return s.AddProfilePhotoKind(ctx, ownerType, ownerID, domain.ProfilePhotoKindProfile, photoID, date)
}

func (s *MediaStore) AddProfilePhotoKind(ctx context.Context, ownerType domain.PeerType, ownerID int64, kind domain.ProfilePhotoKind, photoID int64, date int) error {
	kind = normalizeProfilePhotoKind(kind)
	next, err := s.nextProfilePhotoOrder(ctx, ownerType, ownerID, kind)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
INSERT INTO profile_photos (owner_peer_type, owner_peer_id, kind, photo_id, date, active, sort_order)
VALUES ($1, $2, $3, $4, $5, true, $6)
ON CONFLICT (owner_peer_type, owner_peer_id, kind, photo_id) DO UPDATE SET
  date = EXCLUDED.date,
  active = true,
  sort_order = EXCLUDED.sort_order
`, string(ownerType), ownerID, string(kind), photoID, date, next+1)
	return err
}

func (s *MediaStore) CurrentProfilePhoto(ctx context.Context, ownerType domain.PeerType, ownerID int64) (int64, bool, error) {
	return s.CurrentProfilePhotoKind(ctx, ownerType, ownerID, domain.ProfilePhotoKindProfile)
}

func (s *MediaStore) CurrentProfilePhotoKind(ctx context.Context, ownerType domain.PeerType, ownerID int64, kind domain.ProfilePhotoKind) (int64, bool, error) {
	kind = normalizeProfilePhotoKind(kind)
	row := s.db.QueryRow(ctx, `
SELECT photo_id
FROM profile_photos
WHERE owner_peer_type = $1
  AND owner_peer_id = $2
  AND kind = $3
  AND active
ORDER BY sort_order DESC
LIMIT 1
`, string(ownerType), ownerID, string(kind))
	var id int64
	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return id, true, nil
}

func (s *MediaStore) CurrentProfilePhotos(ctx context.Context, ownerType domain.PeerType, ownerIDs []int64) (map[int64]domain.ProfilePhotoRef, error) {
	return s.CurrentProfilePhotosKind(ctx, ownerType, ownerIDs, domain.ProfilePhotoKindProfile)
}

func (s *MediaStore) CurrentProfilePhotosKind(ctx context.Context, ownerType domain.PeerType, ownerIDs []int64, kind domain.ProfilePhotoKind) (map[int64]domain.ProfilePhotoRef, error) {
	if len(ownerIDs) == 0 {
		return map[int64]domain.ProfilePhotoRef{}, nil
	}
	kind = normalizeProfilePhotoKind(kind)
	rows, err := s.db.Query(ctx, `
SELECT DISTINCT ON (pp.owner_peer_id)
  pp.owner_peer_id,
  pp.photo_id,
  ph.dc_id,
  ph.sizes::text AS sizes_json
FROM profile_photos pp
JOIN photos ph ON ph.id = pp.photo_id
WHERE pp.owner_peer_type = $1
  AND pp.owner_peer_id = ANY($2::bigint[])
  AND pp.kind = $3
  AND pp.active
ORDER BY pp.owner_peer_id, pp.sort_order DESC
`, string(ownerType), ownerIDs, string(kind))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]domain.ProfilePhotoRef, len(ownerIDs))
	for rows.Next() {
		var ownerID, photoID int64
		var dcID int32
		var sizesJSON string
		if err := rows.Scan(&ownerID, &photoID, &dcID, &sizesJSON); err != nil {
			return nil, err
		}
		sizes, err := decodePhotoSizes(sizesJSON)
		if err != nil {
			return nil, err
		}
		out[ownerID] = domain.ProfilePhotoRef{
			PhotoID:  photoID,
			DCID:     int(dcID),
			Stripped: domain.StrippedFromSizes(sizes),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *MediaStore) ListProfilePhotos(ctx context.Context, ownerType domain.PeerType, ownerID int64, offset, limit int, maxID int64) ([]int64, int, error) {
	return s.ListProfilePhotosKind(ctx, ownerType, ownerID, domain.ProfilePhotoKindProfile, offset, limit, maxID)
}

func (s *MediaStore) ListProfilePhotosKind(ctx context.Context, ownerType domain.PeerType, ownerID int64, kind domain.ProfilePhotoKind, offset, limit int, maxID int64) ([]int64, int, error) {
	kind = normalizeProfilePhotoKind(kind)
	rows, err := s.db.Query(ctx, `
SELECT photo_id
FROM profile_photos
WHERE owner_peer_type = $1
  AND owner_peer_id = $2
  AND kind = $3
  AND active
  AND ($4::bigint <= 0 OR photo_id < $4::bigint)
ORDER BY sort_order DESC
OFFSET $5
LIMIT $6
`, string(ownerType), ownerID, string(kind), maxID, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err = s.db.QueryRow(ctx, `
SELECT count(*)::int
FROM profile_photos
WHERE owner_peer_type = $1
  AND owner_peer_id = $2
  AND kind = $3
  AND active
`, string(ownerType), ownerID, string(kind)).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return ids, total, nil
}

func (s *MediaStore) DeleteProfilePhotos(ctx context.Context, ownerType domain.PeerType, ownerID int64, photoIDs []int64) ([]int64, error) {
	return s.DeleteProfilePhotosKind(ctx, ownerType, ownerID, domain.ProfilePhotoKindProfile, photoIDs)
}

func (s *MediaStore) DeleteProfilePhotosKind(ctx context.Context, ownerType domain.PeerType, ownerID int64, kind domain.ProfilePhotoKind, photoIDs []int64) ([]int64, error) {
	if len(photoIDs) == 0 {
		return nil, nil
	}
	kind = normalizeProfilePhotoKind(kind)
	rows, err := s.db.Query(ctx, `
UPDATE profile_photos
SET active = false
WHERE owner_peer_type = $1
  AND owner_peer_id = $2
  AND kind = $3
  AND photo_id = ANY($4::bigint[])
  AND active
RETURNING photo_id
`, string(ownerType), ownerID, string(kind), photoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deleted := make([]int64, 0, len(photoIDs))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		deleted = append(deleted, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return deleted, nil
}

func (s *MediaStore) nextProfilePhotoOrder(ctx context.Context, ownerType domain.PeerType, ownerID int64, kind domain.ProfilePhotoKind) (int64, error) {
	var maxOrder int64
	err := s.db.QueryRow(ctx, `
SELECT COALESCE(MAX(sort_order), 0)::bigint
FROM profile_photos
WHERE owner_peer_type = $1
  AND owner_peer_id = $2
  AND kind = $3
`, string(ownerType), ownerID, string(kind)).Scan(&maxOrder)
	return maxOrder, err
}

func normalizeProfilePhotoKind(kind domain.ProfilePhotoKind) domain.ProfilePhotoKind {
	if kind == domain.ProfilePhotoKindFallback {
		return kind
	}
	return domain.ProfilePhotoKindProfile
}
