package transform

import (
	"context"
	"github.com/firfircelik/go-intake"
	"testing"
)

func TestSnakeCase(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"Product Name", "product_name"},
		{"product name", "product_name"},
		{"ProductName", "product_name"},
		{"product-name", "product_name"},
		{"product.name", "product_name"},
		{"product_name", "product_name"},
		{"X", "x"},
		{"XPathResult", "xpath_result"},
		{"  spaced  ", "spaced"},
		{"a__b", "a_b"},
		{"trailing_", "trailing"},
		{"_leading", "leading"},
	}
	for _, c := range cases {
		got := SnakeCase(c.in)
		if got != c.want {
			t.Errorf("SnakeCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCamelCase(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Product Name", "productName"},
		{"product name", "productName"},
		{"product_name", "productName"},
		{"ProductName", "productName"},
		{"product-name", "productName"},
		{"X", "x"},
		{"xpath result", "xpathResult"},
	}
	for _, c := range cases {
		got := CamelCase(c.in)
		if got != c.want {
			t.Errorf("CamelCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKebabCase(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Product Name", "product-name"},
		{"product_name", "product-name"},
		{"ProductName", "product-name"},
	}
	for _, c := range cases {
		got := KebabCase(c.in)
		if got != c.want {
			t.Errorf("KebabCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeHeaders(t *testing.T) {
	tr := NormalizeHeaders(SnakeCase)
	in := intake.Record{"Product Name": "x", "unit-price": "5"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out["product_name"] != "x" {
		t.Errorf("product_name = %v", out["product_name"])
	}
	if out["unit_price"] != "5" {
		t.Errorf("unit_price = %v", out["unit_price"])
	}
	if _, ok := out["Product Name"]; ok {
		t.Error("old key should be gone")
	}
}

func TestNormalizeHeadersCollisions(t *testing.T) {
	tr := NormalizeHeaders(LowerCase)
	in := intake.Record{"A": "1", "a": "2"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	// First occurrence wins.
	if out["a"] != "1" {
		t.Errorf("a = %v, want 1 (first occurrence wins)", out["a"])
	}
}

func TestNormalizeHeadersEmptyValue(t *testing.T) {
	tr := NormalizeHeaders(func(s string) string { return "" })
	in := intake.Record{"a": "1"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("output = %v, want empty", out)
	}
}

func TestNormalizeHeadersNilRecord(t *testing.T) {
	tr := NormalizeHeaders(SnakeCase)
	out, err := tr.Apply(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Errorf("nil record should produce nil, got %v", out)
	}
}

func TestNormalizeHeadersNoMutation(t *testing.T) {
	tr := NormalizeHeaders(SnakeCase)
	in := intake.Record{"Product Name": "x"}
	out, err := tr.Apply(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in["product_name"]; ok {
		t.Error("input was mutated: product_name was added")
	}
	if in["Product Name"] != "x" {
		t.Errorf("input was mutated: Product Name = %v", in["Product Name"])
	}
	out["product_name"] = "mutated"
	if in["Product Name"] != "x" {
		t.Errorf("mutating output affected input: Product Name = %v", in["Product Name"])
	}
}
