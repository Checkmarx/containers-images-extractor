package imagesExtractor

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Checkmarx/containers-images-extractor/internal/extractors"
	"github.com/Checkmarx/containers-types/types"
)

func TestExtractAndMergeImagesFromFiles(t *testing.T) {
	extractor := NewImagesExtractor()
	// Define test scenarios
	scenarios := []struct {
		Name             string
		Files            types.FileImages
		UserInput        []types.ImageModel
		ExpectedImages   []types.ImageModel
		ExpectedErrorMsg string
	}{
		{
			Name: "DifferentImagesFromDifferentSources",
			Files: types.FileImages{
				Dockerfile: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile", RelativePath: "Dockerfile"},
				},
				DockerCompose: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose.yaml", RelativePath: "docker-compose1.yml"},
				},
				Helm: []types.HelmChartInfo{
					{
						Directory:  "../../test_files/imageExtraction/helm/",
						ValuesFile: "../../test_files/imageExtraction/helm/values.yaml",
						TemplateFiles: []types.FilePath{{FullPath: "../test_files/imageExtraction/helm/templates/containers-worker.yaml", RelativePath: "templates/containers-worker.yaml"},
							{FullPath: "../test_files/imageExtraction/helm/templates/image-insights.yaml", RelativePath: "templates/image-insights.yaml"}},
					},
				},
			},
			UserInput: []types.ImageModel{
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath}}},
			},
			ExpectedImages: []types.ImageModel{
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath, Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "mcr.microsoft.com/dotnet/sdk:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerFileOrigin, Path: "Dockerfile", FinalStage: false, Line: 1, StartIndex: 5, EndIndex: 37}}},
				{Name: "mcr.microsoft.com/dotnet/aspnet:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerFileOrigin, Path: "Dockerfile", FinalStage: true, Line: 25, StartIndex: 5, EndIndex: 40}}},
				{Name: "buildimage:latest", ImageLocations: []types.ImageLocation{{Origin: types.DockerComposeFileOrigin, Path: "docker-compose1.yml", Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "checkmarx.jfrog.io/ast-docker/containers-worker:b201b1f", ImageLocations: []types.ImageLocation{{Origin: types.HelmFileOrigin, Path: "containers/templates/containers-worker.yaml", Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "checkmarx.jfrog.io/ast-docker/image-insights:f4b507b", ImageLocations: []types.ImageLocation{{Origin: types.HelmFileOrigin, Path: "containers/templates/image-insights.yaml", Line: 0, StartIndex: 0, EndIndex: 0}}},
			},
		},
		{
			Name: "SameImagesFromDifferentSources",
			Files: types.FileImages{
				Dockerfile: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile", RelativePath: "Dockerfile"},
				},
				DockerCompose: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose-4.yaml", RelativePath: "docker-compose-4.yml"},
				},
			},
			UserInput: []types.ImageModel{
				{Name: "mcr.microsoft.com/dotnet/sdk:6.0", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath, Line: 1, StartIndex: 5, EndIndex: 37}}},
			},
			ExpectedImages: []types.ImageModel{
				{Name: "mcr.microsoft.com/dotnet/sdk:6.0", ImageLocations: []types.ImageLocation{
					{Origin: types.UserInput, Path: types.NoFilePath, Line: 1, StartIndex: 5, EndIndex: 37},
					{Origin: types.DockerFileOrigin, Path: "Dockerfile", Line: 1, StartIndex: 5, EndIndex: 37},
					{Origin: types.DockerComposeFileOrigin, Path: "docker-compose-4.yml", Line: 0, StartIndex: 0, EndIndex: 0},
				}},
				{Name: "mcr.microsoft.com/dotnet/aspnet:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerFileOrigin, Path: "Dockerfile", FinalStage: true, Line: 25, StartIndex: 5, EndIndex: 40}}},
			},
		},
		{
			Name: "OnlyDockerfileFound",
			Files: types.FileImages{
				Dockerfile: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile", RelativePath: "Dockerfile"},
				},
			},
			UserInput: []types.ImageModel{
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath}}},
			},
			ExpectedImages: []types.ImageModel{
				{Name: "mcr.microsoft.com/dotnet/sdk:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerFileOrigin, Path: "Dockerfile", FinalStage: false, Line: 1, StartIndex: 5, EndIndex: 37}}},
				{Name: "mcr.microsoft.com/dotnet/aspnet:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerFileOrigin, Path: "Dockerfile", FinalStage: true, Line: 25, StartIndex: 5, EndIndex: 40}}},
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath, Line: 0, StartIndex: 0, EndIndex: 0}}}},
		},
		{
			Name: "OnlyDockerComposeFound",
			Files: types.FileImages{
				DockerCompose: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose-4.yaml", RelativePath: "docker-compose-4.yml"},
				},
			},
			UserInput: []types.ImageModel{
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath}}},
			},
			ExpectedImages: []types.ImageModel{
				{Name: "mcr.microsoft.com/dotnet/sdk:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerComposeFileOrigin, Path: "docker-compose-4.yml", Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath, Line: 0, StartIndex: 0, EndIndex: 0}}},
			},
		},
		{
			Name: "OnlyHelmChartFound",
			Files: types.FileImages{
				Helm: []types.HelmChartInfo{
					{
						Directory:  "../../test_files/imageExtraction/helm/",
						ValuesFile: "../../test_files/imageExtraction/helm/values.yaml",
						TemplateFiles: []types.FilePath{{FullPath: "../test_files/imageExtraction/helm/templates/containers-worker.yaml", RelativePath: "templates/containers-worker.yaml"},
							{FullPath: "../test_files/imageExtraction/helm/templates/image-insights.yaml", RelativePath: "templates/image-insights.yaml"}},
					},
				},
			},
			UserInput: []types.ImageModel{
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath}}},
			},
			ExpectedImages: []types.ImageModel{
				{Name: "debian:11", ImageLocations: []types.ImageLocation{{Origin: types.UserInput, Path: types.NoFilePath, Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "checkmarx.jfrog.io/ast-docker/containers-worker:b201b1f", ImageLocations: []types.ImageLocation{{Origin: types.HelmFileOrigin, Path: "containers/templates/containers-worker.yaml", Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "checkmarx.jfrog.io/ast-docker/image-insights:f4b507b", ImageLocations: []types.ImageLocation{{Origin: types.HelmFileOrigin, Path: "containers/templates/image-insights.yaml", Line: 0, StartIndex: 0, EndIndex: 0}}},
			},
		},
		{
			Name: "AllTypesOfFilesWithNoExistingImages",
			Files: types.FileImages{
				Dockerfile: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile", RelativePath: "Dockerfile"},
				},
				DockerCompose: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose.yaml", RelativePath: "docker-compose1.yml"},
				},
				Helm: []types.HelmChartInfo{
					{
						Directory:  "../../test_files/imageExtraction/helm/",
						ValuesFile: "../../test_files/imageExtraction/helm/values.yaml",
						TemplateFiles: []types.FilePath{{FullPath: "../test_files/imageExtraction/helm/templates/containers-worker.yaml", RelativePath: "templates/containers-worker.yaml"},
							{FullPath: "../test_files/imageExtraction/helm/templates/image-insights.yaml", RelativePath: "templates/image-insights.yaml"}},
					},
				},
			},
			ExpectedImages: []types.ImageModel{
				{Name: "mcr.microsoft.com/dotnet/sdk:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerFileOrigin, Path: "Dockerfile", FinalStage: false, Line: 1, StartIndex: 5, EndIndex: 37}}},
				{Name: "mcr.microsoft.com/dotnet/aspnet:6.0", ImageLocations: []types.ImageLocation{{Origin: types.DockerFileOrigin, Path: "Dockerfile", FinalStage: true, Line: 25, StartIndex: 5, EndIndex: 40}}},
				{Name: "buildimage:latest", ImageLocations: []types.ImageLocation{{Origin: types.DockerComposeFileOrigin, Path: "docker-compose1.yml", Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "checkmarx.jfrog.io/ast-docker/containers-worker:b201b1f", ImageLocations: []types.ImageLocation{{Origin: types.HelmFileOrigin, Path: "containers/templates/containers-worker.yaml", Line: 0, StartIndex: 0, EndIndex: 0}}},
				{Name: "checkmarx.jfrog.io/ast-docker/image-insights:f4b507b", ImageLocations: []types.ImageLocation{{Origin: types.HelmFileOrigin, Path: "containers/templates/image-insights.yaml", Line: 0, StartIndex: 0, EndIndex: 0}}},
			},
		},
	}

	// Run test scenarios
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Run the function
			result, err := extractor.ExtractAndMergeImagesFromFiles(scenario.Files, scenario.UserInput, nil)

			// Check for errors
			if scenario.ExpectedErrorMsg != "" {
				if err == nil || err.Error() != scenario.ExpectedErrorMsg {
					t.Errorf("Expected error message '%s' but got '%v'", scenario.ExpectedErrorMsg, err)
				}
			} else {
				// Check for expected images
				expectedImageMap := make(map[string][]types.ImageLocation)
				for _, img := range scenario.ExpectedImages {
					expectedImageMap[img.Name] = img.ImageLocations
				}

				if len(result) != len(scenario.ExpectedImages) {
					t.Errorf("Expected %d images but got %d", len(scenario.ExpectedImages), len(result))
				}

				for _, img := range result {
					expectedLocations, exists := expectedImageMap[img.Name]
					if !exists {
						t.Errorf("Unexpected image found: %s", img.Name)
						continue
					}

					if !reflect.DeepEqual(img.ImageLocations, expectedLocations) {
						t.Errorf("Image locations mismatch for image '%s'", img.Name)
					}

					// Remove processed image from map
					delete(expectedImageMap, img.Name)
				}

				// Check for any remaining expected images
				if len(expectedImageMap) > 0 {
					for name := range expectedImageMap {
						t.Errorf("Expected image not found: %s", name)
					}
				}
			}
		})
	}
}

func TestExtractFiles(t *testing.T) {
	extractor := NewImagesExtractor()

	scenarios := []struct {
		Name                 string
		InputPath            string
		ExpectedFiles        types.FileImages
		ExpectedSettingFiles map[string]map[string]string
		ExpectedErrString    string
	}{
		{
			Name:      "FolderInput",
			InputPath: "../../test_files/imageExtraction",
			ExpectedFiles: types.FileImages{
				Dockerfile: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile", RelativePath: "dockerfiles/Dockerfile"},
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile-2", RelativePath: "dockerfiles/Dockerfile-2"},
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile-3", RelativePath: "dockerfiles/Dockerfile-3"},
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile-4", RelativePath: "dockerfiles/Dockerfile-4"},
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile-5", RelativePath: "dockerfiles/Dockerfile-5"},
					{FullPath: "../../test_files/imageExtraction/dockerfiles/Dockerfile.ubi9", RelativePath: "dockerfiles/Dockerfile.ubi9"},
				},
				DockerCompose: []types.FilePath{
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose.yaml", RelativePath: "dockerCompose/docker-compose.yaml"},
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose-2.yaml", RelativePath: "dockerCompose/docker-compose-2.yaml"},
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose-3.yaml", RelativePath: "dockerCompose/docker-compose-3.yaml"},
					{FullPath: "../../test_files/imageExtraction/dockerCompose/docker-compose-4.yaml", RelativePath: "dockerCompose/docker-compose-4.yaml"},
				},
				Helm: []types.HelmChartInfo{
					{
						Directory:  "../../test_files/imageExtraction/helm",
						ValuesFile: "helm/values.yaml",
						TemplateFiles: []types.FilePath{
							{FullPath: "../../test_files/imageExtraction/helm/templates/containers-worker.yaml", RelativePath: "helm/templates/containers-worker.yaml"},
							{FullPath: "../../test_files/imageExtraction/helm/templates/image-insights.yaml", RelativePath: "helm/templates/image-insights.yaml"},
						},
					},
				},
			},
			ExpectedSettingFiles: map[string]map[string]string{
				"../../test_files/imageExtraction/env":          {"IMAGE": "DEF", "TAG": "2.3.4"},
				"../../test_files/imageExtraction/env/sub-fold": {"IMAGE": "XYZ", "TAG": "3.3.3"},
			},
			ExpectedErrString: "",
		},
		{
			Name:      "TarInput",
			InputPath: "../../test_files/withDockerInTar.tar.gz",
			ExpectedFiles: types.FileImages{
				Dockerfile: []types.FilePath{
					{FullPath: "../../test_files/extracted_tar/withDockerInTar/Dockerfile", RelativePath: "withDockerInTar/Dockerfile"},
					{FullPath: "../../test_files/extracted_tar/withDockerInTar/integrationTests/Dockerfile", RelativePath: "withDockerInTar/integrationTests/Dockerfile"},
				},
				DockerCompose: []types.FilePath{
					{FullPath: "../../test_files/extracted_tar/withDockerInTar/docker-compose.yaml", RelativePath: "withDockerInTar/docker-compose.yaml"},
				},
			},
			ExpectedErrString: "",
		},
		{
			Name:      "ZipInput",
			InputPath: "../../test_files/withDockerInZip.zip",
			ExpectedFiles: types.FileImages{
				Dockerfile: []types.FilePath{
					{FullPath: "../../test_files/extracted_zip/Dockerfile", RelativePath: "Dockerfile"},
					{FullPath: "../../test_files/extracted_zip/integrationTests/Dockerfile", RelativePath: "integrationTests/Dockerfile"},
				},
				DockerCompose: []types.FilePath{
					{FullPath: "../../test_files/extracted_zip/docker-compose.yaml", RelativePath: "docker-compose.yaml"},
				},
			},
			ExpectedErrString: "",
		},
	}

	// Run test scenarios
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Run the function
			files, settingsFiles, _, err := extractor.ExtractFiles(scenario.InputPath)

			// Check for errors
			if scenario.ExpectedErrString != "" {
				if err == nil || !strings.Contains(err.Error(), scenario.ExpectedErrString) {
					t.Errorf("Expected error containing '%s' but got '%v'", scenario.ExpectedErrString, err)
				}
			} else {
				if !CompareDockerfiles(files.Dockerfile, scenario.ExpectedFiles.Dockerfile) {
					t.Errorf("Extracted Dockerfiles mismatch for scenario '%s'", scenario.Name)
				}
				if !CompareDockerCompose(files.DockerCompose, scenario.ExpectedFiles.DockerCompose) {
					t.Errorf("Extracted Docker Compose files mismatch for scenario '%s'", scenario.Name)
				}
				if !CompareHelm(files.Helm, scenario.ExpectedFiles.Helm) {
					t.Errorf("Extracted Helm charts mismatch for scenario '%s'", scenario.Name)
				}
				if scenario.Name == "FolderInput" {
					if !CompareSettingsFiles(settingsFiles, scenario.ExpectedSettingFiles) {
						t.Errorf("Extracted Settings files charts mismatch for scenario '%s'", scenario.Name)
					}
				}
			}
		})
	}
}

func CompareDockerfiles(a, b []types.FilePath) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool {
		return a[i].RelativePath < a[j].RelativePath
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].RelativePath < b[j].RelativePath
	})
	for i := range a {
		if a[i].FullPath != b[i].FullPath {
			return false
		}
		if a[i].RelativePath != b[i].RelativePath {
			return false
		}
	}
	return true
}

// CompareDockerCompose compares two slices of FilePath.
func CompareDockerCompose(a, b []types.FilePath) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool {
		return a[i].RelativePath < a[j].RelativePath
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].RelativePath < b[j].RelativePath
	})
	for i := range a {
		if a[i].FullPath != b[i].FullPath {
			return false
		}
		if a[i].RelativePath != b[i].RelativePath {
			return false
		}
	}
	return true
}

// CompareHelm compares two slices of HelmChartInfo.
func CompareHelm(a, b []types.HelmChartInfo) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort slices by directory to ensure consistent comparison
	sort.Slice(a, func(i, j int) bool {
		return a[i].Directory < a[j].Directory
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].Directory < b[j].Directory
	})

	// Iterate over each HelmChartInfo struct
	for i := range a {
		// Compare Directory and ValuesFile
		if a[i].Directory != b[i].Directory || a[i].ValuesFile != b[i].ValuesFile {
			return false
		}

		// Compare TemplateFiles slices
		if len(a[i].TemplateFiles) != len(b[i].TemplateFiles) {
			return false
		}

		// Sort TemplateFiles slices by RelativePath for consistent comparison
		sort.Slice(a[i].TemplateFiles, func(j, k int) bool {
			return a[i].TemplateFiles[j].RelativePath < a[i].TemplateFiles[k].RelativePath
		})
		sort.Slice(b[i].TemplateFiles, func(j, k int) bool {
			return b[i].TemplateFiles[j].RelativePath < b[i].TemplateFiles[k].RelativePath
		})

		// Iterate over each FilePath struct in TemplateFiles slice
		for j := range a[i].TemplateFiles {
			// Compare FullPath and RelativePath of each FilePath struct
			if a[i].TemplateFiles[j].FullPath != b[i].TemplateFiles[j].FullPath || a[i].TemplateFiles[j].RelativePath != b[i].TemplateFiles[j].RelativePath {
				return false
			}
		}
	}
	return true
}

func CompareSettingsFiles(a, b map[string]map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, subMapA := range a {
		subMapB, exists := b[key]
		if !exists {
			return false
		}
		if len(subMapA) != len(subMapB) {
			return false
		}
		for subKey, valueA := range subMapA {
			valueB, exists := subMapB[subKey]

			if !exists || valueA != valueB {
				return false
			}
		}
	}
	return true
}

// TestDockerComposeExtractorWithLineNumbersAndIndices tests the new Docker Compose extractor that provides accurate line numbers and character indices
func TestDockerComposeExtractorWithLineNumbersAndIndices(t *testing.T) {
	// Test the new extractor that uses yaml.Node and provides line numbers
	filePath := types.FilePath{
		FullPath:     "../../test_files/imageExtraction/dockerCompose/docker-compose-4.yaml",
		RelativePath: "docker-compose-4.yml",
	}

	result, err := extractors.ExtractImagesWithLineNumbersFromDockerComposeFile(filePath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that we got the expected image
	if len(result) != 1 {
		t.Errorf("Expected 1 image, got %d", len(result))
	}

	img := result[0]
	if img.Name != "mcr.microsoft.com/dotnet/sdk:6.0" {
		t.Errorf("Expected image name 'mcr.microsoft.com/dotnet/sdk:6.0', got '%s'", img.Name)
	}

	if len(img.ImageLocations) != 1 {
		t.Errorf("Expected 1 location, got %d", len(img.ImageLocations))
	}

	loc := img.ImageLocations[0]

	// Test all properties of the new extractor
	if loc.Origin != types.DockerComposeFileOrigin {
		t.Errorf("Expected origin %v, got %v", types.DockerComposeFileOrigin, loc.Origin)
	}
	if loc.Path != "docker-compose-4.yml" {
		t.Errorf("Expected path 'docker-compose-4.yml', got '%s'", loc.Path)
	}
	if loc.Line < 0 {
		t.Errorf("Expected 0-based line number >= 0, got %d", loc.Line)
	}

	expectedLine := 4 // Line 5 in file = line 4 in 0-based indexing
	if loc.Line != expectedLine {
		t.Errorf("Expected line number %d, got %d", expectedLine, loc.Line)
	}

	expectedStartIndex := 11
	expectedEndIndex := 43 // End index is exclusive, so it's the position after the last character

	if loc.StartIndex != expectedStartIndex {
		t.Errorf("Expected start index %d, got %d", expectedStartIndex, loc.StartIndex)
	}
	if loc.EndIndex != expectedEndIndex {
		t.Errorf("Expected end index %d, got %d", expectedEndIndex, loc.EndIndex)
	}

	t.Logf("Image '%s' found at line %d, indices [%d:%d]", img.Name, loc.Line, loc.StartIndex, loc.EndIndex)
}
