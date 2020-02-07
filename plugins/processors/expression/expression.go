package expression

import (
	"log"

	"github.com/Knetic/govaluate"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

type Expression struct {
	Name       string   `toml:"name"`
	Eval       string   `toml:"eval"`
	Parameters []string `toml:"parameters"`
}

var SampleConfig = `
  ### The name of the output metric
  name = "scaled_cpu"
  
  ### The expression to evaluate
  eval = "metrics.cpu * tags.scalefactor"
  
  ### Parameters that will be used in the expression
  parameters = [
  	"metrics.cpu",
  	"tags.scalefactor",
  ]
`

func (p *Expression) SampleConfig() string {
	return SampleConfig
}

func (p *Expression) Description() string {
	return "Evaluate an expression containing multiple tags or metrics and produce a new metric based on it"
}

func (p *Expression) Apply(metrics ...telegraf.Metric) []telegraf.Metric {
	expression, err := govaluate.NewEvaluableExpression(p.Eval)
	if err != nil {
		log.Printf("E! [processors.expression] could not construct expression: %v", err)
		return metrics
	}
	parameters := make(map[string]interface{}, 8)

	for _, metric := range metrics {
		for _, key := range p.Parameters {
			for _, field := range metric.FieldList() {
				if field.Key == key {

				}
			}
		}
	}
}

func init() {
	processors.Add("expression", func() telegraf.Processor {
		return &Expression{}
	})
}
