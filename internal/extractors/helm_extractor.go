package extractors

import (
	"fmt"
	"github.com/Checkmarx/containers-types/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"regexp"

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
