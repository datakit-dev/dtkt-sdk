package util_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

func TestSlugify(t *testing.T) {
	var tests = []struct {
		name  string
		input string
		want  string
	}{
		{"Input with hyphens", "ABC-123-def", "abc-123-def"},
		{"Spaces to hyphens, trim ends", "Super Cool, Inc.", "super-cool-inc"},
		{"Spaces to underscores", "Super Cool, Inc.", "super-cool-inc"},
		{"No valid characters", "@#$%^&*()", ""},
		{"Package identity", "FooBar@1.0.0", "foo-bar-1-0-0"},
		{"URI scheme", "unix+tcp", "unix-tcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.Slugify(tt.input)
			if got != tt.want {
				t.Errorf(`%q for %q got %q, want %q`, tt.name, tt.input, got, tt.want)
			}
		})
	}
}

func TestSlugifyWithSeparator(t *testing.T) {
	var tests = []struct {
		name  string
		input string
		want  string
	}{
		{"Input with hyphens", "ABC-123-def", "abc_123_def"},
		{"Spaces to hyphens, trim ends", "Super Cool, Inc.", "super_cool_inc"},
		{"Spaces to underscores", "Super Cool, Inc.", "super_cool_inc"},
		{"No valid characters", "@#$%^&*()", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.SlugifyWithSeparator('_', tt.input)
			if got != tt.want {
				t.Errorf(`%q for %q got %q, want %q`, tt.name, tt.input, got, tt.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "HelloWorld"},
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"  leading spaces", "LeadingSpaces"},
		{"trailing spaces  ", "TrailingSpaces"},
		// {"MIXED_case-string", "MIXEDCaseString"},
		{"alreadyPascal", "AlreadyPascal"},
		{"", ""},
		{"    ", ""},
		{"123number first", "123numberFirst"},
		{"foo_bar-baz", "FooBarBaz"},
		// {"ALLCAPS", "ALLCAPS"},
	}

	for _, tt := range tests {
		t.Run("ToPascalCase_"+tt.input, func(t *testing.T) {
			result := util.ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HelloWorld", "hello_world"},
		{"AlreadySnake", "already_snake"},
		{"fooBarBaz", "foo_bar_baz"},
		// {"ALLCAPS", "allcaps"},
		{"simple", "simple"},
		{"", ""},
		{"XMLHttpRequest", "xml_http_request"},
	}

	for _, tt := range tests {
		t.Run("ToSnakeCase_"+tt.input, func(t *testing.T) {
			result := util.ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HelloWorld", "hello-world"},
		{"AlreadyKebab", "already-kebab"},
		{"fooBarBaz", "foo-bar-baz"},
		{"ALLCAPS", "allcaps"},
		{"simple", "simple"},
		{"", ""},
		{"XMLHttpRequest", "xml-http-request"},
		{"FooBar@1.0.0", "foo-bar@1.0.0"},
		{"OpenURL", "open-url"},
	}

	for _, tt := range tests {
		t.Run("ToKebabCase_"+tt.input, func(t *testing.T) {
			result := util.ToKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToKebabCase(%q) = %q; want %q", tt.input, result, tt.expected)
			} else {
				t.Log(result)
			}
		})
	}
}
