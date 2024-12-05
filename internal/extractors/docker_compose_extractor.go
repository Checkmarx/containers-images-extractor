package extractors

import (
	"bufio"
	"fmt"
	"github.com/Checkmarx/containers-types/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"regexp"
)

// Service represents a service in docker-compose
type Service struct {
	Image string `yaml:"image"`
	Build *Build `yaml:"build"`
}

// Build represents the build context in docker-compose
type Build struct {
	Context string `yaml:"context"`
}

// ComposeFile represents a docker-compose file structure
type ComposeFile struct {
	Services map[string]Service `yaml:"services"`
}

func ExtractImagesFromDockerComposeFiles(filePaths []types.FilePath, envFiles map[string]map[string]string) ([]types.ImageModel, error) {
	var imageNames []types.ImageModel

	for _, filePath := range filePaths {
		log.Debug().Msgf("going to extract images from docker compose file %s", filePath)
		fileImages, err := extractImagesFromDockerComposeFile(filePath, envFiles)
		if err != nil {
			log.Warn().Msgf("could not extract images from docker compose file %s err: %+v", filePath, err)
		}
		printFoundImagesInFile(filePath.RelativePath, fileImages)
		imageNames = append(imageNames, fileImages...)
	}

	return imageNames, nil
}

func extractImagesFromDockerComposeFile(filePath types.FilePath, envFiles map[string]map[string]string) ([]types.ImageModel, error) {
	var imageNames []types.ImageModel

	file, err := os.Open(filePath.FullPath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Warn().Msgf("Could not close docker compose file: %s err: %+v", file.Name(), err)
		}
	}(file)

	var compose ComposeFile
	decoder := yaml.NewDecoder(bufio.NewReader(file))
	err = decoder.Decode(&compose)
	if err != nil {
		log.Err(err).Msg("Error parsing docker-compose file")
		return nil, err
	}

	mergedEnvVars := resolveEnvVariables(filePath.FullPath, envFiles)

	// Regex pattern
	pattern := `^([^:@\s]+)(?::([^@\s]+))?$`
	re := regexp.MustCompile(pattern)

	for serviceName, service := range compose.Services {
		if service.Image != "" {
			fmt.Printf("Service: %s, Image: %s\n", serviceName, service.Image)
		} else if service.Build != nil && service.Build.Context != "" {
			fmt.Printf("Service: %s, Build Context: %s (no image specified)\n", serviceName, service.Build.Context)
		} else {
			fmt.Printf("Service: %s, No image or build context specified\n", serviceName)
		}

		fullImageName := processEnvVars(service.Image, mergedEnvVars)

		if match := re.FindStringSubmatch(fullImageName); match != nil {
			imageName := match[1]
			tag := match[2]

			if tag == "" {
				tag = "latest"
			}

			fullImageName = fmt.Sprintf("%s:%s", imageName, tag)
		}

		imageNames = append(imageNames, types.ImageModel{
			Name: fullImageName,
			ImageLocations: []types.ImageLocation{
				{
					Origin: types.DockerComposeFileOrigin,
					Path:   filePath.RelativePath,
				},
			},
		})
	}

	return imageNames, nil
}

func processEnvVars(extractedImageId string, envVars map[string]string) string {
	for key, value := range envVars {

		pattern := `(\{\{` + regexp.QuoteMeta(key) + `\}\}|\$\{` + regexp.QuoteMeta(key) + `\})`
		pattern2 := `\$\{` + regexp.QuoteMeta(key) + `:-[^}]*\}`

		re := regexp.MustCompile(pattern)
		extractedImageId = re.ReplaceAllString(extractedImageId, value)

		re2 := regexp.MustCompile(pattern2)
		extractedImageId = re2.ReplaceAllString(extractedImageId, value)
	}

	defaultImagePattern := `:-(.+)}`
	re := regexp.MustCompile(defaultImagePattern)
	match := re.FindStringSubmatch(extractedImageId)
	if match != nil {
		return match[1]
	}

	return extractedImageId
}
