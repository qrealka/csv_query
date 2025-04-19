package aggregator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	api "propertytreeanalyzer/pkg/api/aggregator"
	apiAttr "propertytreeanalyzer/pkg/api/attribute"
	apiGroupify "propertytreeanalyzer/pkg/api/groupify"
	num "propertytreeanalyzer/pkg/numeric"

	"github.com/cockroachdb/apd/v3"
	"golang.org/x/sync/errgroup"
)

const (
	groupsQueueSize = 1000
	priceQueueSize  = 10000
)

type avgByGroup struct {
	key apiAttr.BaseAttribute
	val apiAttr.NumericAttribute
}

func (a avgByGroup) GroupKey() apiAttr.BaseAttribute        { return a.key }
func (a avgByGroup) AverageValue() apiAttr.NumericAttribute { return a.val }

type stringAttr string

func (s stringAttr) String() string { return string(s) }

var (
	_ apiAttr.BaseAttribute  = (*stringAttr)(nil)
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
func averagePrice(ctx context.Context, in <-chan apiAttr.NumericAttribute) (apiAttr.NumericAttribute, error) {
	numType := apiAttr.Nothing
	sumDec := apd.New(0, 0)
	sumFloat := float64(0)
	cnt := int64(0)
	done := ctx.Done()

	for val := range in {
		select {
		case <-done:
			return nil, ctx.Err()
		default:
		}

		if numType == apiAttr.Nothing {
			// we initialize numeric type only once. I do notexpect that we could have few numeric types in one group
			numType = val.GetNumericType()
		}

		if numType == apiAttr.Decimal {
			if decVal, err := num.CastToDecimalAttribute(val); err != nil {
				slog.ErrorContext(ctx, "Error casting to decimal", "error", err)
				return nil, err
			} else {
				price := decVal.GetDecimal()
				if _, err := sumCtx.Add(sumDec, sumDec, price); err != nil {
					slog.ErrorContext(ctx, "Error adding price to sum", "price", price.String(), "error", err)
					return nil, err
				}
			}
		} else if numType == apiAttr.Float {
			if floatVal, err := num.CastToFloatAttribute(val); err != nil {
				slog.ErrorContext(ctx, "Error casting to float", "error", err)
				return nil, err
			} else {
				price := floatVal.GetFloat()
				sumFloat += price
			}
		} else {
			slog.ErrorContext(ctx, "Unknown numeric type", "type", numType)
			return nil, fmt.Errorf("unknown numeric type: %s", val)
		}
		cnt++
	}

	// calculate average
	if numType == apiAttr.Decimal {
		avg := apd.New(0, 0)
		if _, err := avgCtx.Quo(avg, sumDec, apd.New(cnt, 0)); err != nil {
			slog.ErrorContext(ctx, "Error calculating average", "error", err)
			return nil, err
		}
		if _, err := avgCtx.Quantize(avg, avg, -2); err != nil {
			slog.ErrorContext(ctx, "Error quantizing average", "error", err)
			return nil, err
		}
		return num.NewDecimalAttribute(avg), nil
	} else if numType == apiAttr.Float {
		avg := sumFloat / float64(cnt)
		return num.NewFloatAttribute(avg), nil
	}

	// nothing to average
	return num.NoneNumeric, nil

}

// Process implements aggregators.AvgerageAggregator.
func (a *avgPriceBy) Process(ctx context.Context, streets <-chan apiAttr.StreetAttribute) (outputs []api.AverageByGroup, resultErr error) {
	// I use regular map here because the number of groups is immutable in the Process method
	// So I precreate and fill maps
	prices := make(map[string]chan apiAttr.NumericAttribute) // parallel calculation AVG price per groups
	// here is the biggest storage complexity, but I do not expect to have more than 100K streets
	// for golang 1.24 it is swiss table and for my microbenchmarks it works faster than existing Patricia tree in Go
	streetToSize := make(map[string]string) // joining street names with group ids
	done := ctx.Done()

	type result struct {
		avg apiAttr.NumericAttribute
		err error
	}

	// groupID → result{Avg, Err}
	var results sync.Map

	// prefill maps from JSON stream
	for item := range a.groups {
		groupId := item.Key().String()
		streetToSize[item.StreetName().String()] = groupId
		if _, ok := prices[groupId]; !ok {
			prices[groupId] = make(chan apiAttr.NumericAttribute, priceQueueSize)
			results.Store(groupId, result{avg: num.NoneNumeric, err: nil})
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

	resultErr = nil
	results.Range(func(key, value any) bool {
		groupId := key.(string)
		res := value.(result)
		if res.err != nil {
			resultErr = res.err
			return false
		}
		outputs = append(outputs, avgByGroup{
			key: stringAttr(groupId),
			val: res.avg,
		})
		return true
	})
	return outputs, resultErr
}
