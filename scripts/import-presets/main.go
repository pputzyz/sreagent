// import-presets reads Prometheus/VMAlert rule YAML files from the monitoring-trading
// project and imports them as PresetRule records in the SREAgent database.
//
// Usage:
//
//	go run scripts/import-presets/main.go \
//	  --dir=/path/to/monitoring-trading/alerts \
//	  --dsn="user:pass@tcp(127.0.0.1:3306)/sreagent?parseTime=true"
//
// Severity mapping: P0→critical, P1→warning, P2→info, P3→info
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// PrometheusRuleFile represents a Prometheus/VMAlert rule file.
type PrometheusRuleFile struct {
	Groups []struct {
		Name  string `yaml:"name"`
		Rules []struct {
			Alert       string            `yaml:"alert"`
			Expr        string            `yaml:"expr"`
			For         string            `yaml:"for"`
			Labels      map[string]string `yaml:"labels"`
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"rules"`
	} `yaml:"groups"`
}

// PresetRule is the DB model matching internal/model/preset_rule.go.
type PresetRule struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:200;not null;index"`
	DisplayName string `gorm:"size:200"`
	Category    string `gorm:"size:50;index"`
	SubCategory string `gorm:"size:50"`
	Component   string `gorm:"size:50"`
	Expression  string `gorm:"type:text;not null"`
	ForDuration string `gorm:"size:32"`
	Severity    string `gorm:"size:20;index"`
	AlertType   string `gorm:"size:50"`
	Labels      string `gorm:"type:json"`
	Annotations string `gorm:"type:json"`
	Source      string `gorm:"size:100"`
	IsBuiltin   bool   `gorm:"default:true"`
	UsageCount  int    `gorm:"default:0"`
	Description string `gorm:"type:text"`
}

func (PresetRule) TableName() string { return "preset_rules" }

func main() {
	dir := flag.String("dir", "", "Path to monitoring-trading/alerts directory")
	dsn := flag.String("dsn", "", "MySQL DSN (user:pass@tcp(host:port)/db?parseTime=true)")
	dryRun := flag.Bool("dry-run", false, "Print rules without inserting to DB")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "--dir is required")
		os.Exit(1)
	}

	// Collect all YAML files
	var yamlFiles []string
	for _, category := range []string{"database", "kubernetes", "middleware", "node-exporter", "probe", "windows-exporter"} {
		catDir := filepath.Join(*dir, category)
		matches, err := filepath.Glob(filepath.Join(catDir, "*.yaml"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "glob %s: %v\n", catDir, err)
			continue
		}
		yamlFiles = append(yamlFiles, matches...)
	}

	fmt.Printf("Found %d YAML files\n", len(yamlFiles))

	var allRules []PresetRule
	for _, f := range yamlFiles {
		rules, err := parseFile(f, *dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", f, err)
			continue
		}
		allRules = append(allRules, rules...)
	}

	fmt.Printf("Parsed %d alert rules total\n", len(allRules))

	// Print summary by category
	catCount := map[string]int{}
	sevCount := map[string]int{}
	for _, r := range allRules {
		catCount[r.Category]++
		sevCount[r.Severity]++
	}
	fmt.Println("\nBy category:")
	for cat, n := range catCount {
		fmt.Printf("  %-20s %d\n", cat, n)
	}
	fmt.Println("\nBy severity:")
	for _, sev := range []string{"critical", "warning", "info"} {
		fmt.Printf("  %-20s %d\n", sev, sevCount[sev])
	}

	if *dryRun {
		fmt.Println("\n[dry-run] No database writes performed.")
		return
	}

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "--dsn is required (or use --dry-run)")
		os.Exit(1)
	}

	db, err := gorm.Open(mysql.Open(*dsn), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}

	// Upsert: skip duplicates by name
	inserted, skipped := 0, 0
	for _, rule := range allRules {
		var existing PresetRule
		if err := db.Where("name = ?", rule.Name).First(&existing).Error; err == nil {
			skipped++
			continue
		}
		if err := db.Create(&rule).Error; err != nil {
			fmt.Fprintf(os.Stderr, "insert %s: %v\n", rule.Name, err)
			continue
		}
		inserted++
	}

	fmt.Printf("\nDone: %d inserted, %d skipped (already exist)\n", inserted, skipped)
}

func parseFile(path, baseDir string) ([]PresetRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ruleFile PrometheusRuleFile
	if err := yaml.Unmarshal(data, &ruleFile); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	// Derive category and component from file path
	rel, _ := filepath.Rel(baseDir, path)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	category := "other"
	component := strings.TrimSuffix(parts[len(parts)-1], ".yaml")
	if len(parts) >= 2 {
		category = parts[0]
	}
	subCategory := component

	var rules []PresetRule
	for _, group := range ruleFile.Groups {
		for _, rule := range group.Rules {
			if rule.Alert == "" {
				continue // skip recording rules
			}

			severity := mapSeverity(rule.Labels["severity"])
			alertType := rule.Labels["alert_type"]
			if alertType == "" {
				alertType = "threshold"
			}

			// Build labels JSON (exclude severity and alert_type which have dedicated columns)
			labelMap := map[string]string{}
			for k, v := range rule.Labels {
				if k != "severity" && k != "alert_type" {
					labelMap[k] = v
				}
			}
			labelMap["category"] = category
			labelsJSON := mapToJSON(labelMap)

			// Build annotations JSON
			annotationsJSON := mapToJSON(rule.Annotations)

			// Description from annotations
			desc := ""
			if d, ok := rule.Annotations["description"]; ok {
				desc = d
			} else if s, ok := rule.Annotations["summary"]; ok {
				desc = s
			}

			displayName := rule.Alert
			if s, ok := rule.Annotations["summary"]; ok && s != "" {
				displayName = s
			}

			rules = append(rules, PresetRule{
				Name:        rule.Alert,
				DisplayName: displayName,
				Category:    category,
				SubCategory: subCategory,
				Component:   component,
				Expression:  strings.TrimSpace(rule.Expr),
				ForDuration: rule.For,
				Severity:    severity,
				AlertType:   alertType,
				Labels:      labelsJSON,
				Annotations: annotationsJSON,
				Source:      "monitoring-trading",
				IsBuiltin:   true,
				Description: desc,
			})
		}
	}
	return rules, nil
}

func mapSeverity(s string) string {
	switch strings.ToUpper(s) {
	case "P0":
		return "critical"
	case "P1":
		return "warning"
	case "P2", "P3":
		return "info"
	default:
		if s == "" {
			return "warning"
		}
		return strings.ToLower(s)
	}
}

func mapToJSON(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%q:%q", k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
