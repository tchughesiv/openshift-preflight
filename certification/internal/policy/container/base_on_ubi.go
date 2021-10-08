package container

import (
	"os"
	"path/filepath"
	"strings"

	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

// BasedOnUBICheck evaluates if the provided image is based on the Red Hat Universal Base Image
// by inspecting the contents of the `/etc/os-release` and identifying if the ID is `rhel` and the
// Name value is `Red Hat Enterprise Linux`
type BasedOnUBICheck struct{}

func (p *BasedOnUBICheck) Validate(imgRef certification.ImageReference) (bool, error) {
	labels, err := p.getLabels(imgRef.ImageInfo)
	if err != nil {
		return false, err
	}
	layers, err := imgRef.ImageInfo.Layers()
	if err != nil {
		return false, err
	}

	log.Info("")
	for _, layer := range layers {
		reader, err := layer.Uncompressed()
		if err != nil {
			return false, err
		}
		reader.Read()

		mType, err := layer.MediaType()
		if err != nil {
			return false, err
		}

		log.Info(mType)
	}
	log.Info(labels["com.redhat.component"])
	log.Info("")

	osRelease, err := p.getOsReleaseContents(imgRef.ImageFSPath)
	if err != nil {
		log.Debugf("could not retrieve contents of os-release")
		return false, err
	}

	return p.validate(labels, osRelease)
}

func (p *BasedOnUBICheck) getLabels(image cranev1.Image) (map[string]string, error) {
	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}

	return configFile.Config.Labels, nil
}

func (p *BasedOnUBICheck) getOsReleaseContents(path string) ([]string, error) {
	osrelease, err := os.ReadFile(filepath.Join(path, "etc", "os-release"))
	if err != nil {
		log.Debug("could not open os-release file for reading")
		return nil, err
	}

	return strings.Split(string(osrelease), "\n"), nil
}

func (p *BasedOnUBICheck) validate(labels map[string]string, osRelease []string) (bool, error) {
	var hasRHELID, hasRHELName, hasUbiComponentLabel bool
	for _, value := range osRelease {
		if strings.HasPrefix(value, `ID="rhel"`) {
			hasRHELID = true
		} else if strings.HasPrefix(value, `NAME="Red Hat Enterprise Linux"`) {
			hasRHELName = true
		}
	}

	if component, exists := labels["com.redhat.component"]; exists {
		hasUbiComponentLabel = strings.Contains(component, "ubi")
	}

	if hasRHELID && hasRHELName && hasUbiComponentLabel {
		return true, nil
	}

	return false, nil
}

func (p *BasedOnUBICheck) Name() string {
	return "BasedOnUbi"
}

func (p *BasedOnUBICheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Checking if the container's base image is based upon the Red Hat Universal Base Image (UBI)",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide", // Placeholder
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *BasedOnUBICheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check BasedOnUbi encountered an error. Please review the preflight.log file for more information.",
		Suggestion: "Change the FROM directive in your Dockerfile or Containerfile to FROM registry.access.redhat.com/ubi8/ubi",
	}
}
