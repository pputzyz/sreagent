//go:build ignore

// import-preset-rules.go — Import alert rules from monitoring-trading YAML files
// into SREAgent's preset_rules database table.
//
// Usage:
//
//	go run scripts/import-preset-rules.go --dry-run          # Preview only
//	go run scripts/import-preset-rules.go                     # Import to DB
//	go run scripts/import-preset-rules.go --dir /path/to/alerts  # Custom dir
package main

import (
	"database/sql/driver"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------------------------------------------------------------------------
// YAML structures (Prometheus/VMAlert rule format)
// ---------------------------------------------------------------------------

type PrometheusRuleFile struct {
	Groups []PrometheusGroup `yaml:"groups"`
}

type PrometheusGroup struct {
	Name  string           `yaml:"name"`
	Rules []PrometheusRule `yaml:"rules"`
}

type PrometheusRule struct {
	Alert       string            `yaml:"alert"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// ---------------------------------------------------------------------------
// PresetRule model (mirrors internal/model/preset_rule.go)
// ---------------------------------------------------------------------------

type JSONLabels map[string]string

func (j JSONLabels) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}
	return string(b), nil
}

func (j *JSONLabels) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONLabels)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for JSONLabels: %T", value)
	}
	return json.Unmarshal(bytes, j)
}

type PresetRule struct {
	ID          uint        `gorm:"primaryKey;autoIncrement"`
	Name        string      `gorm:"size:200;not null;index"`
	DisplayName string      `gorm:"size:200"`
	Category    string      `gorm:"size:50;index"`
	SubCategory string      `gorm:"size:50"`
	Component   string      `gorm:"size:50"`
	Expression  string      `gorm:"type:text;not null"`
	ForDuration string      `gorm:"size:32"`
	Severity    string      `gorm:"size:20;index"`
	AlertType   string      `gorm:"size:50"`
	Labels      JSONLabels  `gorm:"type:json"`
	Annotations JSONLabels  `gorm:"type:json"`
	Source      string      `gorm:"size:100"`
	IsBuiltin   bool        `gorm:"default:true"`
	UsageCount  int         `gorm:"default:0"`
	Description string      `gorm:"type:text"`
	CreatedAt   time.Time   `gorm:"autoCreateTime"`
	UpdatedAt   time.Time   `gorm:"autoUpdateTime"`
}

func (PresetRule) TableName() string { return "preset_rules" }

// ---------------------------------------------------------------------------
// Category mapping from directory structure
// ---------------------------------------------------------------------------

var dirCategoryMap = map[string]struct {
	Category   string
	SubFromDir bool // true = subcategory from filename (without .yaml)
}{
	"node-exporter":    {Category: "infrastructure"},
	"kubernetes":       {Category: "kubernetes"},
	"middleware":       {Category: "middleware"},
	"database":         {Category: "database"},
	"probe":            {Category: "probe"},
	"windows-exporter": {Category: "windows"},
}

// componentFromFilename maps filename stems to human-readable component names.
var componentFromFilename = map[string]string{
	"cpu":                    "cpu",
	"disk":                   "disk",
	"filesystem":             "filesystem",
	"memory":                 "memory",
	"network":                "network",
	"system":                 "system",
	"container":              "container",
	"apiserver":              "apiserver",
	"coredns":                "coredns",
	"kube-controller-manager": "kube-controller-manager",
	"kube-etcd":              "kube-etcd",
	"kube-scheduler":         "kube-scheduler",
	"kube-state-metrics":     "kube-state-metrics",
	"kubelet":                "kubelet",
	"resource-reservation":   "resource-reservation",
	"etcd":                   "etcd",
	"kafka":                  "kafka",
	"nacos":                  "nacos",
	"rabbitmq":               "rabbitmq",
	"rocketmq":               "rocketmq",
	"clickhouse":             "clickhouse",
	"elasticsearch":          "elasticsearch",
	"mongodb":                "mongodb",
	"redis":                  "redis",
	"blackbox-http":          "blackbox-http",
	"blackbox-tcp":           "blackbox-tcp",
}

// severityMap maps P-level severity labels to normalized values.
var severityMap = map[string]string{
	"P0": "critical",
	"P1": "warning",
	"P2": "info",
	"P3": "info",
}

// ---------------------------------------------------------------------------
// Stats tracking
// ---------------------------------------------------------------------------

type CategoryStats struct {
	Total      int
	SubCats    map[string]int
}

type ImportStats struct {
	FileCount  int
	RuleCount  int
	SkipCount  int
	Categories map[string]*CategoryStats
	Severities map[string]int
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	var (
		dryRun bool
		dir    string
		dsn    string
	)

	flag.BoolVar(&dryRun, "dry-run", false, "Preview import without writing to database")
	flag.StringVar(&dir, "dir", "", "Alerts directory (default: ../monitoring-trading-main/monitoring-trading-main/alerts/)")
	flag.StringVar(&dsn, "dsn", "", "MySQL DSN (default: $SREAGENT_DATABASE_DSN or root:password@tcp(127.0.0.1:3306)/sreagent?parseTime=true)")
	flag.Parse()

	// Resolve alerts directory
	if dir == "" {
		dir = filepath.Join("..", "monitoring-trading-main", "monitoring-trading-main", "alerts")
	}
	dir = filepath.Clean(dir)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: alerts directory not found: %s\n", dir)
		os.Exit(1)
	}

	fmt.Println("=== Import Preset Rules ===")
	fmt.Printf("Source: %s\n", dir)
	if dryRun {
		fmt.Println("Mode:   dry-run")
	} else {
		fmt.Println("Mode:   import")
	}
	fmt.Println()

	// Walk directory and collect all rules
	rules, stats, err := collectRules(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting rules: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	printSummary(stats)

	if len(rules) == 0 {
		fmt.Println("No rules found. Nothing to import.")
		os.Exit(0)
	}

	if dryRun {
		fmt.Println("Use --confirm (or run without --dry-run) to proceed with import.")
		os.Exit(0)
	}

	// Resolve DSN
	if dsn == "" {
		dsn = os.Getenv("SREAGENT_DATABASE_DSN")
	}
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/sreagent?parseTime=true"
	}

	// Connect to database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	// Batch insert with upsert
	if err := batchUpsert(db, rules); err != nil {
		fmt.Fprintf(os.Stderr, "Error importing rules: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSuccessfully imported %d preset rules.\n", len(rules))
}

// ---------------------------------------------------------------------------
// collectRules walks the alerts directory and parses all YAML files.
// ---------------------------------------------------------------------------

func collectRules(dir string) ([]PresetRule, *ImportStats, error) {
	stats := &ImportStats{
		Categories: make(map[string]*CategoryStats),
		Severities: make(map[string]int),
	}

	var allRules []PresetRule

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".yaml") && !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		// Determine category and subcategory from relative path
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) < 2 {
			return nil // skip files at root level (e.g., templates/)
		}

		dirName := parts[0]
		fileName := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))

		catInfo, ok := dirCategoryMap[dirName]
		if !ok {
			// Skip unknown directories (e.g., templates/)
			fmt.Printf("  Skipping unknown directory: %s\n", dirName)
			return nil
		}

		category := catInfo.Category
		subCategory := fileName
		component := fileName
		if mapped, ok := componentFromFilename[fileName]; ok {
			component = mapped
		}

		// Parse YAML file
		rules, skipped, err := parseRuleFile(path, category, subCategory, component)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to parse %s: %v\n", relPath, err)
			return nil // continue walking
		}

		if len(rules) > 0 {
			stats.FileCount++
			stats.RuleCount += len(rules)
			stats.SkipCount += skipped

			// Update category stats
			catKey := category
			if stats.Categories[catKey] == nil {
				stats.Categories[catKey] = &CategoryStats{
					SubCats: make(map[string]int),
				}
			}
			stats.Categories[catKey].Total += len(rules)
			stats.Categories[catKey].SubCats[subCategory] += len(rules)

			// Update severity stats
			for _, r := range rules {
				stats.Severities[r.Severity]++
			}

			allRules = append(allRules, rules...)
		} else if skipped > 0 {
			stats.SkipCount += skipped
		}

		return nil
	})

	return allRules, stats, err
}

// ---------------------------------------------------------------------------
// parseRuleFile parses a single YAML file and returns PresetRule records.
// ---------------------------------------------------------------------------

func parseRuleFile(path, category, subCategory, component string) ([]PresetRule, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read file: %w", err)
	}

	var ruleFile PrometheusRuleFile
	if err := yaml.Unmarshal(data, &ruleFile); err != nil {
		return nil, 0, fmt.Errorf("parse yaml: %w", err)
	}

	var rules []PresetRule
	skipped := 0

	for _, group := range ruleFile.Groups {
		for _, rule := range group.Rules {
			// Skip rules without an alert name (empty/malformed)
			if rule.Alert == "" {
				skipped++
				continue
			}

			// Skip rules with empty expression
			rule.Expr = strings.TrimSpace(rule.Expr)
			if rule.Expr == "" {
				skipped++
				continue
			}

			preset := PresetRule{
				Name:        rule.Alert,
				DisplayName: getAnnotation(rule.Annotations, "summary"),
				Category:    category,
				SubCategory: subCategory,
				Component:   component,
				Expression:  rule.Expr,
				ForDuration: strings.TrimSpace(rule.For),
				Severity:    mapSeverity(rule.Labels["severity"]),
				AlertType:   getOrDefault(rule.Labels, "alert_type", "threshold"),
				Source:      "monitoring-trading",
				IsBuiltin:   true,
				UsageCount:  0,
				Description: getAnnotation(rule.Annotations, "description"),
			}

			// Build labels JSON (include all labels from the rule)
			preset.Labels = make(JSONLabels)
			for k, v := range rule.Labels {
				preset.Labels[k] = v
			}
			// Ensure category and component are in labels
			if _, ok := preset.Labels["category"]; !ok {
				preset.Labels["category"] = category
			}
			if _, ok := preset.Labels["component"]; !ok {
				preset.Labels["component"] = component
			}

			// Build annotations JSON
			preset.Annotations = make(JSONLabels)
			for k, v := range rule.Annotations {
				preset.Annotations[k] = v
			}

			rules = append(rules, preset)
		}
	}

	return rules, skipped, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mapSeverity(s string) string {
	if mapped, ok := severityMap[strings.ToUpper(strings.TrimSpace(s))]; ok {
		return mapped
	}
	return "warning" // default
}

func getOrDefault(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

func getAnnotation(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[key])
}

// ---------------------------------------------------------------------------
// printSummary outputs the dry-run summary.
// ---------------------------------------------------------------------------

func printSummary(stats *ImportStats) {
	fmt.Printf("Found %d YAML files, %d rules total", stats.FileCount, stats.RuleCount)
	if stats.SkipCount > 0 {
		fmt.Printf(" (%d skipped)", stats.SkipCount)
	}
	fmt.Println()
	fmt.Println()

	// Category breakdown
	fmt.Println("Category breakdown:")
	catOrder := []string{"infrastructure", "kubernetes", "middleware", "database", "probe", "windows"}
	for _, cat := range catOrder {
		cs, ok := stats.Categories[cat]
		if !ok {
			continue
		}
		// Sort subcategories for consistent output
		subKeys := make([]string, 0, len(cs.SubCats))
		for k := range cs.SubCats {
			subKeys = append(subKeys, k)
		}
		sort.Strings(subKeys)

		subParts := make([]string, 0, len(subKeys))
		for _, sk := range subKeys {
			subParts = append(subParts, fmt.Sprintf("%s=%d", sk, cs.SubCats[sk]))
		}
		fmt.Printf("  %s: %d rules (%s)\n", cat, cs.Total, strings.Join(subParts, ", "))
	}
	fmt.Println()

	// Severity breakdown
	fmt.Println("Severity breakdown:")
	sevOrder := []string{"critical", "warning", "info"}
	for _, sev := range sevOrder {
		if count, ok := stats.Severities[sev]; ok && count > 0 {
			fmt.Printf("  %s: %d rules\n", sev, count)
		}
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// batchUpsert inserts rules into the database with upsert behavior.
// ---------------------------------------------------------------------------

func batchUpsert(db *gorm.DB, rules []PresetRule) error {
	const batchSize = 100

	fmt.Printf("Importing %d rules in batches of %d...\n", len(rules), batchSize)

	for i := 0; i < len(rules); i += batchSize {
		end := i + batchSize
		if end > len(rules) {
			end = len(rules)
		}
		batch := rules[i:end]

		batchNum := i/batchSize + 1
		totalBatches := (len(rules) + batchSize - 1) / batchSize
		fmt.Printf("  Batch %d/%d (%d rules)...\n", batchNum, totalBatches, len(batch))

		result := db.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"display_name",
					"category",
					"sub_category",
					"component",
					"expression",
					"for_duration",
					"severity",
					"alert_type",
					"labels",
					"annotations",
					"source",
					"is_builtin",
					"description",
					"updated_at",
				}),
			},
		).Create(&batch)

		if result.Error != nil {
			return fmt.Errorf("batch %d failed: %w", batchNum, result.Error)
		}
	}

	return nil
}
