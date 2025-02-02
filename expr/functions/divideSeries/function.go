package divideSeries

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/go-graphite/carbonapi/expr/helper"
	"github.com/go-graphite/carbonapi/expr/interfaces"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
)

type divideSeries struct {
	interfaces.FunctionBase
}

func GetOrder() interfaces.Order {
	return interfaces.Any
}

func New(configFile string) []interfaces.FunctionMetadata {
	res := make([]interfaces.FunctionMetadata, 0)
	f := &divideSeries{}
	functions := []string{"divideSeries"}
	for _, n := range functions {
		res = append(res, interfaces.FunctionMetadata{Name: n, F: f})
	}
	return res
}

// divideSeries(dividendSeriesList, divisorSeriesList)
func (f *divideSeries) Do(ctx context.Context, e parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) ([]*types.MetricData, error) {
	if len(e.Args()) < 1 {
		return nil, parser.ErrMissingTimeseries
	}

	firstArg, err := helper.GetSeriesArg(ctx, e.Args()[0], from, until, values)
	if err != nil {
		return nil, err
	}

	var useMetricNames bool

	var numerators []*types.MetricData
	var denominator *types.MetricData
	if len(e.Args()) == 2 {
		useMetricNames = true
		numerators = firstArg
		denominators, err := helper.GetSeriesArg(ctx, e.Args()[1], from, until, values)
		if err != nil {
			return nil, err
		}
		if len(denominators) != 1 {
			return nil, types.ErrWildcardNotAllowed
		}

		denominator = denominators[0]
	} else if len(firstArg) == 2 && len(e.Args()) == 1 {
		numerators = append(numerators, firstArg[0])
		denominator = firstArg[1]
	} else {
		return nil, errors.New("must be called with 2 series or a wildcard that matches exactly 2 series")
	}

	alignedSeries := helper.AlignSeries(append(numerators, denominator))
	numerators = alignedSeries[:len(numerators)]
	denominator = alignedSeries[len(numerators)]

	for _, numerator := range numerators {
		if numerator.StepTime != denominator.StepTime {
			return nil, fmt.Errorf("series %s must have the same length as %s", numerator.Name, denominator.Name)
		}
	}

	var results []*types.MetricData
	for _, numerator := range numerators {
		r := *numerator
		if useMetricNames {
			r.Name = fmt.Sprintf("divideSeries(%s,%s)", numerator.Name, denominator.Name)
		} else {
			r.Name = fmt.Sprintf("divideSeries(%s)", e.RawArgs())
		}
		r.Values = make([]float64, len(numerator.Values))

		for i, v := range numerator.Values {

			// math.IsNaN(v) || math.IsNaN(denominator.Values[i]) covered by nature of math.NaN
			if denominator.Values[i] == 0 {
				r.Values[i] = math.NaN()
				continue
			}

			r.Values[i] = v / denominator.Values[i]
		}
		results = append(results, &r)
	}

	return results, nil

}

// Description is auto-generated description, based on output of https://github.com/graphite-project/graphite-web
func (f *divideSeries) Description() map[string]types.FunctionDescription {
	return map[string]types.FunctionDescription{
		"divideSeries": {
			Description: "Takes a dividend metric and a divisor metric and draws the division result.\nA constant may *not* be passed. To divide by a constant, use the scale()\nfunction (which is essentially a multiplication operation) and use the inverse\nof the dividend. (Division by 8 = multiplication by 1/8 or 0.125)\n\nExample:\n\n.. code-block:: none\n\n  &target=divideSeries(Series.dividends,Series.divisors)",
			Function:    "divideSeries(dividendSeriesList, divisorSeries)",
			Group:       "Combine",
			Module:      "graphite.render.functions",
			Name:        "divideSeries",
			Params: []types.FunctionParam{
				{
					Name:     "dividendSeriesList",
					Required: true,
					Type:     types.SeriesList,
				},
				{
					Name:     "divisorSeries",
					Required: true,
					Type:     types.SeriesList,
				},
			},
		},
	}
}
