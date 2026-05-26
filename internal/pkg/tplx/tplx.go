// Package tplx provides a rich set of template functions for message rendering.
// Ported from Nightingale's pkg/tplx with SREAgent-specific adaptations.
package tplx

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	templateT "text/template"
	"time"
)

// TemplateFuncMap contains all template functions available for message rendering.
var TemplateFuncMap = template.FuncMap{
	// Time functions
	"timeformat": Timeformat,
	"timestamp":  Timestamp,
	"now":        Now,

	// String functions
	"toUpper":      strings.ToUpper,
	"toLower":      strings.ToLower,
	"contains":     strings.Contains,
	"title":        Title,
	"split":        strings.Split,
	"join":         strings.Join,
	"toString":     ToString,
	"reReplaceAll": ReReplaceAll,
	"match":        regexp.MatchString,
	"escape":       url.PathEscape,
	"stripPort":    StripPort,
	"stripDomain":  StripDomain,

	// Number formatting
	"humanize":                  Humanize,
	"humanizeDuration":          HumanizeDuration,
	"humanizeDurationInterface": HumanizeDurationInterface,
	"humanizePercentage":        HumanizePercentage,
	"humanizePercentageH":       HumanizePercentageH,
	"formatDecimal":             FormatDecimal,

	// Arithmetic
	"add": Add,
	"sub": Subtract,
	"mul": Multiply,
	"div": Divide,

	// Encoding
	"b64enc":     B64Enc,
	"b64dec":     B64Dec,
	"jsonMarshal": JsonMarshal,

	// HTML / template helpers
	"unescaped":  Unescaped,
	"safeHtml":   SafeHtml,
	"printf":     Printf,

	// Map helpers
	"tagsMapToStr":  TagsMapToStr,
	"mapDifference": MapDifference,
}

// NewTemplateFuncMap returns a copy of TemplateFuncMap (copy-on-write pattern).
func NewTemplateFuncMap() template.FuncMap {
	m := template.FuncMap{}
	for k, v := range TemplateFuncMap {
		m[k] = v
	}
	return m
}

// ---------------------------------------------------------------------------
// Time functions
// ---------------------------------------------------------------------------

// Timeformat formats a Unix timestamp (seconds) with an optional layout pattern.
func Timeformat(ts int64, pattern ...string) string {
	defp := "2006-01-02 15:04:05"
	if len(pattern) > 0 {
		defp = pattern[0]
	}
	return time.Unix(ts, 0).Format(defp)
}

// Timestamp returns the current time formatted with an optional layout pattern.
func Timestamp(pattern ...string) string {
	defp := "2006-01-02 15:04:05"
	if len(pattern) > 0 {
		defp = pattern[0]
	}
	return time.Now().Format(defp)
}

// Now returns the current time.
func Now() time.Time {
	return time.Now()
}

// ---------------------------------------------------------------------------
// String helpers
// ---------------------------------------------------------------------------

// Title returns a copy of s with the first letter of each word capitalised.
func Title(s string) string {
	return strings.Title(s) //nolint:staticcheck // ported from Nightingale
}

// ToString converts any value to its string representation.
func ToString(v interface{}) string {
	return fmt.Sprint(v)
}

// ReReplaceAll replaces all matches of pattern in text with repl.
func ReReplaceAll(pattern, repl, text string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(text, repl)
}

// StripPort removes the port from a host:port string.
func StripPort(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}

// StripDomain extracts the hostname (first label) from an FQDN:port string.
func StripDomain(hostPort string) string {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return hostPort
	}
	host = strings.Split(host, ".")[0]
	if port != "" {
		return net.JoinHostPort(host, port)
	}
	return host
}

// ---------------------------------------------------------------------------
// Number formatting
// ---------------------------------------------------------------------------

// Humanize converts a numeric string to a human-readable form with SI prefixes.
func Humanize(s string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	if v == 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("%.2f", v)
	}
	if math.Abs(v) >= 1 {
		prefix := ""
		for _, p := range []string{"k", "M", "G", "T", "P", "E", "Z", "Y"} {
			if math.Abs(v) < 1000 {
				break
			}
			prefix = p
			v /= 1000
		}
		return fmt.Sprintf("%.2f%s", v, prefix)
	}
	prefix := ""
	for _, p := range []string{"m", "u", "n", "p", "f", "a", "z", "y"} {
		if math.Abs(v) >= 1 {
			break
		}
		prefix = p
		v *= 1000
	}
	return fmt.Sprintf("%.2f%s", v, prefix)
}

// HumanizeDuration converts a duration in seconds (as string) to a human-readable form.
func HumanizeDuration(s string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return humanizeDurationFloat64(v)
}

// HumanizeDurationInterface converts a duration in seconds (any numeric type) to a human-readable form.
func HumanizeDurationInterface(i interface{}) string {
	f, err := toFloat64(i)
	if err != nil {
		return ToString(i)
	}
	return humanizeDurationFloat64(f)
}

func humanizeDurationFloat64(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("%.4g", v)
	}
	if v == 0 {
		return fmt.Sprintf("%.4gs", v)
	}
	if math.Abs(v) >= 1 {
		sign := ""
		if v < 0 {
			sign = "-"
			v = -v
		}
		seconds := int64(v) % 60
		minutes := (int64(v) / 60) % 60
		hours := (int64(v) / 60 / 60) % 24
		days := int64(v) / 60 / 60 / 24
		if days != 0 {
			return fmt.Sprintf("%s%dd %dh %dm %ds", sign, days, hours, minutes, seconds)
		}
		if hours != 0 {
			return fmt.Sprintf("%s%dh %dm %ds", sign, hours, minutes, seconds)
		}
		if minutes != 0 {
			return fmt.Sprintf("%s%dm %ds", sign, minutes, seconds)
		}
		return fmt.Sprintf("%s%.4gs", sign, v)
	}
	prefix := ""
	for _, p := range []string{"m", "u", "n", "p", "f", "a", "z", "y"} {
		if math.Abs(v) >= 1 {
			break
		}
		prefix = p
		v *= 1000
	}
	return fmt.Sprintf("%.4g%ss", v, prefix)
}

// HumanizePercentage multiplies a fractional value by 100 and appends "%".
func HumanizePercentage(s string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%.2f%%", v*100)
}

// HumanizePercentageH formats a pre-multiplied percentage value with "%".
func HumanizePercentageH(s string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%.2f%%", v)
}

// FormatDecimal formats a numeric string to n decimal places.
func FormatDecimal(s string, n int) string {
	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	format := fmt.Sprintf("%%.%df", n)
	return fmt.Sprintf(format, num)
}

// ---------------------------------------------------------------------------
// Arithmetic
// ---------------------------------------------------------------------------

// Add returns the sum of a and b.
func Add(a, b interface{}) (interface{}, error) {
	return arithmeticOp(a, b, "add", func(av, bv float64) float64 { return av + bv })
}

// Subtract returns the difference of a and b.
func Subtract(a, b interface{}) (interface{}, error) {
	return arithmeticOp(a, b, "subtract", func(av, bv float64) float64 { return av - bv })
}

// Multiply returns the product of a and b.
func Multiply(a, b interface{}) (interface{}, error) {
	return arithmeticOp(a, b, "multiply", func(av, bv float64) float64 { return av * bv })
}

// Divide returns the quotient of a and b.
func Divide(a, b interface{}) (interface{}, error) {
	return arithmeticOp(a, b, "divide", func(av, bv float64) float64 { return av / bv })
}

func arithmeticOp(a, b interface{}, name string, op func(float64, float64) float64) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if name == "divide" {
				return float64(av.Int()) / float64(bv.Int()), nil
			}
			return intOp(av.Int(), bv.Int(), name, op), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if name == "divide" {
				return float64(av.Int()) / float64(bv.Uint()), nil
			}
			return intOp(av.Int(), int64(bv.Uint()), name, op), nil
		case reflect.Float32, reflect.Float64:
			return op(float64(av.Int()), bv.Float()), nil
		default:
			return nil, fmt.Errorf("%s: unknown type for %q (%T)", name, bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if name == "divide" {
				return float64(av.Uint()) / float64(bv.Int()), nil
			}
			return intOp(int64(av.Uint()), bv.Int(), name, op), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if name == "divide" {
				return float64(av.Uint()) / float64(bv.Uint()), nil
			}
			return uintOp(av.Uint(), bv.Uint(), name, op), nil
		case reflect.Float32, reflect.Float64:
			return op(float64(av.Uint()), bv.Float()), nil
		default:
			return nil, fmt.Errorf("%s: unknown type for %q (%T)", name, bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return op(av.Float(), float64(bv.Int())), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return op(av.Float(), float64(bv.Uint())), nil
		case reflect.Float32, reflect.Float64:
			return op(av.Float(), bv.Float()), nil
		default:
			return nil, fmt.Errorf("%s: unknown type for %q (%T)", name, bv, b)
		}
	default:
		return nil, fmt.Errorf("%s: unknown type for %q (%T)", name, av, a)
	}
}

func intOp(a, b int64, name string, op func(float64, float64) float64) interface{} {
	if name == "multiply" {
		return a * b
	}
	if name == "add" {
		return a + b
	}
	if name == "subtract" {
		return a - b
	}
	return op(float64(a), float64(b))
}

func uintOp(a, b uint64, name string, op func(float64, float64) float64) interface{} {
	if name == "multiply" {
		return a * b
	}
	if name == "add" {
		return a + b
	}
	if name == "subtract" {
		return a - b
	}
	return op(float64(a), float64(b))
}

// ---------------------------------------------------------------------------
// Encoding
// ---------------------------------------------------------------------------

// B64Enc encodes a string to base64.
func B64Enc(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// B64Dec decodes a base64 string. Returns the original string on error.
func B64Dec(s string) string {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return string(data)
}

// JsonMarshal serializes a value to JSON and returns it as safe HTML.
func JsonMarshal(v interface{}) template.HTML {
	j, err := json.Marshal(v)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(string(j))
}

// ---------------------------------------------------------------------------
// HTML / template helpers
// ---------------------------------------------------------------------------

// Unescaped marks a string as safe HTML (no escaping).
func Unescaped(str string) interface{} {
	return template.HTML(str)
}

// SafeHtml marks text as safe HTML.
func SafeHtml(text string) template.HTML {
	return template.HTML(text)
}

// Printf formats a value using the given format string.
// If the value is a string with a unit suffix (e.g. "95.2%"), it is returned as-is.
func Printf(format string, value interface{}) string {
	valType := reflect.TypeOf(value).Kind()

	switch valType {
	case reflect.String:
		strValue := value.(string)
		if isValueWithUnit(strValue) {
			return strValue
		}
		if floatValue, err := strconv.ParseFloat(strValue, 64); err == nil {
			return fmt.Sprintf(format, floatValue)
		}
		return fmt.Sprintf(format, value)
	case reflect.Float64, reflect.Float32:
		return fmt.Sprintf(format, value)
	default:
		return fmt.Sprintf(format, value)
	}
}

// isValueWithUnit checks if a string is a numeric value with a unit suffix.
func isValueWithUnit(s string) bool {
	if s == "" {
		return false
	}
	hasDigit := false
	hasUnit := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			hasDigit = true
		} else if r == '.' || r == '-' || r == '+' {
			continue
		} else {
			hasUnit = true
		}
	}
	return hasDigit && hasUnit
}

// ---------------------------------------------------------------------------
// Map helpers
// ---------------------------------------------------------------------------

// TagsMapToStr converts a map[string]string to sorted "key=value" pairs joined by commas.
func TagsMapToStr(m map[string]string) string {
	strs := make([]string, 0, len(m))
	for key, value := range m {
		strs = append(strs, key+"="+value)
	}
	sort.Strings(strs)
	return strings.Join(strs, ",")
}

// MapDifference returns key-value pairs present in firstMap but not in secondMap.
func MapDifference(firstMap, secondMap map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	for key, value := range firstMap {
		if _, exists := secondMap[key]; !exists {
			result[key] = value
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Template rendering helpers
// ---------------------------------------------------------------------------

// ReplaceTemplateUseHtml renders a template string using html/template.
func ReplaceTemplateUseHtml(name string, templateText string, templateData any) string {
	tpl, err := template.New(name).Funcs(TemplateFuncMap).Parse(templateText)
	if err != nil {
		return templateText
	}
	var body bytes.Buffer
	if err := tpl.Execute(&body, templateData); err != nil {
		return templateText
	}
	return body.String()
}

// ReplaceTemplateUseText renders a template string using text/template.
func ReplaceTemplateUseText(name string, templateText string, templateData any) string {
	tpl, err := templateT.New(name).Funcs(TextTemplateFuncMap()).Parse(templateText)
	if err != nil {
		return templateText
	}
	var body bytes.Buffer
	if err := tpl.Execute(&body, templateData); err != nil {
		return templateText
	}
	return body.String()
}

// TextTemplateFuncMap returns a text/template-compatible FuncMap.
func TextTemplateFuncMap() templateT.FuncMap {
	m := templateT.FuncMap{}
	for k, v := range TemplateFuncMap {
		m[k] = v
	}
	return m
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// toFloat64 converts an interface{} to float64.
func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
		if i, err := strconv.ParseInt(v, 0, 64); err == nil {
			return float64(i), nil
		}
		b, err := strconv.ParseBool(v)
		if err == nil {
			if b {
				return 1, nil
			}
			return 0, nil
		}
		if v == "Yes" || v == "yes" || v == "YES" || v == "Y" || v == "ON" || v == "on" || v == "On" || v == "ok" || v == "up" {
			return 1, nil
		}
		if v == "No" || v == "no" || v == "NO" || v == "N" || v == "OFF" || v == "off" || v == "Off" || v == "fail" || v == "err" || v == "down" {
			return 0, nil
		}
		return 0, fmt.Errorf("unparsable value %v", v)
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return strconv.ParseFloat(fmt.Sprint(v), 64)
	}
}
