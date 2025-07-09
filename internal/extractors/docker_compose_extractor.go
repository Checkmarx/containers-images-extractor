package extractors

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"strings"

	"github.com/Checkmarx/containers-types/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
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

// calculateYAMLIndices calculates the start and end index of a value in a specific line of a YAML file
func calculateYAMLIndices(filePath string, lineNum int, value string) (startIdx, endIdx int) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Debug().Msgf("Could not open file %s for index calculation: %v", filePath, err)
		return 0, 0 // fallback
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentLine := 1

	for scanner.Scan() {
		if currentLine == lineNum {
			lineText := scanner.Text()
			startIdx = strings.Index(lineText, value)
			if startIdx != -1 {
				endIdx = startIdx + len(value)
				return startIdx, endIdx
			}
			break
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		log.Debug().Msgf("Error scanning file %s for index calculation: %v", filePath, err)
	}

	return 0, 0 // fallback if not found
}

// ExtractImagesWithLineNumbersFromDockerComposeFile extracts images and their line numbers using yaml.Node.
func ExtractImagesWithLineNumbersFromDockerComposeFile(filePath types.FilePath) ([]types.ImageModel, error) {
	var imageNames []types.ImageModel

	file, err := os.Open(filePath.FullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var root yaml.Node
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}

	// Find the "services" mapping node
	for i := 0; i < len(root.Content); i++ {
		node := root.Content[i]
		if node.Kind == yaml.MappingNode {
			for j := 0; j < len(node.Content); j += 2 {
				key := node.Content[j]
				value := node.Content[j+1]
				if key.Value == "services" && value.Kind == yaml.MappingNode {
					// Iterate over services
					for k := 0; k < len(value.Content); k += 2 {
						serviceName := value.Content[k].Value
						serviceNode := value.Content[k+1]
						// Find "image" field in this service
						for l := 0; l < len(serviceNode.Content); l += 2 {
							fieldKey := serviceNode.Content[l]
							fieldValue := serviceNode.Content[l+1]
							if fieldKey.Value == "image" {
								log.Debug().Msgf("Found image %s for service %s at line %d", fieldValue.Value, serviceName, fieldValue.Line)

								// Calculate start and end indices
								startIdx, endIdx := calculateYAMLIndices(filePath.FullPath, fieldValue.Line, fieldValue.Value)
								if startIdx == 0 && endIdx == 0 {
									log.Debug().Msgf("Could not calculate indices for image %s at line %d, using fallback", fieldValue.Value, fieldValue.Line)
								}

								imageNames = append(imageNames, types.ImageModel{
									Name: fieldValue.Value,
									ImageLocations: []types.ImageLocation{
										{
											Origin:     types.DockerComposeFileOrigin,
											Path:       filePath.RelativePath,
											Line:       fieldValue.Line,
											StartIndex: startIdx,
											EndIndex:   endIdx,
										},
									},
								})
							}
						}
					}
				}
			}
		}
	}
	return imageNames, nil
}
