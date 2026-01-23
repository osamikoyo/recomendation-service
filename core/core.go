package core

import (
	"context"
	"time"

	"github.com/osamikoyo/recomendation-service/entity/content"
)

type Repository interface {
	CreateContent(ctx context.Context, content *content.Content) error
	CheckConnectionExist(ctx context.Context, rid1, rid2 string) (bool, error)
	UpdateConnectionValue(ctx context.Context, rid1, rid2 string, value int) error
	GetConnectionValue(rid1, rid2 string) (int, error)
	GetNextConnections(rid string) ([]string, error)
}

type Core struct {
	repo Repository

	timeout time.Duration
}

func (c *Core) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

func (c *Core) GetBestOneForRID(rID string) (string, error) {
	conns, err := c.repo.GetNextConnections(rID)
	if err != nil {
		return "", err
	}

	if len(conns) == 0 {
		return rID, nil
	}
	
	if len(conns) == 1{
		return conns[0], nil
	}

	bvalue := 0
	bid := ""

	for _, id2 := range conns {
		value, err := c.repo.GetConnectionValue(rID, id2)
		if err != nil {
			return "", err
		}

		if value > bvalue {
			bid = id2
		}
	}

	return bid, nil
}

func (c *Core) GetOrderedRecsForRID(rID string) ([]*content.Content, error) {

}

func (c *Core) RouteAction(lastID, rID string) error {
	ctx, cancel := c.context()
	defer cancel()

	exist, err := c.repo.CheckConnectionExist(ctx, lastID, rID)
	if err != nil {
		return err
	}

	if exist {
		value, err := c.repo.GetConnectionValue(lastID, rID)
		if err != nil {
			return err
		}

		value++
		if err = c.repo.UpdateConnectionValue(ctx, lastID, rID, value); err != nil {
			return err
		}

		return nil
	} else {
		content := content.NewContent(lastID, rID)

		if err = c.repo.CreateContent(ctx, content); err != nil {
			return err
		}

		return nil
	}
}
