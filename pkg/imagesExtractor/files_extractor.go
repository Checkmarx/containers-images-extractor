package imagesExtractor

import (
	"bufio"
	"encoding/json"
	"github.com/Checkmarx/containers-images-extractor/internal/extractors"
	"github.com/Checkmarx/containers-types/types"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"strings"
)

type ImagesExtractorInterface interface {
	ExtractAndMergeImagesFromFiles(files types.FileImages, images []types.ImageModel,
		settingsFiles map[string]map[string]string) ([]types.ImageModel, error)
	ExtractFiles(scanPath string) (types.FileImages, map[string]map[string]string, string, error)
	SaveObjectToFile(folderPath string, obj interface{}) error
}

type ImagesExtractor struct {
}

func (e ImagesExtractor) ExtractAndMergeImagesFromFiles(files types.FileImages, images []types.ImageModel,
	settingsFiles map[string]map[string]string) ([]types.ImageModel, error) {
	dockerfileImages, err := extractors.ExtractImagesFromDockerfiles(files.Dockerfile, settingsFiles)
	if err != nil {
		log.Err(err).Msg("Could not extract images from docker files")
		return nil, err
	}

	dockerComposeFileImages, err := extractors.ExtractImagesFromDockerComposeFiles(files.DockerCompose, settingsFiles)
	if err != nil {
		log.Err(err).Msg("Could not extract images from docker compose files")
		return nil, err
	}

	helmImages, extErr := extractors.ExtractImagesFromHelmFiles(files.Helm)
	if extErr != nil {
		log.Err(extErr).Msg("Could not extract images from helm files")
		return nil, extErr
	}

	imagesFromFiles := mergeImages(images, dockerfileImages, dockerComposeFileImages, helmImages)

	return imagesFromFiles, nil
}

func (e ImagesExtractor) ExtractFiles(scanPath string) (types.FileImages, map[string]map[string]string, string, error) {
	filesPath, err := extractCompressedPath(scanPath)
	if err != nil {
		log.Err(err).Msgf("Could not extract compressed folder")
		return types.FileImages{}, nil, scanPath, err
	}

	var f types.FileImages
	envFiles := make(map[string][]string)

	err = filepath.Walk(filesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the current path matches the Dockerfile pattern
		if dockerfilePattern.MatchString(info.Name()) {
			f.Dockerfile = append(f.Dockerfile, types.FilePath{
				FullPath:     path,
				RelativePath: getRelativePath(filesPath, path),
			})
		}

		// Check if the current path matches the Docker Compose file pattern
		if dockerComposePattern.MatchString(info.Name()) {
			f.DockerCompose = append(f.DockerCompose, types.FilePath{
				FullPath:     path,
				RelativePath: getRelativePath(filesPath, path),
			})
		}

		if strings.HasSuffix(info.Name(), ".env") || strings.HasSuffix(info.Name(), ".env_cxcontainers") {
			dir := filepath.Dir(path)
			envFiles[dir] = append(envFiles[dir], path)
		}

		return nil
	})

	if err != nil {
		log.Warn().Msgf("Could not extract docker or docker compose files: %s", err.Error())
	}

	helmCharts, err := findHelmCharts(filesPath)
	if err != nil {
		log.Warn().Msgf("Could not extract helm charts: %s", err.Error())
	}
	if len(helmCharts) > 0 {
		f.Helm = helmCharts
	}
	printFilePaths(f.Dockerfile, "Successfully found dockerfiles")
	printFilePaths(f.DockerCompose, "Successfully found docker compose files")

	envVars := parseEnvFiles(envFiles)
	return f, envVars, filesPath, nil
}

func parseEnvFiles(envFiles map[string][]string) map[string]map[string]string {
	envVars := make(map[string]map[string]string)

	for dir, files := range envFiles {
		for _, file := range files {
			fileVars, err := parseEnvFile(file)
			if err != nil {
				continue // skip on error
			}
			if envVars[dir] == nil {
				envVars[dir] = make(map[string]string)
			}
			for k, v := range fileVars {
				envVars[dir][k] = v
			}
		}
	}

	return envVars
}

func parseEnvFile(filePath string) (map[string]string, error) {
	envVars := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Warn().Msgf("Could not close env file: %s err: %+v", file.Name(), err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envVars, nil
}

func (e ImagesExtractor) SaveObjectToFile(folderPath string, obj interface{}) error {
	containerResolutionFullPath, err := getContainerResolutionFullPath(folderPath)
	if err != nil {
		log.Err(err).Msgf("Error getting container resolution full file path")
		return err
	}
	log.Debug().Msgf("containers-resolution.json full path is: %s", containerResolutionFullPath)

	resultBytes, err := json.Marshal(obj)
	if err != nil {
		log.Err(err).Msgf("Error marshaling struct")
		return err
	}

	err = os.WriteFile(containerResolutionFullPath, resultBytes, 0644)
	if err != nil {
		log.Err(err).Msgf("Error writing file")
		return err
	}
	return nil
}
