package imagesExtractor

import (
	"os"
	"testing"
)

func TestExtractZip(t *testing.T) {
	t.Run("ValidZip", func(t *testing.T) {
		// Provide the path to a valid zip file for testing
		validZipPath := "../../test_files/withDockerInZip.zip"

		extractDir, err := extractZip(validZipPath)
		if err != nil {
			t.Fatalf("Error extracting valid zip file: %v", err)
		}

		// Check if the extraction directory exists
		if _, err := os.Stat(extractDir); os.IsNotExist(err) {
			t.Errorf("Extraction directory does not exist: %s", extractDir)
		}
		_ = os.RemoveAll(extractDir)
	})

	t.Run("InvalidZip", func(t *testing.T) {
		// Provide the path to an invalid zip file for testing
		invalidZipPath := "../../test_files/invalidWithDockerInZip.zip"

		_, err := extractZip(invalidZipPath)
		if err == nil {
			t.Error("Expected error extracting invalid zip file, but got nil")
		}
	})
}
