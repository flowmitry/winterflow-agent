package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveWithSpecialCharacters(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "env-save-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file path
	testFilePath := filepath.Join(tempDir, ".env")

	// Test cases with problematic values
	testCases := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "Value with question marks",
			value:    "'URL\"?zf6WH?BACd",
			expected: "TEST_URL=\"'URL\\\\\"?zf6WH?BACd\"\n",
		},
		{
			name:     "Value with equals sign",
			value:    "key=value",
			expected: "TEST_KEY=\"key=value\"\n",
		},
		{
			name:     "Value with spaces",
			value:    "value with spaces",
			expected: "TEST_SPACES=\"value with spaces\"\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test variables with the problematic value
			vars := map[string]string{}

			// Use different keys for different test cases
			var key string
			if tc.name == "Value with question marks" {
				key = "TEST_URL"
			} else if tc.name == "Value with equals sign" {
				key = "TEST_KEY"
			} else {
				key = "TEST_SPACES"
			}

			vars[key] = tc.value

			// Save the variables to the file
			err = Save(testFilePath, vars)
			if err != nil {
				t.Fatalf("Failed to save env file: %v", err)
			}

			// Read the file content
			content, err := os.ReadFile(testFilePath)
			if err != nil {
				t.Fatalf("Failed to read env file: %v", err)
			}

			// Check if the content matches the expected output
			if string(content) != tc.expected {
				t.Errorf("Expected file content to be %q, got %q", tc.expected, string(content))
			}
		})
	}
}

func TestSaveWithMultilineValues(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "env-save-multiline-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file path
	testFilePath := filepath.Join(tempDir, ".env")

	// Test cases with multiline values
	testCases := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "Simple multiline value with Unix line endings",
			value:    "line1\nline2\nline3",
			expected: "MULTILINE_UNIX=\"line1\\nline2\\nline3\"\n",
		},
		{
			name:     "Simple multiline value with Windows line endings",
			value:    "line1\r\nline2\r\nline3",
			expected: "MULTILINE_WINDOWS=\"line1\\nline2\\nline3\"\n",
		},
		{
			name:     "Simple multiline value with old Mac line endings",
			value:    "line1\rline2\rline3",
			expected: "MULTILINE_MAC=\"line1\\rline2\\rline3\"\n",
		},
		{
			name:     "Complex multiline value with special characters",
			value:    "line1 with spaces\nline2 with \"quotes\"\nline3 with ?special=chars&",
			expected: "MULTILINE_COMPLEX=\"line1 with spaces\\nline2 with \\\\\"quotes\\\\\"\\nline3 with ?special=chars&\"\n",
		},
		{
			name:     "Multiline value with mixed line endings",
			value:    "line1\nline2\r\nline3\r",
			expected: "MULTILINE_MIXED=\"line1\\nline2\\nline3\\r\"\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test variables with the multiline value
			vars := map[string]string{}

			// Use different keys for different test cases
			var key string
			switch tc.name {
			case "Simple multiline value with Unix line endings":
				key = "MULTILINE_UNIX"
			case "Simple multiline value with Windows line endings":
				key = "MULTILINE_WINDOWS"
			case "Simple multiline value with old Mac line endings":
				key = "MULTILINE_MAC"
			case "Complex multiline value with special characters":
				key = "MULTILINE_COMPLEX"
			case "Multiline value with mixed line endings":
				key = "MULTILINE_MIXED"
			}

			vars[key] = tc.value

			// Save the variables to the file
			err = Save(testFilePath, vars)
			if err != nil {
				t.Fatalf("Failed to save env file: %v", err)
			}

			// Read the file content
			content, err := os.ReadFile(testFilePath)
			if err != nil {
				t.Fatalf("Failed to read env file: %v", err)
			}

			// Check if the content matches the expected output
			if string(content) != tc.expected {
				t.Errorf("Expected file content to be %q, got %q", tc.expected, string(content))
			}
		})
	}
}
