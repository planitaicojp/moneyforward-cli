package invoice

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"gopkg.in/yaml.v3"
)

// parseItemFlag parses a single --item flag value like "name=Consulting,price=100000,quantity=1,excise=10".
func parseItemFlag(s string) (model.InvoiceTemplateLine, error) {
	var item model.InvoiceTemplateLine
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return item, fmt.Errorf("invalid key=value pair: %q", pair)
		}
		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch key {
		case "name":
			item.Name = val
		case "code":
			item.Code = val
		case "detail":
			item.Detail = val
		case "unit":
			item.Unit = val
		case "price":
			n, err := strconv.Atoi(val)
			if err != nil {
				return item, fmt.Errorf("invalid price %q: %w", val, err)
			}
			item.Price = n
		case "quantity":
			n, err := strconv.Atoi(val)
			if err != nil {
				return item, fmt.Errorf("invalid quantity %q: %w", val, err)
			}
			item.Quantity = n
		case "excise":
			item.Excise = resolveExcise(val)
		default:
			return item, fmt.Errorf("unknown item field: %q", key)
		}
	}
	if item.Name == "" {
		return item, fmt.Errorf("item name is required")
	}
	return item, nil
}

// parseItemsJSON reads JSON array of InvoiceTemplateLine from a reader.
func parseItemsJSON(r io.Reader) ([]model.InvoiceTemplateLine, error) {
	var items []model.InvoiceTemplateLine
	if err := json.NewDecoder(r).Decode(&items); err != nil {
		return nil, fmt.Errorf("parsing items JSON: %w", err)
	}
	return items, nil
}

// parseItemsYAML reads YAML array of InvoiceTemplateLine from a reader.
func parseItemsYAML(r io.Reader) ([]model.InvoiceTemplateLine, error) {
	var items []model.InvoiceTemplateLine
	if err := yaml.NewDecoder(r).Decode(&items); err != nil {
		return nil, fmt.Errorf("parsing items YAML: %w", err)
	}
	return items, nil
}

// parseItemsFromFile reads line items from a file, detecting format by extension.
// .yaml and .yml files are parsed as YAML; everything else as JSON.
func parseItemsFromFile(path string) ([]model.InvoiceTemplateLine, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening items file: %w", err)
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		return parseItemsYAML(f)
	}
	return parseItemsJSON(f)
}

// resolveLineItems resolves line items from --items-stdin, --items-file, or --item flags.
// Priority: stdin > file > flags.
func resolveLineItems(itemFlags []string, itemsFile string, itemsStdin bool) ([]model.InvoiceTemplateLine, error) {
	if itemsStdin {
		return parseItemsJSON(os.Stdin)
	}
	if itemsFile != "" {
		return parseItemsFromFile(itemsFile)
	}
	if len(itemFlags) > 0 {
		items := make([]model.InvoiceTemplateLine, 0, len(itemFlags))
		for _, flag := range itemFlags {
			item, err := parseItemFlag(flag)
			if err != nil {
				return nil, fmt.Errorf("parsing --item %q: %w", flag, err)
			}
			items = append(items, item)
		}
		return items, nil
	}
	return nil, nil
}
