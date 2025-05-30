package postgres

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/glanceapp/glance/pkg/sources/activities"
	"github.com/glanceapp/glance/pkg/sources/activities/types"
	"github.com/pgvector/pgvector-go"

	"github.com/glanceapp/glance/pkg/storage/postgres/ent"
	"github.com/glanceapp/glance/pkg/storage/postgres/ent/activity"
)

type ActivityRepository struct {
	db *DB
}

func NewActivityRepository(db *DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) Add(activity *types.DecoratedActivity) error {
	ctx := context.Background()

	rawJson, err := activity.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal activity: %w", err)
	}

	_, err = r.db.Client().Activity.Create().
		SetID(activity.UID()).
		SetUID(activity.UID()).
		SetSourceUID(activity.SourceUID()).
		SetTitle(activity.Title()).
		SetBody(activity.Body()).
		SetURL(activity.URL()).
		SetImageURL(activity.ImageURL()).
		SetCreatedAt(activity.CreatedAt()).
		SetSourceType(activity.SourceType()).
		SetRawJSON(string(rawJson)).
		SetShortSummary(activity.Summary.ShortSummary).
		SetFullSummary(activity.Summary.FullSummary).
		SetEmbedding(pgvector.NewVector(activity.Embedding)).
		Save(ctx)

	return err
}

func (r *ActivityRepository) Remove(uid string) error {
	ctx := context.Background()
	return r.db.Client().Activity.DeleteOneID(uid).Exec(ctx)
}

func (r *ActivityRepository) List() ([]*types.DecoratedActivity, error) {
	ctx := context.Background()

	results, err := r.db.Client().Activity.Query().
		Order(ent.Desc(activity.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*types.DecoratedActivity, len(results))
	for i, a := range results {
		out, err := activityFromEnt(a)
		if err != nil {
			return nil, fmt.Errorf("deserialize activity: %w", err)
		}
		result[i] = out
	}

	return result, nil
}

func (r *ActivityRepository) Search(req types.SearchRequest) ([]*types.DecoratedActivity, error) {
	ctx := context.Background()

	query := r.db.Client().Activity.Query()

	// Apply source filter if specified
	if len(req.SourceUIDs) > 0 {
		query = query.Where(activity.SourceUIDIn(req.SourceUIDs...))
	}

	// Only compute similarity if query embedding is provided
	if len(req.QueryEmbedding) > 0 {
		vector := pgvector.NewVector(req.QueryEmbedding)
		simExprStr := fmt.Sprintf("(1 - (embedding <=> '%s'))", vector)

		query = query.Order(func(s *sql.Selector) {
			s.AppendSelect(sql.As(simExprStr, "similarity"))
			// Only apply similarity filter if min similarity is specified
			if req.MinSimilarity > 0 {
				s.Where(sql.GT(simExprStr, req.MinSimilarity))
			}
			s.OrderExpr(sql.Expr("similarity DESC"))
		})
	} else {
		// If no query embedding, set similarity to 0
		query = query.Order(func(s *sql.Selector) {
			s.AppendSelect(sql.As("CAST(0 AS float8)", "similarity"))
		})
	}

	// Apply limit if specified
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}

	fields := []string{
		activity.FieldID,
		activity.FieldUID,
		activity.FieldSourceUID,
		activity.FieldSourceType,
		activity.FieldTitle,
		activity.FieldBody,
		activity.FieldURL,
		activity.FieldImageURL,
		activity.FieldCreatedAt,
		activity.FieldShortSummary,
		activity.FieldFullSummary,
		activity.FieldRawJSON,
		activity.FieldEmbedding,
	}

	var rows []struct {
		ID           string          `json:"id"`
		UID          string          `json:"uid"`
		SourceUID    string          `json:"source_uid"`
		SourceType   string          `json:"source_type"`
		Title        string          `json:"title"`
		Body         string          `json:"body"`
		URL          string          `json:"url"`
		ImageURL     string          `json:"image_url"`
		CreatedAt    interface{}     `json:"created_at"`
		ShortSummary string          `json:"short_summary"`
		FullSummary  string          `json:"full_summary"`
		RawJSON      string          `json:"raw_json"`
		Embedding    pgvector.Vector `json:"embedding"`
		Similarity   float64         `json:"similarity"`
	}

	err := query.Select(fields...).Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("search scan: %w", err)
	}

	result := make([]*types.DecoratedActivity, len(rows))
	for i, a := range rows {
		act, err := activities.NewActivity(a.SourceType)
		if err != nil {
			return nil, fmt.Errorf("new activity: %w", err)
		}
		err = act.UnmarshalJSON([]byte(a.RawJSON))
		if err != nil {
			return nil, fmt.Errorf("unmarshal activity: %w", err)
		}
		result[i] = &types.DecoratedActivity{
			Activity: act,
			Summary: &types.ActivitySummary{
				ShortSummary: a.ShortSummary,
				FullSummary:  a.FullSummary,
			},
			Embedding:  a.Embedding.Slice(),
			Similarity: float32(a.Similarity),
		}
	}

	return result, nil
}

func activityFromEnt(in *ent.Activity) (*types.DecoratedActivity, error) {
	act, err := activities.NewActivity(in.SourceType)
	if err != nil {
		return nil, fmt.Errorf("new activity: %w", err)
	}

	err = act.UnmarshalJSON([]byte(in.RawJSON))
	if err != nil {
		return nil, fmt.Errorf("unmarshal activity: %w", err)
	}

	return &types.DecoratedActivity{
		Activity: act,
		Summary: &types.ActivitySummary{
			ShortSummary: in.ShortSummary,
			FullSummary:  in.FullSummary,
		},
	}, nil
}
