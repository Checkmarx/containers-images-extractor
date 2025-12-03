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
			relativeValuesPath = filepath.ToSlash(relativeValuesPath)

			templatesDir := filepath.Join(path, "templates")

			var templateFiles []types.FilePath
			err = filepath.Walk(templatesDir, func(templatePath string, templateInfo os.FileInfo, templateErr error) error {
				if templateErr != nil {
					return templateErr
				}
				if !templateInfo.IsDir() && isYAMLFile(templatePath) {
					relativeTemplatePath, _ := filepath.Rel(baseDir, templatePath)
					relativeTemplatePath = filepath.ToSlash(relativeTemplatePath)
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
	// Normalize path separator to forward slashes for cross-platform consistency
	return filepath.ToSlash(relativePath)
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

func findHelmFilesInDirectory(scanPath string) ([]types.HelmChartInfo, error) {
	// Validate scan path and find helm directory
	helmDir, err := validateScanPathAndFindHelmDir(scanPath)
	if err != nil {
		return nil, err
	}

	// Collect all valid Helm files
	valuesFiles, templateFiles, err := collectHelmFiles(helmDir, helmDir)
	if err != nil {
		return nil, err
	}

	return createHelmChartInfo(helmDir, valuesFiles, templateFiles), nil
}

func validateScanPathAndFindHelmDir(scanPath string) (string, error) {
	// Check if directory exists
	if _, err := os.Stat(scanPath); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", scanPath)
	}

	helmDir, isUnderHelm := findHelmDirectory(scanPath)
	if !isUnderHelm {
		return "", fmt.Errorf("no helm directory found in path hierarchy: %s", scanPath)
	}

	relativeScanPath, err := filepath.Rel(helmDir, scanPath)
	if err != nil {
		return "", fmt.Errorf("could not get relative path: %v", err)
	}

	fullScanPath := filepath.Join(helmDir, relativeScanPath)

	scanPathInfo, err := os.Stat(fullScanPath)
	if err != nil {
		return "", fmt.Errorf("could not get scan path info: %v", err)
	}
	if !scanPathInfo.IsDir() {
		return "", fmt.Errorf("scan path must be a directory: %s", scanPath)
	}

	return helmDir, nil
}

func collectHelmFiles(scanPath, helmDir string) ([]string, []types.FilePath, error) {
	var valuesFiles []string
	var templateFiles []types.FilePath

	err := filepath.Walk(helmDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !isYAMLFile(path) {
			return nil
		}

		if info.Name() == "Chart.yaml" {
			return nil
		}

		relativePath, err := filepath.Rel(helmDir, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)

		if isValuesFile(path, helmDir) {
			valuesFiles = append(valuesFiles, relativePath)
		} else if isTemplateFile(relativePath) {
			templateFiles = append(templateFiles, types.FilePath{
				FullPath:     path,
				RelativePath: relativePath,
			})
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error walking directory: %v", err)
	}

	return valuesFiles, templateFiles, nil
}

func isValuesFile(filePath, helmDir string) bool {
	fileDir := filepath.Dir(filePath)
	isDirectlyUnderHelm := filepath.Clean(fileDir) == filepath.Clean(helmDir)

	if !isDirectlyUnderHelm {
		return false
	}

	fileName := filepath.Base(filePath)
	return strings.Contains(strings.ToLower(fileName), "values")
}

func isTemplateFile(relativePath string) bool {
	relativeDir := filepath.Dir(relativePath)
	return strings.HasPrefix(relativeDir, "templates") || relativeDir == "templates"
}

func createHelmChartInfo(helmDir string, valuesFiles []string, templateFiles []types.FilePath) []types.HelmChartInfo {
	if len(valuesFiles) == 0 && len(templateFiles) == 0 {
		return []types.HelmChartInfo{}
	}

	helmChart := types.HelmChartInfo{
		Directory: helmDir,
	}

	// Add values files (take the first one as ValuesFile, others could be added to TemplateFiles if needed)
	if len(valuesFiles) > 0 {
		helmChart.ValuesFile = valuesFiles[0]
		// Add remaining values files to template files for completeness
		for i := 1; i < len(valuesFiles); i++ {
			templateFiles = append(templateFiles, types.FilePath{
				FullPath:     filepath.Join(helmDir, valuesFiles[i]),
				RelativePath: valuesFiles[i],
			})
		}
	}

	helmChart.TemplateFiles = templateFiles

	return []types.HelmChartInfo{helmChart}
}

func findHelmDirectory(dirPath string) (string, bool) {
	currentDir := dirPath

	for {
		dirName := filepath.Base(currentDir)
		if isHelmDirectoryName(dirName) {
			return currentDir, true
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root directory
			break
		}
		currentDir = parentDir
	}

	return "", false
}

func isHelmDirectoryName(dirName string) bool {
	dirNameLower := strings.ToLower(dirName)

	return strings.Contains(dirNameLower, "helm")
}
