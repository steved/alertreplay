package rule

import (
	"fmt"
	"os"

	v1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"
)

// ParseAlertRule parses a VMRule file and returns the alert rule with the given name.
func ParseAlertRule(filePath string, alertName string) (*rulefmt.Rule, error) {
	groups, err := parseVMRuleFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed parsing VMRule file: %w", err)
	}

	rule, err := findVMAlertRule(groups, alertName)
	if err != nil {
		return nil, fmt.Errorf("failed to find alert rule: %w", err)
	}

	return rule, nil
}

func parseVMRuleFile(filePath string) ([]v1beta1.RuleGroup, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading file: %w", err)
	}

	var vmRule v1beta1.VMRule
	if err := yaml.Unmarshal(data, &vmRule); err != nil {
		return nil, fmt.Errorf("failed unmarshaling YAML: %w", err)
	}

	return vmRule.Spec.Groups, nil
}

func findVMAlertRule(groups []v1beta1.RuleGroup, alertName string) (*rulefmt.Rule, error) {
	for _, group := range groups {
		for _, r := range group.Rules {
			if r.Alert == alertName {
				var forDur model.Duration
				if r.For != "" {
					parsed, err := model.ParseDuration(r.For)
					if err != nil {
						return nil, fmt.Errorf("failed parsing duration %q: %w", r.For, err)
					}
					forDur = parsed
				}

				return &rulefmt.Rule{
					Alert:       r.Alert,
					Expr:        r.Expr,
					For:         forDur,
					Labels:      r.Labels,
					Annotations: r.Annotations,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("alert %q not found", alertName)
}
