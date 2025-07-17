package imagesExtractor

import "github.com/Checkmarx/containers-types/types"

func mergeImages(images, imagesFromDockerFiles, imagesFromDockerComposeFiles, helmImages []types.ImageModel) []types.ImageModel {
	if len(imagesFromDockerFiles) > 0 {
		images = append(images, imagesFromDockerFiles...)
	}
	if len(imagesFromDockerComposeFiles) > 0 {
		images = append(images, imagesFromDockerComposeFiles...)
	}
	if len(helmImages) > 0 {
		images = append(images, helmImages...)
	}
	return mergeDuplicates(images)
}

func mergeDuplicates(imageModels []types.ImageModel) []types.ImageModel {
	aggregated := make(map[string][]types.ImageLocation)
	var result []types.ImageModel

	for _, img := range imageModels {
		if _, ok := aggregated[img.Name]; !ok {
			// If the image name is not yet in the result, add it with its locations and IsSha value
			result = append(result, types.ImageModel{Name: img.Name, ImageLocations: img.ImageLocations, IsSha: img.IsSha})
			aggregated[img.Name] = img.ImageLocations
		} else {
			// If the image name is already in the result, merge the locations
			for _, location := range img.ImageLocations {
				found := false
				for _, existingLocation := range aggregated[img.Name] {
					if isSameLocation(location, existingLocation) {
						found = true
						break
					}
				}
				if !found {
					// Append only new locations to the existing entry in the result slice
					for i := range result {
						if result[i].Name == img.Name {
							result[i].ImageLocations = append(result[i].ImageLocations, location)
							break
						}
					}
					// Update the map to include the new location
					aggregated[img.Name] = append(aggregated[img.Name], location)
				}
			}
		}
	}

	return result
}

func isSameLocation(loc1, loc2 types.ImageLocation) bool {
	return loc1.Origin == loc2.Origin && loc1.Path == loc2.Path &&
		loc1.FinalStage == loc2.FinalStage && loc1.Line == loc2.Line &&
		loc1.StartIndex == loc2.StartIndex && loc1.EndIndex == loc2.EndIndex
}
