package invoice

import (
	"os"
	"strings"
	"testing"
)

func TestParseItemFlag(t *testing.T) {
	input := "name=Consulting,price=100000,quantity=1,excise=10"
	item, err := parseItemFlag(input)
	if err != nil {
		t.Fatalf("parseItemFlag: %v", err)
	}
	if item.Name != "Consulting" {
		t.Errorf("Name = %q, want %q", item.Name, "Consulting")
	}
	if item.Price != 100000 {
		t.Errorf("Price = %d, want 100000", item.Price)
	}
	if item.Quantity != 1 {
		t.Errorf("Quantity = %d, want 1", item.Quantity)
	}
	if item.Excise != "ten_percent" {
		t.Errorf("Excise = %q, want %q", item.Excise, "ten_percent")
	}
}

func TestParseItemFlag_AllFields(t *testing.T) {
	input := "name=Test,code=T001,detail=Desc,unit=hours,price=5000,quantity=3,excise=8r"
	item, err := parseItemFlag(input)
	if err != nil {
		t.Fatalf("parseItemFlag: %v", err)
	}
	if item.Code != "T001" {
		t.Errorf("Code = %q, want %q", item.Code, "T001")
	}
	if item.Detail != "Desc" {
		t.Errorf("Detail = %q, want %q", item.Detail, "Desc")
	}
	if item.Unit != "hours" {
		t.Errorf("Unit = %q, want %q", item.Unit, "hours")
	}
	if item.Excise != "eight_percent_as_reduced_tax_rate" {
		t.Errorf("Excise = %q, want %q", item.Excise, "eight_percent_as_reduced_tax_rate")
	}
}

func TestParseItemFlag_MissingName(t *testing.T) {
	_, err := parseItemFlag("price=100,quantity=1,excise=10")
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseItemsJSON(t *testing.T) {
	jsonStr := `[{"name":"A","price":100,"quantity":1,"excise":"ten_percent"},{"name":"B","price":200,"quantity":2,"excise":"untaxable"}]`
	items, err := parseItemsJSON(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("parseItemsJSON: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	if items[0].Name != "A" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "A")
	}
	if items[1].Price != 200 {
		t.Errorf("items[1].Price = %d, want 200", items[1].Price)
	}
}

func TestParseItemsYAML(t *testing.T) {
	yamlStr := `- name: A
  price: 100
  quantity: 1
  excise: ten_percent
- name: B
  price: 200
  quantity: 2
  excise: untaxable
`
	items, err := parseItemsYAML(strings.NewReader(yamlStr))
	if err != nil {
		t.Fatalf("parseItemsYAML: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	if items[0].Name != "A" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "A")
	}
	if items[1].Price != 200 {
		t.Errorf("items[1].Price = %d, want 200", items[1].Price)
	}
}

func TestParseItemsFromFile_JSON(t *testing.T) {
	f := t.TempDir() + "/items.json"
	os.WriteFile(f, []byte(`[{"name":"X","price":1,"quantity":1,"excise":"ten_percent"}]`), 0644)
	items, err := parseItemsFromFile(f)
	if err != nil {
		t.Fatalf("parseItemsFromFile JSON: %v", err)
	}
	if len(items) != 1 || items[0].Name != "X" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestParseItemsFromFile_YAML(t *testing.T) {
	f := t.TempDir() + "/items.yaml"
	os.WriteFile(f, []byte("- name: Y\n  price: 2\n  quantity: 3\n  excise: untaxable\n"), 0644)
	items, err := parseItemsFromFile(f)
	if err != nil {
		t.Fatalf("parseItemsFromFile YAML: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Y" {
		t.Errorf("unexpected items: %+v", items)
	}
	if items[0].Excise != "untaxable" {
		t.Errorf("Excise = %q, want %q", items[0].Excise, "untaxable")
	}
}
