package postgres

import (
	"context"
	"time"

	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/glanceapp/glance/pkg/storage/postgres/ent"
	"github.com/glanceapp/glance/pkg/storage/postgres/ent/activity"
)

type ActivityRepository struct {
	db *DB
}

func NewActivityRepository(db *DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) Add(activity common.DecoratedActivity) error {
	ctx := context.Background()

	_, err := r.db.Client().Activity.Create().
		SetID(activity.UID()).
		SetUID(activity.UID()).
		SetSourceUID(activity.SourceUID()).
		SetTitle(activity.Title()).
		SetBody(activity.Body()).
		SetURL(activity.URL()).
		SetImageURL(activity.ImageURL()).
		SetCreatedAt(activity.CreatedAt()).
		SetShortSummary(activity.Summary.ShortSummary).
		SetFullSummary(activity.Summary.FullSummary).
		Save(ctx)

	return err
}

func (r *ActivityRepository) Remove(uid string) error {
	ctx := context.Background()
	return r.db.Client().Activity.DeleteOneID(uid).Exec(ctx)
}

func (r *ActivityRepository) List() ([]common.DecoratedActivity, error) {
	ctx := context.Background()

	activities, err := r.db.Client().Activity.Query().
		Order(ent.Desc(activity.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]common.DecoratedActivity, len(activities))
	for i, a := range activities {
		result[i] = common.DecoratedActivity{
			Activity: &activityImpl{
				uid:       a.UID,
				sourceUID: a.SourceUID,
				title:     a.Title,
				body:      a.Body,
				url:       a.URL,
				imageURL:  a.ImageURL,
				createdAt: a.CreatedAt,
			},
			Summary: &common.ActivitySummary{
				ShortSummary: a.ShortSummary,
				FullSummary:  a.FullSummary,
			},
		}
	}

	return result, nil
}

type activityImpl struct {
	uid       string
	sourceUID string
	title     string
	body      string
	url       string
	imageURL  string
	createdAt time.Time
}

func (a *activityImpl) UID() string          { return a.uid }
func (a *activityImpl) SourceUID() string    { return a.sourceUID }
func (a *activityImpl) Title() string        { return a.title }
func (a *activityImpl) Body() string         { return a.body }
func (a *activityImpl) URL() string          { return a.url }
func (a *activityImpl) ImageURL() string     { return a.imageURL }
func (a *activityImpl) CreatedAt() time.Time { return a.createdAt }
