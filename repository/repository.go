package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/osamikoyo/recomendation-service/entity/content"
	"github.com/osamikoyo/recomendation-service/logger"
	"go.uber.org/zap"
)

var (
	ErrNotFound = errors.New("content not found")
)

type Repository struct {
	driver neo4j.DriverWithContext
	logger *logger.Logger
}

func NewRepository(driver neo4j.DriverWithContext, logger *logger.Logger) *Repository {
	return &Repository{
		driver: driver,
		logger: logger,
	}
}

func (r *Repository) CreateContent(ctx context.Context, c *content.Content, lastRID string) error {
	if c == nil || c.RIndentifer == "" {
		return errors.New("rIdentifier is required")
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := "CREATE (n:Content {rIdentifier: $rIdentifier})"

		result, err := tx.Run(ctx, query, map[string]any{
			"rIndentifer": c.RIndentifer,
		})
		if err != nil {
			r.logger.Error("failed create content",
				zap.String("rid", c.RIndentifer),
				zap.Error(err))

			return nil, fmt.Errorf("failed create content: %w", err)
		}

		_, err = result.Single(ctx)
		if err != nil {
			r.logger.Error("node creating did not returt result",
				zap.Error(err))

			return nil, fmt.Errorf("node creation did not return result: %w", err)
		}

		return nil, nil
	})

	return err
}

func (r *Repository) CheckConnectionExist(ctx context.Context, rid1, rid2 string) (bool, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (a:Content {rIdentifier: $rid1})-[r:NEXT]->(b:Content {rIdentifier: $rid2})
			RETURN r IS NOT NULL AS exists
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"rid1": rid1,
			"rid2": rid2,
		})
		if err != nil {
			r.logger.Error("failed check connection exist",
				zap.String("rid1", rid1),
				zap.String("rid2", rid2),
				zap.Error(err))

			return false, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return false, nil
		}

		exists, _ := record.AsMap()["exists"].(bool)
		return exists, nil
	})

	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

func (r *Repository) UpdateConnectionValue(ctx context.Context, rid1, rid2 string, value int) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (a:Content {rIdentifier: $rid1})-[r:NEXT]->(b:Content {rIdentifier: $rid2})
			SET r.value = $value
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"rid1":  rid1,
			"rid2":  rid2,
			"value": value,
		})
		if err != nil {
			r.logger.Error("failed update connection value",
				zap.String("ri1d", rid1),
				zap.String("rid2", rid2),
				zap.Int("value", value),
				zap.Error(err))

			return nil, err
		}

		return nil, nil
	})

	return err
}

func (r *Repository) GetConnectionValue(ctx context.Context, rid1, rid2 string) (int, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (a:Content {rIdentifier: $rid1})-[r:NEXT]->(b:Content {rIdentifier: $rid2})
			RETURN r.value AS value
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"rid1": rid1,
			"rid2": rid2,
		})
		if err != nil {
			return 0, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return 0, ErrNotFound
		}

		val, _ := record.AsMap()["value"].(int64)
		return int(val), nil
	})

	if err != nil {
		return 0, err
	}
	return result.(int), nil
}

func (r *Repository) GetNextConnections(ctx context.Context, rid string) ([]string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (c:Content {rIdentifier: $rid})-[:NEXT]->(next:Content)
			RETURN collect(next.rIdentifier) AS rids
		`
		res, err := tx.Run(ctx, query, map[string]any{"rid": rid})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return []string{}, nil
		}

		ridsAny := record.AsMap()["rids"]
		rids, ok := ridsAny.([]any)
		if !ok {
			return []string{}, nil
		}

		result := make([]string, 0, len(rids))
		for _, v := range rids {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

func (r *Repository) CreateConnection(ctx context.Context, fromRID, toRID string) error {
	if fromRID == "" || toRID == "" {
		return errors.New("both fromRID and toRID are required")
	}
	if fromRID == toRID {
		return errors.New("self-loops are not allowed")
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (from:Content {rIdentifier: $fromRID})
			MATCH (to:Content {rIdentifier: $toRID})
			MERGE (from)-[rel:NEXT]->(to)
			ON CREATE SET rel.value = 1
			ON MATCH  SET rel.value = coalesce(rel.value, 0) + 1
			RETURN rel.value AS newValue
		`

		result, err := tx.Run(ctx, query, map[string]any{
			"fromRID": fromRID,
			"toRID":   toRID,
		})
		if err != nil {
			return nil, err
		}

		_, err = result.Single(ctx)
		if err != nil {
			return nil, err
		}

		return nil, nil
	})

	return err
}

func (r *Repository) CheckContentExistsByRID(ctx context.Context, rid string) (bool, error) {
	if rid == "" {
		return false, errors.New("rid is required")
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (c:Content {rIdentifier: $rid})
			RETURN count(c) > 0 AS exists
		`

		res, err := tx.Run(ctx, query, map[string]any{
			"rid": rid,
		})
		if err != nil {
			return false, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return false, err
		}

		exists, _ := record.AsMap()["exists"].(bool)
		return exists, nil
	})

	if err != nil {
		return false, err
	}

	return result.(bool), nil
}
