package imagesExtractor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Checkmarx/containers-types/types"
	"github.com/rs/zerolog/log"
)

var (
	dockerfilePattern    = regexp.MustCompile(`^Dockerfile(?:-[a-zA-Z0-9]+|\.[a-zA-Z0-9.]+|[a-zA-Z0-9.]+)?$`)
	dockerComposePattern = regexp.MustCompile(`docker-compose(-[a-zA-Z0-9]+)?(\.yml|\.yaml)$`)
)

func IsValidFolderPath(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func DeleteDirectory(dirPath string) error {
	err := os.RemoveAll(dirPath)
	if err != nil {
		return err
	}
	return nil
}

func findHelmCharts(baseDir string) ([]types.HelmChartInfo, error) {
	var helmCharts []types.HelmChartInfo

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && isHelmChart(path) {

			valuesFile := filepath.Join(path, "values.yaml")
			relativeValuesPath, _ := filepath.Rel(baseDir, valuesFile)

			templatesDir := filepath.Join(path, "templates")

			var templateFiles []types.FilePath
			err = filepath.Walk(templatesDir, func(templatePath string, templateInfo os.FileInfo, templateErr error) error {
				if templateErr != nil {
					return templateErr
				}
				if !templateInfo.IsDir() && isYAMLFile(templatePath) {
					relativeTemplatePath, _ := filepath.Rel(baseDir, templatePath)
					templateFiles = append(templateFiles, types.FilePath{
						FullPath:     templatePath,
						RelativePath: relativeTemplatePath,
					})
				}

				return nil
			})

			if err != nil {
				return err
			}

			helmChart := types.HelmChartInfo{
				Directory:     path,
				ValuesFile:    relativeValuesPath,
				TemplateFiles: templateFiles,
			}

			helmCharts = append(helmCharts, helmChart)
		}

		return nil
	})

	return helmCharts, err
}

func extractCompressedPath(inputPath string) (string, error) {
	if fileInfo, err := os.Stat(inputPath); err == nil && fileInfo.IsDir() {
		return inputPath, nil
	}

	if strings.HasSuffix(inputPath, ".zip") {
		return extractZip(inputPath)
	}

	if strings.HasSuffix(inputPath, ".tar") || strings.HasSuffix(inputPath, ".tar.gz") || strings.HasSuffix(inputPath, ".tgz") {
		return extractTar(inputPath)
	}

	return "", fmt.Errorf("unsupported file type: %s", inputPath)
}

func getRelativePath(baseDir, filePath string) string {
	relativePath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		return filePath
	}
	return relativePath
}

func isYAMLFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".yml" || ext == ".yaml"
}

func isHelmChart(directory string) bool {
	chartFilePath := filepath.Join(directory, "Chart.yaml")
	valuesFilePath := filepath.Join(directory, "values.yaml")
	templatesDirPath := filepath.Join(directory, "templates")

	_, errChart := os.Stat(chartFilePath)
	_, errValues := os.Stat(valuesFilePath)
	_, errTemplatesDir := os.Stat(templatesDirPath)

	return errChart == nil && errValues == nil && errTemplatesDir == nil
}

func getContainerResolutionFullPath(folderPath string) (string, error) {
	return folderPath + "/containers-resolution.json", nil // Hard-coding the containers resolution filename
}

func printFilePaths(f []types.FilePath, message string) {
	if len(f) > 0 {
		log.Debug().Msgf("%s. files: %v\n", message, strings.Join(func() []string {
			var result []string
			for _, obj := range f {
				result = append(result, obj.RelativePath)
			}
			return result
		}(), ", "))
	}
}
