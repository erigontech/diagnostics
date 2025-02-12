package erigon_node

import (
	"context"
)

func (c *NodeClient) Subscribe(ctx context.Context, channel chan []byte, service string) error {
	request, err := c.fetch(ctx, "subscribe/"+service, nil)

	if err != nil {
		return err
	}

	for {
		more, result, err := request.nextResult(ctx)

		if err != nil {
			return err
		}

		channel <- result

		if !more {
			break
		}
	}

	return nil
}

func (c *NodeClient) Unsubscribe(ctx context.Context, channel chan []byte, service string) error {
	request, err := c.fetch(ctx, "unsubscribe/"+service, nil)

	if err != nil {
		return err
	}

	for {
		more, result, err := request.nextResult(ctx)

		if err != nil {
			return err
		}

		channel <- result

		if !more {
			break
		}
	}

	return nil
}
