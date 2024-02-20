package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

const (
	DockerImagesCmd  = "docker images"
	DockerPsCmd      = "docker ps -a"
	DockerHistoryCmd = "docker history %s"
)

var (
	allImages     []*Image
	allContainers []*Container
	RE            = regexp.MustCompile(`\s+\s+`)
)

type Image struct {
	ImageID string
	Name    string
	Size    string
}

type Container struct {
	Image *Image
}

func init() {
	imageInfos := execCommand(DockerImagesCmd)[1:]
	for _, imageInfo := range imageInfos {
		imageInfo = strings.TrimSpace(imageInfo)
		if imageInfo == "" {
			continue
		}
		allImages = append(allImages, NewImage(imageInfo))
	}

	containerInfos := execCommand(DockerPsCmd)[1:]
	for _, containerInfo := range containerInfos {
		containerInfo = strings.TrimSpace(containerInfo)
		if containerInfo == "" {
			continue
		}
		allContainers = append(allContainers, NewContainer(containerInfo))
	}
}

func execCommand(command string) []string {
	result, err := exec.Command("bash", "-c", command).Output()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Split(string(result), "\n")
}

func getImageByID(imageID string) *Image {
	for _, image := range allImages {
		if image.ImageID == imageID {
			return image
		}
	}
	return nil
}

func NewImage(imageInfo string) *Image {
	splitInfo := RE.Split(imageInfo, -1)
	imageID := splitInfo[2]
	size := splitInfo[len(splitInfo)-1]
	image := &Image{
		ImageID: imageID,
		Size:    size,
	}
	image.generateImageName(splitInfo[0], splitInfo[1])
	return image
}

func (image *Image) generateImageName(repo, tag string) {
	if tag == "latest" {
		image.Name = repo
	} else {
		image.Name = repo + ":" + tag
	}
}

func (image *Image) getRelatedImages() []*Image {
	history := execCommand(fmt.Sprintf(DockerHistoryCmd, image.ImageID))
	var imageIDs []string
	for _, line := range history[1:] {
		splitLine := RE.Split(line, -1)
		if splitLine[0] != "<missing>" {
			imageIDs = append(imageIDs, splitLine[0])
		}
	}
	var relatedImages []*Image
	for _, imageID := range imageIDs {
		relatedImage := getImageByID(imageID)
		if relatedImage != nil {
			relatedImages = append(relatedImages, relatedImage)
		}
	}
	return relatedImages
}

func (image *Image) String() string {
	return fmt.Sprintf("id:%s name:%s size:%s", image.ImageID, image.Name, image.Size)
}

func (image *Image) Equals(other *Image) bool {
	return image.ImageID == other.ImageID
}

type ImageUtil struct{}

func (util *ImageUtil) GetImageByName(name string) *Image {
	for _, image := range allImages {
		if image.Name == name {
			return image
		}
	}
	return nil
}

func NewContainer(containerInfo string) *Container {
	splitInfo := RE.Split(containerInfo, -1)
	imageName := splitInfo[1]
	iUtil := ImageUtil{}
	image := iUtil.GetImageByName(imageName)
	container := &Container{
		Image: image,
	}
	return container
}

type ContainerUtil struct{}

func (util *ContainerUtil) GetUsedImages() []*Image {
	var usedImages []*Image
	for _, container := range allContainers {
		if container.Image != nil {
			usedImages = append(usedImages, container.Image)
		}
	}
	var relatedImages []*Image
	for _, usedImage := range usedImages {
		relatedImages = append(relatedImages, usedImage.getRelatedImages()...)
	}
	usedImages = append(usedImages, relatedImages...)
	return usedImages
}

func main() {
	cUtil := ContainerUtil{}
	imagesUsedByContainer := cUtil.GetUsedImages()

	unusedImages := make(map[*Image]bool)
	for _, image := range allImages {
		unusedImages[image] = true
	}

	for _, image := range imagesUsedByContainer {
		delete(unusedImages, image)
	}

	fmt.Println("unused images")
	for image := range unusedImages {
		fmt.Println(image)
	}
}
