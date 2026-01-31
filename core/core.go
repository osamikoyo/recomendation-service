package core

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/osamikoyo/recomendation-service/entity/content"
)

type Repository interface {
	CreateContent(ctx context.Context, content *content.Content, lastID string) error
	CheckConnectionExist(ctx context.Context, rid1, rid2 string) (bool, error)
	UpdateConnectionValue(ctx context.Context, rid1, rid2 string, value int) error
	GetConnectionValue(ctx context.Context, rid1, rid2 string) (int, error)
	GetNextConnections(ctx context.Context, rid string) ([]string, error)
	CreateConnection(ctx context.Context, fromRID, toRID string) error
	CheckContentExistsByRID(ctx context.Context, rid string) (bool, error)
}

type Core struct {
	repo Repository

	timeout time.Duration
}

func NewCore(repo Repository, timeout time.Duration) *Core {
	return &Core{
		repo: repo,
		timeout: timeout,
	}
}

func (c *Core) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

func (c *Core) GetBestOneForRID(rID string) (string, error) {
	ctx, cancel := c.context()
	defer cancel()

	conns, err := c.repo.GetNextConnections(ctx, rID)
	if err != nil {
		return "", err
	}

	if len(conns) == 0 {
		return rID, nil
	}

	if len(conns) == 1 {
		return conns[0], nil
	}

	bvalue := 0
	bid := ""

	for _, id2 := range conns {
		value, err := c.repo.GetConnectionValue(ctx, rID, id2)
		if err != nil {
			return "", err
		}

		if value > bvalue {
			bid = id2
		}
	}

	return bid, nil
}

func (c *Core) GetOrderedRecsForRID(rID string, number ...int) ([]string, error) {
	ctx, cancel := c.context()
	defer cancel()

	conns, err := c.repo.GetNextConnections(ctx, rID)
	if err != nil {
		return nil, err
	}

	if len(conns) == 0 {
		return nil, fmt.Errorf("empty conns: %s", rID)
	}

	if len(conns) == 1 {
		return conns, nil
	}

	recs := make(Recs, len(conns))

	for i, id2 := range conns {
		value, err := c.repo.GetConnectionValue(ctx, rID, id2)
		if err != nil {
			return nil, err
		}

		recs[i] = rec{
			Value: value,
			Rid:   id2,
		}
	}

	sort.Sort(recs)

	var rids []string
	if len(number) == 0 {
		rids = make([]string, len(recs))
	} else {
		rids = make([]string, number[0])
	}

	for i, rec := range recs {
		rids[i] = rec.Rid
	}

	return rids, nil
}

func (c *Core) RouteAction(lastID, rID string) error {
	ctx, cancel := c.context()
	defer cancel()

	exist, err := c.repo.CheckConnectionExist(ctx, lastID, rID)
	if err != nil {
		return err
	}

	if exist {
		value, err := c.repo.GetConnectionValue(ctx, lastID, rID)
		if err != nil {
			return err
		}

		value++
		if err = c.repo.UpdateConnectionValue(ctx, lastID, rID, value); err != nil {
			return err
		}

		return nil
	} else {
		cexist, err := c.repo.CheckContentExistsByRID(ctx, rID)
		if err != nil {
			return err
		}

		if cexist {
			if err = c.repo.CreateConnection(ctx, lastID, rID); err != nil {
				return err
			}

			return nil
		} else {
			content := content.NewContent(rID)

			if err = c.repo.CreateContent(ctx, content, lastID); err != nil {
				return err
			}

			return nil
		}
	}
}
