package extractors

import (
	"bufio"
	"fmt"
	"regexp"

	"github.com/Checkmarx/containers-types/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"

	"os"
	"path/filepath"
	"strings"
)

func ExtractImagesFromHelmFiles(helmCharts []types.HelmChartInfo) ([]types.ImageModel, error) {

	var imagesFromHelmDirectories []types.ImageModel
	for _, h := range helmCharts {
		log.Info().Msgf("going to extract images from helm directory %s", h.Directory)

		renderedTemplates, err := generateRenderedTemplates(h)
		if err != nil {
			log.Err(err).Msgf("Could not render templates from helm directory %s", h.Directory)
			continue
		}

		images, err := extractImageInfo(renderedTemplates)
		if err != nil {
			log.Err(err).Msgf("Could not extract images from helm directory %s", h.Directory)
			continue
		}

		printFoundImages(h, images)
		imagesFromHelmDirectories = append(imagesFromHelmDirectories, images...)
	}

	return imagesFromHelmDirectories, nil
}

func printFoundImages(h types.HelmChartInfo, images []types.ImageModel) {
	log.Debug().Msgf("Found images in helm directory: %s, images: %v", h.Directory, strings.Join(func() []string {
		var result []string
		for _, obj := range images {
			result = append(result, obj.Name)
		}
		return result
	}(), ", "))
}

func generateRenderedTemplates(c types.HelmChartInfo) (string, error) {
	actionConfig := new(action.Configuration)

	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = "temp-release"
	client.ClientOnly = true

	chartPath, err := filepath.Abs(c.Directory)
	if err != nil {
		return "", err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", err
	}

	release, err := client.Run(chart, nil)
	if err != nil {
		return "", err
	}

	return release.Manifest, nil
}

func extractImageInfo(yamlString string) ([]types.ImageModel, error) {
	sections := strings.Split(yamlString, "---")

	var imageInfoList []types.ImageModel

	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}

		var microservice types.Microservice
		err := yaml.Unmarshal([]byte(section), &microservice)
		if err != nil {
			return nil, err
		}

		s, _ := extractSource(section)
		n := extractImageName(microservice)

		if n != "" {
			v := types.ImageModel{
				Name: n,
				ImageLocations: []types.ImageLocation{
					{
						Origin: types.HelmFileOrigin,
						Path:   s,
					},
				},
			}

			imageInfoList = append(imageInfoList, v)
		}
	}

	return imageInfoList, nil
}

func extractImageName(microservice types.Microservice) string {
	var imageName string
	if microservice.Spec.Image.Registry != "" {
		imageName += microservice.Spec.Image.Registry + "/"
	}

	if microservice.Spec.Image.Name == "" {
		return ""
	}
	imageName += microservice.Spec.Image.Name + ":"
	imageName += microservice.Spec.Image.Tag

	return imageName
}

func extractSource(yamlBlock string) (string, error) {
	sourceRegex := regexp.MustCompile(`#\s*Source:\s*([^\n]+)`)
	match := sourceRegex.FindStringSubmatch(yamlBlock)

	if len(match) != 2 {
		return "", fmt.Errorf("source not found in YAML block")
	}

	source := strings.TrimSpace(match[1])
	return source, nil
}

// ExtractImagesWithLineNumbersFromHelmFiles extracts image references with line numbers and character indices from Helm template and values files.
func ExtractImagesWithLineNumbersFromHelmFiles(helmCharts []types.HelmChartInfo) ([]types.ImageModel, error) {
	var imagesFromHelmDirectories []types.ImageModel
	imagePattern := regexp.MustCompile(`(?m)^\s*image:\s*([^\s#]+:[^\s#]+)\s*$`)

	for _, chart := range helmCharts {
		// Process template files recursively
		for _, templateFile := range chart.TemplateFiles {
			fileImages, err := extractImagesWithLineInfoFromFile(templateFile.RelativePath, templateFile.FullPath, imagePattern)
			if err != nil {
				log.Err(err).Msgf("Could not extract images with line info from template file: %s", templateFile.FullPath)
				continue
			}
			imagesFromHelmDirectories = append(imagesFromHelmDirectories, fileImages...)
		}

		// Process values files (values.yaml, values-*.yaml, values.yml, values-*.yml)
		valuesFiles, err := findValuesFiles(chart.Directory)
		if err != nil {
			log.Err(err).Msgf("Could not find values files in chart directory: %s", chart.Directory)
			continue
		}
		for _, valuesFile := range valuesFiles {
			fileImages, err := extractImagesWithLineInfoFromFile(valuesFile.RelativePath, valuesFile.FullPath, imagePattern)
			if err != nil {
				log.Err(err).Msgf("Could not extract images with line info from values file: %s", valuesFile.FullPath)
				continue
			}
			imagesFromHelmDirectories = append(imagesFromHelmDirectories, fileImages...)
		}
	}
	return imagesFromHelmDirectories, nil
}

// extractImagesWithLineInfoFromFile scans a file for image references and returns ImageModels with line and index info.
func extractImagesWithLineInfoFromFile(relativePath, fullPath string, imagePattern *regexp.Regexp) ([]types.ImageModel, error) {
	var images []types.ImageModel
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lineNum++
			continue
		}
		matches := imagePattern.FindStringSubmatchIndex(line)
		if matches != nil && len(matches) >= 4 {
			// matches[2] and matches[3] are the start and end indices of the image reference
			imageRef := line[matches[2]:matches[3]]
			images = append(images, types.ImageModel{
				Name: imageRef,
				ImageLocations: []types.ImageLocation{{
					Origin:     types.HelmFileOrigin,
					Path:       relativePath,
					Line:       lineNum,
					StartIndex: matches[2],
					EndIndex:   matches[3],
				}},
			})
		}
		lineNum++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return images, nil
}

// findValuesFiles returns all values.yaml, values-*.yaml, values.yml, values-*.yml files in the chart directory.
func findValuesFiles(chartDir string) ([]types.FilePath, error) {
	var valuesFiles []types.FilePath
	patterns := []string{"values.yaml", "values-*.yaml", "values.yml", "values-*.yml"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(chartDir, pattern))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			valuesFiles = append(valuesFiles, types.FilePath{
				FullPath:     match,
				RelativePath: filepath.Base(match),
			})
		}
	}
	return valuesFiles, nil
}
