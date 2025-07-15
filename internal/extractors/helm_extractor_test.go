package extractors

import (
	"testing"

	"github.com/Checkmarx/containers-types/types"
)

func TestExtractImagesFromHelmFiles(t *testing.T) {
	t.Run("ValidHelmFiles", func(t *testing.T) {
		helmCharts := []types.HelmChartInfo{
			{Directory: "../../test_files/imageExtraction/helm"},
		}

		images, err := ExtractImagesFromHelmFiles(helmCharts)
		if err != nil {
			t.Errorf("Error extracting images: %v", err)
		}

		expectedImages := map[string]types.ImageLocation{
			"checkmarx.jfrog.io/ast-docker/containers-worker:b201b1f": {Origin: types.HelmFileOrigin, Path: "containers/templates/containers-worker.yaml"},
			"checkmarx.jfrog.io/ast-docker/image-insights:f4b507b":    {Origin: types.HelmFileOrigin, Path: "containers/templates/image-insights.yaml"},
		}

		checkHelmResult(t, images, expectedImages)
	})

	t.Run("NoHelmFilesFound", func(t *testing.T) {
		helmCharts := []types.HelmChartInfo{}

		images, err := ExtractImagesFromHelmFiles(helmCharts)
		if err != nil {
			t.Errorf("Error extracting images: %v", err)
		}

		if len(images) != 0 {
			t.Errorf("Expected 0 images, but got %d", len(images))
		}
	})

	t.Run("OneValidOneInvalidHelmFiles", func(t *testing.T) {
		helmCharts := []types.HelmChartInfo{
			{Directory: "../../test_files/imageExtraction/helm/"},
			{Directory: "../../test_files/imageExtraction/helm2/"},
		}

		images, err := ExtractImagesFromHelmFiles(helmCharts)
		if err != nil {
			t.Errorf("Error extracting images: %v", err)
		}

		expectedImages := map[string]types.ImageLocation{
			"checkmarx.jfrog.io/ast-docker/containers-worker:b201b1f": {Origin: types.HelmFileOrigin, Path: "containers/templates/containers-worker.yaml"},
			"checkmarx.jfrog.io/ast-docker/image-insights:f4b507b":    {Origin: types.HelmFileOrigin, Path: "containers/templates/image-insights.yaml"},
		}

		checkHelmResult(t, images, expectedImages)
	})
}

func TestExtractImageInfo(t *testing.T) {
	t.Run("ValidYAMLString", func(t *testing.T) {
		yamlString := `---
# Source: containers/templates/image-insights.yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: default
  name: release-name-containers-image-insights
  labels:
    helm.sh/chart: containers-0.0.133
    app.kubernetes.io/name: containers
    app.kubernetes.io/instance: release-name
    app.kubernetes.io/version: "0.0.133"
    app.kubernetes.io/managed-by: Helm
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["get", "create"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["list"]
---
# Source: containers/templates/image-insights.yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: default
  name: release-name-containers-image-insights
  labels:
    helm.sh/chart: containers-0.0.133
    app.kubernetes.io/name: containers
    app.kubernetes.io/instance: release-name
    app.kubernetes.io/version: "0.0.133"
    app.kubernetes.io/managed-by: Helm
subjects:
  - kind: ServiceAccount
    name: release-name-containers-image-insights
    namespace: default
roleRef:
  kind: Role
  name: release-name-containers-image-insights
  apiGroup: rbac.authorization.k8s.io
---
# Source: containers/templates/containers-image-risks.yaml
apiVersion: ast.checkmarx.com/v1
kind: Microservice
metadata:
  name: release-name-containers-containers-image-risks
  labels:
    helm.sh/chart: containers-0.0.133
    app.kubernetes.io/name: containers
    app.kubernetes.io/instance: release-name
    app.kubernetes.io/version: "0.0.133"
    app.kubernetes.io/managed-by: Helm


spec:
  component: "containers"

  image:
    registry: 
    name: nginx
    pullPolicy: IfNotPresent # Overrides the image tag whose default is the chart appVersion.
    tag: latest
    imagePullSecrets: [ ]

  scale:
    static:
      count: 1
    auto:
      enabled: false
      minReplicas: 1
      maxReplicas: 20
      targetCPUUtilizationPercentage: 70
      targetMemoryUtilizationPercentage: 70

  environmentVariablesUnencrypted:
    - key: BucketName
      value: "containers-image-risks-bucket"
    - key: REDISSHAREDS3BUCKET
      value: "containers-image-risks-bucket"
    - key: MaxReconnectAttempts
      value: "10"
    - key: ReconnectDelayInSeconds
      value: "3"
    - key: RabbitMqProtocol
      value: "AMQP"
    - key: RabbitMqSendTimeout
      value: "30000"
    - key: TopicExchangeName
      value: "containers.topic"
    - key: RoutingKey
      value: "containers.scan"
    - key: ActivateImageRisksQueueName
      value: "activate-image-risks"
    - key: FinishedImageRisksQueueName
      value: "finished-image-risks"
    - key: ImageCorrelationsUrl
      value: /image-correlations
    - key: VulnerabilitiesServiceUrl
      value: /vulnerabilities
    - key: ImageInsightsGrpcUrl
      value: http://image-insights:50051
    - key: CacheObjectTimeToLiveInMinutes
      value: "60"
    - key: GrpcPort
      value: 50051

  persistent:
    minio:
      enabled: true
      includeSchema: true
      environmentVariablesMap:
        address:
          - "LocalServerUrl"
        region:
          - "LocalRegion"
        accessKey:
          - "LocalAccessKey"
        accessSecret:
          - "LocalSecretKey"
        tls:
          enabled:
            - "OBJECT_STORE_STORAGE_TLS_ENABLED"
          skipVerify:
            - "OBJECT_STORE_STORAGE_TLS_SKIP_VERIFY"
    redis:
      enabled: true
      environmentVariablesMap:
        isCluster: "REDIS_IS_CLUSTER_MODE"
        address: "RedisAddresses"
        password: "RedisPassword"
        tls:
          enabled: "RedisSSLEnabled"
    postgres:
      enabled: true
      liquibase:
        enabled: false
        definitionsDirInImage: "/app/db"
      environmentVariablesMap:
        read:
          connection_strings:
          - "DATABASE_READ_URL"
        readWrite:
          host: "DatabaseHost"
          port: "DatabasePort"
          db: "DatabaseName"
          username: "DatabaseUser"
          password: "DatabasePassword"
          connection_strings:
          - "DATABASE_WRITE_URL"

  messaging:
    rabbitMQ:
      enabled: true
      environmentVariablesMap:
        tls:
          enabled: ""
          skipVerify: "RABBIT_TLS_SKIP_VERIFY"
        uri: "RabbitMqUrl"

  internalNetworking:
    additionalServiceName: "image-risks"
    ports:
      - port: 80
        name: "rest"
      - port: 50051
        name: "containers-image-risks"

  livenessProbe:
    enabled: true
    type: "httpGet"
    httpGet:
      path: "/health"
      port: 80
  readinessProbe:
    enabled: true
    type: "httpGet"
    httpGet:
      path: "/health"
      port: 80`
		images, err := extractImageInfo(yamlString)
		if err != nil {
			t.Errorf("Error extracting images: %v", err)
		}

		expectedImages := map[string]types.ImageLocation{
			"nginx:latest": {Origin: types.HelmFileOrigin, Path: "containers/templates/containers-image-risks.yaml"},
		}

		checkHelmResult(t, images, expectedImages)
	})

	t.Run("InvalidYAMLString", func(t *testing.T) {
		yamlString := `invalid yaml string`

		_, err := extractImageInfo(yamlString)
		if err == nil {
			t.Errorf("Expected error extracting images from invalid YAML string, but got none")
		}
	})
}

type ExpectedLocation struct {
	File  string
	Line  int
	Start int
	End   int
}

func TestExtractImagesWithLineNumbersFromHelmFiles_Comprehensive(t *testing.T) {
	helmCharts := []types.HelmChartInfo{
		{
			Directory: "../../test_files/imageExtraction/helm-testcases",
			TemplateFiles: []types.FilePath{
				{FullPath: "../../test_files/imageExtraction/helm-testcases/templates/valid-image.yaml", RelativePath: "templates/valid-image.yaml"},
				{FullPath: "../../test_files/imageExtraction/helm-testcases/templates/extra-text-image.yaml", RelativePath: "templates/extra-text-image.yaml"},
				{FullPath: "../../test_files/imageExtraction/helm-testcases/templates/commented-image.yaml", RelativePath: "templates/commented-image.yaml"},
				{FullPath: "../../test_files/imageExtraction/helm-testcases/templates/no-tag-image.yaml", RelativePath: "templates/no-tag-image.yaml"},
				{FullPath: "../../test_files/imageExtraction/helm-testcases/templates/multiple-images.yaml", RelativePath: "templates/multiple-images.yaml"},
				{FullPath: "../../test_files/imageExtraction/helm-testcases/templates/no-images.yaml", RelativePath: "templates/no-images.yaml"},
				{FullPath: "../../test_files/imageExtraction/helm-testcases/templates/invalid.yaml", RelativePath: "templates/invalid.yaml"},
			},
		},
	}

	images, err := ExtractImagesWithLineNumbersFromHelmFiles(helmCharts)
	if err != nil {
		t.Fatalf("Error extracting images: %v", err)
	}

	// Build a map for easy lookup: image name -> []ImageLocation
	found := make(map[string][]types.ImageLocation)
	for _, img := range images {
		found[img.Name] = append(found[img.Name], img.ImageLocations...)
	}

	t.Run("Valid image in template", func(t *testing.T) {
		assertImageFoundWithLocations(t, found, "myrepo/valid:1.0.0", []ExpectedLocation{{"templates/valid-image.yaml", 7, 13, 31}})
	})
	t.Run("Image with extra text and inline comment in template", func(t *testing.T) {
		assertImageFoundWithLocations(t, found, "myrepo/extratext:3.0.0", []ExpectedLocation{
			{"templates/extra-text-image.yaml", 7, 13, 35},
			{"templates/extra-text-image.yaml", 9, 13, 35},
		})
	})
	t.Run("Multiple images in one file", func(t *testing.T) {
		assertImageFoundWithLocations(t, found, "myrepo/first:1.0.0", []ExpectedLocation{{"templates/multiple-images.yaml", 7, 13, 31}})
		assertImageFoundWithLocations(t, found, "myrepo/second:2.0.0", []ExpectedLocation{{"templates/multiple-images.yaml", 9, 13, 32}})
	})
	t.Run("Image in values.yaml", func(t *testing.T) {
		assertImageFoundWithLocations(t, found, "myrepo/valuesimage:1.2.3", []ExpectedLocation{{"values.yaml", 0, 7, 31}})
	})
	t.Run("Image in values-extra.yaml", func(t *testing.T) {
		assertImageFoundWithLocations(t, found, "myrepo/valuesextra:4.5.6", []ExpectedLocation{{"values-extra.yaml", 0, 7, 31}})
	})

	t.Run("Commented-out image in template", func(t *testing.T) {
		assertImageNotFound(t, found, "myrepo/commented:2.0.0")
	})
	t.Run("Image without tag in template", func(t *testing.T) {
		assertImageNotFound(t, found, "myrepo/notag")
	})
	t.Run("No images in file", func(t *testing.T) {
		assertNoImageFromFile(t, found, "templates/no-images.yaml")
	})
	t.Run("Invalid YAML file", func(t *testing.T) {
		assertNoImageFromFile(t, found, "templates/invalid.yaml")
	})
	t.Run("Commented-out image in values.yml", func(t *testing.T) {
		assertImageNotFound(t, found, "myrepo/commented:7.8.9")
	})
}

// Helper: assert that all expected locations for an image are found
func assertImageFoundWithLocations(t *testing.T, found map[string][]types.ImageLocation, imageName string, expected []ExpectedLocation) {
	locs, ok := found[imageName]
	if !ok {
		t.Errorf("Expected image %q, but not found", imageName)
		return
	}
	for _, exp := range expected {
		matched := false
		for _, loc := range locs {
			if loc.Path == exp.File && loc.Line == exp.Line && loc.StartIndex == exp.Start && loc.EndIndex == exp.End {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("Image %q: expected location %+v not found in actual locations: %+v", imageName, exp, locs)
		}
	}
}

// Helper: assert that an image is not found
func assertImageNotFound(t *testing.T, found map[string][]types.ImageLocation, imageName string) {
	if _, ok := found[imageName]; ok {
		t.Errorf("Did not expect image %q to be found, but it was", imageName)
	}
}

// Helper: assert that no image is found from a given file
func assertNoImageFromFile(t *testing.T, found map[string][]types.ImageLocation, file string) {
	for _, locs := range found {
		for _, loc := range locs {
			if loc.Path == file {
				t.Errorf("Did not expect any image to be found in file %q, but found one", file)
			}
		}
	}
}

func checkHelmResult(t *testing.T, images []types.ImageModel, expectedImages map[string]types.ImageLocation) {
	for _, image := range images {
		// Check if the image name exists in the expected images map
		expectedLocation, ok := expectedImages[image.Name]
		if !ok {
			t.Errorf("Unexpected image found: %s", image.Name)
			continue
		}

		// Check if the file path matches the expected file path
		if len(image.ImageLocations) != 1 {
			t.Errorf("Expected image %s to have exactly one location, but got %d", image.Name, len(image.ImageLocations))
			continue
		}

		if image.ImageLocations[0].Path != expectedLocation.Path {
			t.Errorf("Expected image %s to have path %s, but got %s", image.Name, expectedLocation.Path, image.ImageLocations[0].Path)
		}

		if image.ImageLocations[0].Origin != expectedLocation.Origin {
			t.Errorf("Expected image %s to have origin %s, but got %s", image.Name, expectedLocation.Origin, image.ImageLocations[0].Origin)
		}

		// Remove the checked image from the expected images map
		delete(expectedImages, image.Name)
	}

	// Check if any expected images are left unchecked
	for imageName, expectedLocation := range expectedImages {
		t.Errorf("Expected image %s not found (Origin: %s, Path: %s)", imageName, expectedLocation.Origin, expectedLocation.Path)
	}
}
