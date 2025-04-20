package aggregator

import (
	"context"
	"log/slog"
	"sync"

	api "propertytreeanalyzer/pkg/api/aggregator"
	apiAttr "propertytreeanalyzer/pkg/api/attribute"
	apiGroupify "propertytreeanalyzer/pkg/api/groupify"

	"github.com/cockroachdb/apd/v3"
	"golang.org/x/sync/errgroup"
)

const (
	groupsQueueSize = 1000
	priceQueueSize  = 10000
)

type avgByGroup struct {
	key string
	val string
}

func (a avgByGroup) GroupKey() string     { return a.key }
func (a avgByGroup) AverageValue() string { return a.val }

var (
	_ api.AverageByGroup     = (*avgByGroup)(nil)
	_ api.AvgerageAggregator = (*avgPriceBy)(nil)

	sumCtx apd.Context = apd.Context{
		Precision:   100,
		MaxExponent: apd.MaxExponent,
		MinExponent: apd.MinExponent,
		Traps:       apd.DefaultTraps,
		Rounding:    apd.RoundHalfEven,
	}

	avgCtx apd.Context = apd.Context{
		Precision:   50,
		MaxExponent: apd.MaxExponent,
		MinExponent: apd.MinExponent,
		Traps:       apd.DefaultTraps,
		Rounding:    apd.RoundHalfEven, // Banker's rounding for final result
	}
)

type avgPriceBy struct {
	groups <-chan apiGroupify.StreetGroupItem
}

func NewAvgPriceBy(groups <-chan apiGroupify.StreetGroupItem) api.AvgerageAggregator {
	return &avgPriceBy{
		groups: groups,
	}
}

// averagePrice calculates the average price of a group of attributes
func averagePrice(ctx context.Context, in <-chan string) (string, error) {
	sumDec := apd.New(0, 0)
	valDec := apd.New(0, 0)
	cnt := int64(0)
	done := ctx.Done()

	for val := range in {
		select {
		case <-done:
			return "", ctx.Err()
		default:
		}

		// minimize memory allocations
		if _, c, err := valDec.SetString(val); err != nil {
			slog.ErrorContext(ctx, "Error parsing value", "value", val, "condition", c, "error", err)
			return "", err
		} else {
			if _, err := sumCtx.Add(sumDec, sumDec, valDec); err != nil {
				slog.ErrorContext(ctx, "Error adding price to sum", "price", valDec.String(), "error", err)
				return "", err
			}
		}

		cnt++
	}

	// calculate average
	valDec.SetInt64(0)
	if _, err := avgCtx.Quo(valDec, sumDec, apd.New(cnt, 0)); err != nil {
		slog.ErrorContext(ctx, "Error calculating average", "error", err)
		return "", err
	}
	if _, err := avgCtx.Quantize(valDec, valDec, -2); err != nil {
		slog.ErrorContext(ctx, "Error quantizing average", "error", err)
		return "", err
	}
	return valDec.String(), nil
}

// Process implements aggregators.AvgerageAggregator.
func (a *avgPriceBy) Process(ctx context.Context, streets <-chan apiAttr.StreetAttribute) (outputs []api.AverageByGroup, resultErr error) {
	// I use regular map here because the number of groups is immutable in the Process method
	// So I precreate and fill maps
	prices := make(map[string]chan string) // parallel calculation AVG price per groups
	// here is the biggest storage complexity, but I do not expect to have more than 100K streets
	// for golang 1.24 it is swiss table and for my microbenchmarks it works faster than existing Patricia tree in Go
	streetToSize := make(map[string]string) // joining street names with group ids
	done := ctx.Done()

	type result struct {
		avg string
		err error
	}

	var (
		results sync.Map
		order   []string
	)

	// prefill maps from JSON stream
	for item := range a.groups {
		groupId := item.Key().String()
		streetToSize[item.StreetName().String()] = groupId
		if _, ok := prices[groupId]; !ok {
			prices[groupId] = make(chan string, priceQueueSize)
			results.Store(groupId, result{avg: "", err: nil})
			order = append(order, groupId)
		}
	}

	// spawn workers under errgroup
	eg, ctx := errgroup.WithContext(ctx)
	for groupId, ch := range prices {
		groupId, ch := groupId, ch
		eg.Go(func() error {
			avgVal, err := averagePrice(ctx, ch)
			results.Store(groupId, result{avg: avgVal, err: err})
			return err
		})
	}

	// feed price channels
	go func() {
		defer func() {
			// close() is a cheap operation
			for _, ch := range prices {
				close(ch)
			}
		}()
		for street := range streets {
			select {
			case <-done:
				return
			default:
			}
			groupID, ok := streetToSize[street.StreetName()]
			if !ok {
				continue
			}
			prices[groupID] <- street.AttributeValue()
		}
	}()

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// build outputs in recorded order
	for _, id := range order {
		if v, found := results.Load(id); found {
			r := v.(result)
			if r.err != nil {
				return nil, r.err
			}
			outputs = append(outputs, avgByGroup{key: id, val: r.avg})
		}
	}
	return outputs, nil
}
