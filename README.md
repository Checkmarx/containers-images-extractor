# Containers Images Extractor 

This project provides tools for extracting and merging image data from Dockerfiles, Docker Compose files, and Helm charts. The extracted image data can be saved to a file for further analysis or use. 

## Table of Contents
1. [Overview](#overview)
2. [Features](#features)
3. [Installation](#installation)
4. [Usage](#usage)
5. [Contributing](#contributing)
6. [License](#license)

## Overview

The Containers Images Extractor helps in extracting container images from various configuration files used in container orchestration. This includes Dockerfiles, Docker Compose files, and Helm charts.

## Features

- Extract images from Dockerfiles, Docker Compose files, and Helm charts.
- Merge extracted images with existing image lists.
- Save image data to JSON files for further use.

## Installation

To install this package, you need to have [Go](https://golang.org/doc/install) installed on your machine.

1. Clone the repository:
    ```sh
    git clone https://github.com/Checkmarx/containers-images-extractor.git
    ```

2. Navigate to the project directory:
    ```sh
    cd containers-images-extractor
    ```

3. Install dependencies:
    ```sh
    go mod tidy
    ```

## Usage

Here is an example of how to use the `ImagesExtractor`:

```go
package main

import (
    "github.com/Checkmarx/containers-images-extractor/imagesExtractor"
    "github.com/Checkmarx/containers-types/types"
    "log"
)

func main() {
    extractor := imagesExtractor.ImagesExtractor{}
    scanPath := "/path/to/scan"

    // Extract files
    files, envVars, extractedPath, err := extractor.ExtractFiles(scanPath)
    if err != nil {
        log.Fatalf("Error extracting files: %v", err)
    }

    // Merge images
    images, err := extractor.ExtractAndMergeImagesFromFiles(files, []types.ImageModel{}, envVars)
    if err != nil {
        log.Fatalf("Error merging images: %v", err)
    }

    // Save to file
    err = extractor.SaveObjectToFile(extractedPath, images)
    if err != nil {
        log.Fatalf("Error saving images to file: %v", err)
    }
}
