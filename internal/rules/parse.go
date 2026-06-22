package rules

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/steved/alertreplay/internal/prometheus"
)

func parseFile(filePath string) (*rulefmt.RuleGroups, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", filePath, err)
	}

	// This isn't the exact type, but rulefmt.RuleGroup is enough of a subset for us right now.
	// Importing the type directly causes go-mod conflicts for VictoriaMetrics unless the versions are exactly right..
	// https://github.com/VictoriaMetrics/operator/blob/v0.68.3/api/operator/v1beta1/vmrule_types.go
	var vmRule struct {
		Kind string              `yaml:"kind"`
		Spec *rulefmt.RuleGroups `yaml:"spec"`
	}

	if err := yaml.Unmarshal(content, &vmRule); err != nil {
		return nil, fmt.Errorf("unmarshaling YAML: %w", err)
	}

	if vmRule.Kind == "VMRule" {
		return vmRule.Spec, nil
	}

	groups, errs := rulefmt.Parse(
		content,
		false,
		model.UTF8Validation,
		prometheus.Parser,
		slog.New(zerolog.NewSlogHandler(zlog.Logger)),
	)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return groups, nil
}

func ParseAlertRule(filePath string, alertName string) (rulefmt.Rule, error) {
	groups, err := parseFile(filePath)
	if err != nil {
		return rulefmt.Rule{}, err
	}

	return findAlertRule(groups, alertName)
}

func findAlertRule(groups *rulefmt.RuleGroups, alertName string) (rulefmt.Rule, error) {
	for _, group := range groups.Groups {
		for _, r := range group.Rules {
			if r.Alert == alertName {
				return r, nil
			}
		}
	}

	return rulefmt.Rule{}, fmt.Errorf("alert %q not found", alertName)
}
