package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/glanceapp/glance/pkg/sources"
	"github.com/glanceapp/glance/pkg/storage/postgres/ent"
	"github.com/glanceapp/glance/pkg/storage/postgres/ent/source"
)

type SourceRepository struct {
	db *DB
}

func NewSourceRepository(db *DB) *SourceRepository {
	return &SourceRepository{db: db}
}

func (r *SourceRepository) Add(s sources.Source) error {
	ctx := context.Background()

	configBytes, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal source config: %w", err)
	}

	_, err = r.db.Client().Source.Create().
		SetID(s.UID()).
		SetName(s.Name()).
		SetURL(s.URL()).
		SetType(s.Type()).
		SetConfigJSON(string(configBytes)).
		Save(ctx)

	return err
}

func (r *SourceRepository) Remove(uid string) error {
	ctx := context.Background()
	return r.db.Client().Source.DeleteOneID(uid).Exec(ctx)
}

func (r *SourceRepository) List() ([]sources.Source, error) {
	ctx := context.Background()

	sourcesEnt, err := r.db.Client().Source.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]sources.Source, len(sourcesEnt))
	for i, s := range sourcesEnt {
		configStr := ""
		if s.ConfigJSON != nil {
			configStr = *s.ConfigJSON
		}
		src, err := sourceFromEnt(s.Type, configStr)
		if err != nil {
			return nil, fmt.Errorf("deserialize source: %w", err)
		}
		result[i] = src
	}

	return result, nil
}

func (r *SourceRepository) GetByID(uid string) (sources.Source, error) {
	ctx := context.Background()

	s, err := r.db.Client().Source.Query().Where(source.ID(uid)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	configStr := ""
	if s.ConfigJSON != nil {
		configStr = *s.ConfigJSON
	}
	return sourceFromEnt(s.Type, configStr)
}

func sourceFromEnt(typeName, config string) (sources.Source, error) {
	src, err := sources.NewSource(typeName)
	if err != nil {
		return nil, fmt.Errorf("new source: %w", err)
	}
	if err := json.Unmarshal([]byte(config), src); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return src, nil
}
