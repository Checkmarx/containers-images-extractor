package extractors

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Checkmarx/containers-types/types"
	"github.com/rs/zerolog/log"
)

func ExtractImagesFromDockerfiles(filePaths []types.FilePath, envFiles map[string]map[string]string) ([]types.ImageModel, error) {
	var imageNames []types.ImageModel

	for _, filePath := range filePaths {
		log.Debug().Msgf("going to extract images from dockerfile %s", filePath)

		fileImages, err := extractImagesFromDockerfile(filePath, envFiles)
		if err != nil {
			log.Warn().Msgf("could not extract images from dockerfile %s err: %+v", filePath, err)
		}
		printFoundImagesInFile(filePath.RelativePath, fileImages)
		imageNames = append(imageNames, fileImages...)
	}

	return imageNames, nil
}

func extractImagesFromDockerfile(filePath types.FilePath, envFiles map[string]map[string]string) ([]types.ImageModel, error) {
	var imageNames []types.ImageModel
	aliases := make(map[string]string)
	argsAndEnv := make(map[string]string)
	mergedEnvVars := resolveEnvVariables(filePath.FullPath, envFiles)

	file, err := os.Open(filePath.FullPath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Warn().Msgf("Could not close dockerfile: %s err: %+v", file.Name(), err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	lineNum := -1 // Start from -1 so first line becomes 0 (0-based indexing)
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++ // Increment at the beginning to ensure it's always updated

		// Parse ARG and ENV lines within the Dockerfile
		if match := regexp.MustCompile(`^\s*(ARG|ENV)\s+(\w+)=([^\s]+)`).FindStringSubmatch(line); match != nil {
			varName := match[2]
			varValue := match[3]
			argsAndEnv[varName] = varValue
		}

		// Replace placeholders with values from mergedEnvVars and argsAndEnv
		line = replacePlaceholders(line, mergedEnvVars, argsAndEnv)

		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Parse FROM instructions
		if match := regexp.MustCompile(`^\s*FROM\s+(?:--platform=[^\s]+\s+)?([\w./-]+(?::[\w.-]+)?)(?:\s+AS\s+(\w+))?`).FindStringSubmatch(line); match != nil {
			imageName := match[1]
			alias := match[2]

			if imageName == "scratch" {
				continue
			}

			if alias != "" {
				realName := resolveAlias(alias, aliases)
				if realName != "" {
					aliases[alias] = realName
				} else {
					aliases[alias] = imageName
				}
			}
		}

		if match := regexp.MustCompile(`\bFROM\s+(?:--platform=[^\s]+\s+)?([\w./-]+)(?::([\w.-]+))?\b`).FindStringSubmatch(line); match != nil {
			imageName := match[1]
			tag := match[2]

			if imageName == "scratch" {
				continue
			}
			if tag == "" {
				tag = "latest"
			}

			fullImageName := fmt.Sprintf("%s:%s", imageName, tag)

			if realName, ok := aliases[imageName]; ok {
				if realName != imageName {
					continue
				}
			}
			log.Debug().Msgf("Found image %s at line %d (will assign to ImageLocation.Line/StartIndex/EndIndex)", fullImageName, lineNum)

			// Robust regex to find the start and end index of the image name in the line
			re := regexp.MustCompile(`FROM\s+([^\s]+)`)
			indices := re.FindStringSubmatchIndex(line)
			imgIdx, imgEnd := -1, -1
			if len(indices) >= 4 {
				imgIdx = indices[2]
				imgEnd = indices[3]
				fmt.Printf("DEBUG: imgIdx=%d, imgEnd=%d, matched='%s'\n", imgIdx, imgEnd, line[imgIdx:imgEnd])
				// Removed the code that extends imgEnd by the tag length
			}

			imageNames = append(imageNames, types.ImageModel{
				Name: fullImageName,
				ImageLocations: []types.ImageLocation{
					{
						Origin:     types.DockerFileOrigin,
						Path:       filePath.RelativePath,
						FinalStage: false,
						Line:       lineNum,
						StartIndex: imgIdx,
						EndIndex:   imgEnd,
					},
				},
			})
		}
	}

	if len(imageNames) > 0 {
		lastImage := imageNames[len(imageNames)-1]
		if len(lastImage.ImageLocations) > 0 {
			lastImage.ImageLocations[len(lastImage.ImageLocations)-1].FinalStage = true
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return imageNames, nil
}

func replacePlaceholders(line string, envVars, argsAndEnv map[string]string) string {
	for varName, varValue := range envVars {
		placeholderWithBraces := fmt.Sprintf("${%s}", varName)
		line = strings.ReplaceAll(line, placeholderWithBraces, varValue)

		placeholderWithoutBraces := fmt.Sprintf("$%s", varName)
		line = strings.ReplaceAll(line, placeholderWithoutBraces, varValue)
	}

	for varName, varValue := range argsAndEnv {
		placeholderWithBraces := fmt.Sprintf("${%s}", varName)
		line = strings.ReplaceAll(line, placeholderWithBraces, varValue)

		placeholderWithoutBraces := fmt.Sprintf("$%s", varName)
		line = strings.ReplaceAll(line, placeholderWithoutBraces, varValue)
	}

	return line
}

func resolveEnvVariables(dockerfilePath string, envFiles map[string]map[string]string) map[string]string {
	resolvedVars := make(map[string]string)

	dirs := getDirsForHierarchy(dockerfilePath)
	for _, dir := range dirs {
		if envVars, ok := envFiles[dir]; ok {
			for k, v := range envVars {
				if _, exists := resolvedVars[k]; !exists {
					resolvedVars[k] = v
				}
			}
		}
	}

	return resolvedVars
}

func getDirsForHierarchy(dockerfilePath string) []string {
	var dirs []string

	dir := filepath.Dir(dockerfilePath)
	for dir != "" && dir != "." && dir != "/" {

		if isRootDir(dir) {
			break
		}

		dirs = append(dirs, dir)
		dir = filepath.Dir(dir)
	}
	return dirs
}

func isRootDir(dir string) bool {
	return (len(dir) == 3 && strings.HasSuffix(dir, `:\`)) || (len(dir) >= 2 && strings.HasPrefix(dir, `\\`))
}

func resolveAlias(alias string, aliases map[string]string) string {
	realName, ok := aliases[alias]
	if !ok {
		return "" // Alias not found
	}

	// Check if the real name is also an alias, resolve recursively
	if resolvedRealName, ok := aliases[realName]; ok {
		return resolveAlias(resolvedRealName, aliases)
	}

	return realName
}

func printFoundImagesInFile(filePath string, imageNames []types.ImageModel) {
	if len(imageNames) > 0 {
		log.Debug().Msgf("Successfully found images in file: %s images are: %v\n", filePath, strings.Join(func() []string {
			var result []string
			for _, obj := range imageNames {
				result = append(result, obj.Name)
			}
			return result
		}(), ", "))

	} else {
		log.Debug().Msgf("Could not find any images in file: %s\n", filePath)
	}
}
