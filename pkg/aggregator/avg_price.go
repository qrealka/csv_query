package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	apiAgg "propertytreeanalyzer/pkg/api/aggregator"
	apiAttr "propertytreeanalyzer/pkg/api/attribute"
	apiGroupify "propertytreeanalyzer/pkg/api/groupify"
	num "propertytreeanalyzer/pkg/numeric"

	"github.com/cockroachdb/apd/v3"
)

const (
	groupsQueueSize = 1000
	priceQueueSize  = 10000
)

// treeSizeAggregator implements StreetGroupAggregator
type treeSizeAggregator struct {
	grouper apiGroupify.StreetGroups
}

var (
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

// NewTreeSizeAggregator constructs one from a JsonStream
func NewTreeSizeAggregator(g apiGroupify.StreetGroups) apiAgg.StreetGroupAggregator {
	return &treeSizeAggregator{grouper: g}
}

// averagePrice calculates the average price of a group of attributes
func (a *treeSizeAggregator) averagePrice(in <-chan apiAttr.NumericAttribute) (apiAttr.NumericAttribute, error) {
	numType := apiAttr.Nothing
	sumDec := apd.New(0, 0)
	sumFloat := float64(0)
	cnt := int64(0)
	for val := range in {
		if numType == apiAttr.Nothing {
			// we initialize numeric type only once. I do notexpect that we could have few numeric types in one group
			numType = val.GetNumericType()
		}

		if numType == apiAttr.Decimal {
			if decVal, err := num.CastToDecimalAttribute(val); err != nil {
				slog.Error("Error casting to decimal", "error", err)
				return nil, err
			} else {
				price := decVal.GetDecimal()
				if _, err := sumCtx.Add(sumDec, sumDec, price); err != nil {
					slog.Error("Error adding price to sum", "price", price.String(), "error", err)
					return nil, err
				}
			}
		} else if numType == apiAttr.Float {
			if floatVal, err := num.CastToFloatAttribute(val); err != nil {
				slog.Error("Error casting to float", "error", err)
				return nil, err
			} else {
				price := floatVal.GetFloat()
				sumFloat += price
			}
		} else {
			slog.Error("Unknown numeric type", "type", numType)
			return nil, fmt.Errorf("unknown numeric type: %s", val)
		}
		cnt++
	}

	// calculate average
	if numType == apiAttr.Decimal {
		avg := apd.New(0, 0)
		if _, err := avgCtx.Quo(avg, sumDec, apd.New(cnt, 0)); err != nil {
			slog.Error("Error calculating average", "error", err)
			return nil, err
		}
		if _, err := avgCtx.Quantize(avg, avg, -2); err != nil {
			slog.Error("Error quantizing average", "error", err)
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

// Process reads StreetAttribute from 'attributes', groups them by tree‑size,
// and prints the average price per group.
func (a *treeSizeAggregator) Process(ctx context.Context, attributes <-chan apiAttr.StreetAttribute) error {
	// I use regular map here because the number of groups is immutable in the Process method
	// So I precreate and fill maps
	prices := make(map[string]chan apiAttr.NumericAttribute) // parallel calculation AVG price per groups
	streetToSize := make(map[string]string)                  // joining street names with group ids

	type result struct {
		avg apiAttr.NumericAttribute
		err error
	}

	// groupID → result{Avg, Err}
	results := make(map[string]result)

	// Load the street→TreeSize map
	groupCh := make(chan apiGroupify.StreetGroupItem, groupsQueueSize)
	go a.grouper.GroupStreets(ctx, groupCh)

	// prefill maps
	for item := range groupCh {
		groupId := item.Key().String()
		streetToSize[item.StreetName().String()] = groupId
		if _, ok := prices[groupId]; !ok {
			prices[groupId] = make(chan apiAttr.NumericAttribute, priceQueueSize)
			results[groupId] = result{avg: num.NoneNumeric, err: nil}
		}
	}

	// Now maps price, streetToSize and results are filled and immutable
	var wg sync.WaitGroup

	// Spawn the summarizer goroutines
	for group, ch := range prices {
		wg.Add(1)
		go func(groupId string, in <-chan apiAttr.NumericAttribute) {
			defer wg.Done()
			averagePrice, err := a.averagePrice(in)
			results[groupId] = result{avg: averagePrice, err: err}
		}(group, ch)
	}

	wg.Wait()

	// print as JSON for easy parsing
	type avgOutput struct {
		GroupID      string `json:"tree_size"`
		AveragePrice string `json:"average_price"`
	}

	// Compute and print all averages as a JSON array
	var outputs []avgOutput
	for groupId, res := range results {
		if res.err != nil {
			slog.Error("Error calculating average price", "groupId", groupId, "error", res.err)
			return res.err
		}
		outputs = append(outputs, avgOutput{
			GroupID:      groupId,
			AveragePrice: res.avg.String(),
		})
	}

	jsonData, err := json.Marshal(outputs)
	if err != nil {
		slog.Error("Error marshaling JSON array", "error", err)
		return err
	}

	fmt.Println(string(jsonData))
	return nil
}
